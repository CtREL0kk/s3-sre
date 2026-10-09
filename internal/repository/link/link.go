package link

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"myS3/internal/domain"
	"myS3/internal/repository/postgres"
)

type LinkRepo struct {
	pool postgres.PgPool
}

func NewLinkRepo(pool postgres.PgPool) *LinkRepo {
	return &LinkRepo{pool: pool}
}

func (r *LinkRepo) ReplaceForObject(ctx context.Context, sourceID uuid.UUID, links []domain.ObjectLink) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `DELETE FROM object_links WHERE source_id = $1`, sourceID); err != nil {
		return err
	}
	for _, l := range links {
		if _, err := tx.Exec(ctx,
			`INSERT INTO object_links (id, source_id, target_id, target_name) VALUES ($1, $2, $3, $4)`,
			l.ID, sourceID, l.TargetID, l.TargetName,
		); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (r *LinkRepo) ListOutgoing(ctx context.Context, objectID uuid.UUID) ([]domain.ObjectLink, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, source_id, target_id, target_name, created_at FROM object_links WHERE source_id = $1 ORDER BY created_at`,
		objectID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanLinks(rows)
}

func (r *LinkRepo) ListIncoming(ctx context.Context, objectID uuid.UUID) ([]domain.ObjectLink, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, source_id, target_id, target_name, created_at FROM object_links WHERE target_id = $1 ORDER BY created_at`,
		objectID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanLinks(rows)
}

func (r *LinkRepo) LinksWithin(ctx context.Context, ids []uuid.UUID) ([]domain.ObjectLink, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, source_id, target_id, target_name, created_at
		 FROM object_links WHERE target_id IS NOT NULL AND source_id = ANY($1) AND target_id = ANY($1)`,
		ids,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanLinks(rows)
}

func (r *LinkRepo) ResolveDanglingByTarget(ctx context.Context, ownerID uuid.UUID, parentID *uuid.UUID, name string, newObjectID uuid.UUID) error {
	base := name
	if i := strings.IndexByte(name, '.'); i > 0 {
		base = name[:i]
	}
	_, err := r.pool.Exec(ctx,
		`UPDATE object_links SET target_id = $1
		 WHERE target_id IS NULL AND target_name IN ($2, $3)
		   AND source_id IN (SELECT id FROM objects WHERE owner_id = $4 AND parent_id IS NOT DISTINCT FROM $5)`,
		newObjectID, name, base, ownerID, parentID,
	)
	return err
}

func scanLinks(rows pgx.Rows) ([]domain.ObjectLink, error) {
	links := []domain.ObjectLink{}
	for rows.Next() {
		var l domain.ObjectLink
		if err := rows.Scan(&l.ID, &l.SourceID, &l.TargetID, &l.TargetName, &l.CreatedAt); err != nil {
			return nil, err
		}
		links = append(links, l)
	}
	return links, rows.Err()
}
