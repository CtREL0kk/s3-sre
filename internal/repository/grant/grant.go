package grant

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"

	"myS3/internal/domain"
	"myS3/internal/repository/postgres"
)

type GrantRepo struct {
	pool postgres.PgPool
}

func NewGrantRepo(pool postgres.PgPool) *GrantRepo {
	return &GrantRepo{pool: pool}
}

func (r *GrantRepo) Upsert(ctx context.Context, g *domain.Grant) error {
	err := r.pool.QueryRow(ctx,
		`INSERT INTO object_grants (id, object_id, grantee_id, permission) VALUES ($1, $2, $3, $4)
		 ON CONFLICT (object_id, grantee_id) DO UPDATE SET permission = EXCLUDED.permission
		 RETURNING created_at`,
		g.ID, g.ObjectID, g.GranteeID, g.Permission,
	).Scan(&g.CreatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23503" {
			return domain.ErrNotFound
		}
		return err
	}
	return nil
}

func (r *GrantRepo) Delete(ctx context.Context, objectID, granteeID uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`DELETE FROM object_grants WHERE object_id = $1 AND grantee_id = $2`,
		objectID, granteeID,
	)
	return err
}

func (r *GrantRepo) ListByObject(ctx context.Context, objectID uuid.UUID) ([]domain.Grant, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT object_id, grantee_id, permission, created_at FROM object_grants WHERE object_id = $1 ORDER BY created_at`,
		objectID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	grants := []domain.Grant{}
	for rows.Next() {
		var g domain.Grant
		if err := rows.Scan(&g.ObjectID, &g.GranteeID, &g.Permission, &g.CreatedAt); err != nil {
			return nil, err
		}
		grants = append(grants, g)
	}
	return grants, rows.Err()
}

func (r *GrantRepo) HasGrantInChain(ctx context.Context, objectID, granteeID uuid.UUID, required domain.Permission) (bool, error) {
	permFilter := "g.permission IN ('read','write')"
	if required == domain.PermissionWrite {
		permFilter = "g.permission = 'write'"
	}

	query := fmt.Sprintf(`
		WITH RECURSIVE chain AS (
			SELECT id, parent_id FROM objects WHERE id = $1
			UNION ALL
			SELECT o.id, o.parent_id FROM objects o JOIN chain c ON o.id = c.parent_id
		)
		SELECT EXISTS (
			SELECT 1 FROM object_grants g JOIN chain c ON g.object_id = c.id
			WHERE g.grantee_id = $2 AND %s
		)`, permFilter)

	var exists bool
	if err := r.pool.QueryRow(ctx, query, objectID, granteeID).Scan(&exists); err != nil {
		return false, err
	}
	return exists, nil
}
