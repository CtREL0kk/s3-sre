package object

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"myS3/internal/domain"
	"myS3/internal/repository/postgres"
)

type ObjectRepo struct {
	pool postgres.PgPool
}

func NewObjectRepo(pool postgres.PgPool) *ObjectRepo {
	return &ObjectRepo{pool: pool}
}

const objectColumns = `id, owner_id, parent_id, name, type, visibility, storage_key,
	size_bytes, content_type, etag, created_at, updated_at`

func (r *ObjectRepo) Create(ctx context.Context, o *domain.Object) error {
	err := r.pool.QueryRow(ctx,
		`INSERT INTO objects (id, owner_id, parent_id, name, type, visibility, storage_key, size_bytes, content_type, etag)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		 RETURNING created_at, updated_at`,
		o.ID, o.OwnerID, o.ParentID, o.Name, o.Type, o.Visibility,
		o.StorageKey, o.SizeBytes, o.ContentType, o.ETag,
	).Scan(&o.CreatedAt, &o.UpdatedAt)
	if err != nil {
		return mapUnique(err)
	}
	return nil
}

func (r *ObjectRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Object, error) {
	row := r.pool.QueryRow(ctx, `SELECT `+objectColumns+` FROM objects WHERE id = $1`, id)
	return scanObject(row)
}

func (r *ObjectRepo) GetChildren(ctx context.Context, parentID *uuid.UUID, limit, offset int) ([]domain.Object, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+objectColumns+` FROM objects
		 WHERE parent_id IS NOT DISTINCT FROM $1
		 ORDER BY type DESC, name ASC
		 LIMIT $2 OFFSET $3`,
		parentID, limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []domain.Object{}
	for rows.Next() {
		var o domain.Object
		if err := rows.Scan(&o.ID, &o.OwnerID, &o.ParentID, &o.Name, &o.Type, &o.Visibility,
			&o.StorageKey, &o.SizeBytes, &o.ContentType, &o.ETag, &o.CreatedAt, &o.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, o)
	}
	return items, rows.Err()
}

func (r *ObjectRepo) CountChildren(ctx context.Context, parentID *uuid.UUID) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM objects WHERE parent_id IS NOT DISTINCT FROM $1`,
		parentID,
	).Scan(&count)
	return count, err
}

func (r *ObjectRepo) GetChildrenAccessible(ctx context.Context, actorID uuid.UUID, parentID *uuid.UUID, limit, offset int) ([]domain.Object, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+objectColumns+` FROM objects
		 WHERE parent_id IS NOT DISTINCT FROM $1
		   AND (
			 owner_id = $2
			 OR visibility = 'public'
			 OR id IN (SELECT object_id FROM object_grants WHERE grantee_id = $2)
		   )
		 ORDER BY type DESC, name ASC
		 LIMIT $3 OFFSET $4`,
		parentID, actorID, limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []domain.Object{}
	for rows.Next() {
		var o domain.Object
		if err := rows.Scan(&o.ID, &o.OwnerID, &o.ParentID, &o.Name, &o.Type, &o.Visibility,
			&o.StorageKey, &o.SizeBytes, &o.ContentType, &o.ETag, &o.CreatedAt, &o.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, o)
	}
	return items, rows.Err()
}

func (r *ObjectRepo) CountChildrenAccessible(ctx context.Context, actorID uuid.UUID, parentID *uuid.UUID) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM objects
		 WHERE parent_id IS NOT DISTINCT FROM $1
		   AND (
			 owner_id = $2
			 OR visibility = 'public'
			 OR id IN (SELECT object_id FROM object_grants WHERE grantee_id = $2)
		   )`,
		parentID, actorID,
	).Scan(&count)
	return count, err
}

func (r *ObjectRepo) GetRootObjects(ctx context.Context, actorID uuid.UUID, limit, offset int) ([]domain.Object, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+objectColumns+` FROM objects
		 WHERE parent_id IS NULL
		   AND (
			 owner_id = $1
			 OR visibility = 'public'
			 OR id IN (SELECT object_id FROM object_grants WHERE grantee_id = $1)
		   )
		 ORDER BY type DESC, name ASC
		 LIMIT $2 OFFSET $3`,
		actorID, limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []domain.Object{}
	for rows.Next() {
		var o domain.Object
		if err := rows.Scan(&o.ID, &o.OwnerID, &o.ParentID, &o.Name, &o.Type, &o.Visibility,
			&o.StorageKey, &o.SizeBytes, &o.ContentType, &o.ETag, &o.CreatedAt, &o.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, o)
	}
	return items, rows.Err()
}

func (r *ObjectRepo) CountRootObjects(ctx context.Context, actorID uuid.UUID) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM objects
		 WHERE parent_id IS NULL
		   AND (
			 owner_id = $1
			 OR visibility = 'public'
			 OR id IN (SELECT object_id FROM object_grants WHERE grantee_id = $1)
		   )`,
		actorID,
	).Scan(&count)
	return count, err
}

func (r *ObjectRepo) UpdateName(ctx context.Context, id uuid.UUID, name string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE objects SET name = $2, updated_at = now() WHERE id = $1`,
		id, name,
	)
	return mapUnique(err)
}

func (r *ObjectRepo) UpdateParent(ctx context.Context, id uuid.UUID, parentID *uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE objects SET parent_id = $2, updated_at = now() WHERE id = $1`,
		id, parentID,
	)
	return mapUnique(err)
}

func (r *ObjectRepo) IsAncestor(ctx context.Context, ancestor, node uuid.UUID) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx,
		`WITH RECURSIVE parents AS (
			SELECT parent_id FROM objects WHERE id = $2
			UNION ALL
			SELECT o.parent_id FROM objects o JOIN parents p ON o.id = p.parent_id
		 )
		 SELECT EXISTS (SELECT 1 FROM parents WHERE parent_id = $1)`,
		ancestor, node,
	).Scan(&exists)
	return exists, err
}

func (r *ObjectRepo) SubtreeStorageKeys(ctx context.Context, rootID uuid.UUID) ([]string, error) {
	rows, err := r.pool.Query(ctx,
		`WITH RECURSIVE subtree AS (
			SELECT id, storage_key FROM objects WHERE id = $1
			UNION ALL
			SELECT o.id, o.storage_key FROM objects o JOIN subtree s ON o.parent_id = s.id
		 )
		 SELECT storage_key FROM subtree WHERE storage_key IS NOT NULL`,
		rootID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	keys := []string{}
	for rows.Next() {
		var k string
		if err := rows.Scan(&k); err != nil {
			return nil, err
		}
		keys = append(keys, k)
	}
	return keys, rows.Err()
}

func (r *ObjectRepo) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM objects WHERE id = $1`, id)
	return err
}

func (r *ObjectRepo) SetVisibility(ctx context.Context, id uuid.UUID, visibility domain.Visibility) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE objects SET visibility = $2, updated_at = now() WHERE id = $1`,
		id, visibility,
	)
	return err
}

func (r *ObjectRepo) GetByNameInFolder(ctx context.Context, ownerID uuid.UUID, parentID *uuid.UUID, name string) (*domain.Object, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT `+objectColumns+` FROM objects
		 WHERE owner_id = $1 AND parent_id IS NOT DISTINCT FROM $2
		   AND (name = $3 OR split_part(name, '.', 1) = $3)
		 LIMIT 1`,
		ownerID, parentID, name,
	)
	return scanObject(row)
}

func (r *ObjectRepo) GetByNameAnywhere(ctx context.Context, ownerID uuid.UUID, name string) (*domain.Object, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT `+objectColumns+` FROM objects
		 WHERE owner_id = $1 AND (name = $2 OR split_part(name, '.', 1) = $2) LIMIT 1`,
		ownerID, name,
	)
	return scanObject(row)
}

func (r *ObjectRepo) ListSubtree(ctx context.Context, rootID uuid.UUID) ([]domain.Object, error) {
	rows, err := r.pool.Query(ctx,
		`WITH RECURSIVE subtree AS (
			SELECT id FROM objects WHERE id = $1
			UNION ALL
			SELECT o.id FROM objects o JOIN subtree s ON o.parent_id = s.id
		 )
		 SELECT `+objectColumns+` FROM objects WHERE id IN (SELECT id FROM subtree)`,
		rootID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanObjects(rows)
}

func (r *ObjectRepo) ListAllByOwner(ctx context.Context, ownerID uuid.UUID) ([]domain.Object, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+objectColumns+` FROM objects WHERE owner_id = $1`,
		ownerID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanObjects(rows)
}

func (r *ObjectRepo) ListAllFiles(ctx context.Context) ([]domain.Object, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+objectColumns+` FROM objects WHERE type = 'file'`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanObjects(rows)
}

func (r *ObjectRepo) ListAllStorageKeys(ctx context.Context) ([]string, error) {
	rows, err := r.pool.Query(ctx, `SELECT storage_key FROM objects WHERE storage_key IS NOT NULL`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	keys := []string{}
	for rows.Next() {
		var k string
		if err := rows.Scan(&k); err != nil {
			return nil, err
		}
		keys = append(keys, k)
	}
	return keys, rows.Err()
}

func (r *ObjectRepo) GetByStorageKey(ctx context.Context, key string) (*domain.Object, error) {
	row := r.pool.QueryRow(ctx, `SELECT `+objectColumns+` FROM objects WHERE storage_key = $1`, key)
	return scanObject(row)
}

func scanObjects(rows pgx.Rows) ([]domain.Object, error) {
	items := []domain.Object{}
	for rows.Next() {
		var o domain.Object
		if err := rows.Scan(&o.ID, &o.OwnerID, &o.ParentID, &o.Name, &o.Type, &o.Visibility,
			&o.StorageKey, &o.SizeBytes, &o.ContentType, &o.ETag, &o.CreatedAt, &o.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, o)
	}
	return items, rows.Err()
}

func (r *ObjectRepo) IsPublicInChain(ctx context.Context, objectID uuid.UUID) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx,
		`WITH RECURSIVE chain AS (
			SELECT id, parent_id, visibility FROM objects WHERE id = $1
			UNION ALL
			SELECT o.id, o.parent_id, o.visibility FROM objects o JOIN chain c ON o.id = c.parent_id
		 )
		 SELECT EXISTS (SELECT 1 FROM chain WHERE visibility = 'public')`,
		objectID,
	).Scan(&exists)
	return exists, err
}

func mapUnique(err error) error {
	if err == nil {
		return nil
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return domain.ErrNameConflict
	}
	return err
}

func scanObject(row pgx.Row) (*domain.Object, error) {
	var o domain.Object
	if err := row.Scan(&o.ID, &o.OwnerID, &o.ParentID, &o.Name, &o.Type, &o.Visibility,
		&o.StorageKey, &o.SizeBytes, &o.ContentType, &o.ETag, &o.CreatedAt, &o.UpdatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &o, nil
}
