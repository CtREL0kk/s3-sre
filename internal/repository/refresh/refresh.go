package refresh

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"myS3/internal/domain"
	"myS3/internal/repository/postgres"
)

type RefreshRepo struct {
	pool postgres.PgPool
}

func NewRefreshRepo(pool postgres.PgPool) *RefreshRepo {
	return &RefreshRepo{pool: pool}
}

func (r *RefreshRepo) Create(ctx context.Context, rt *domain.RefreshToken) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO refresh_tokens (id, user_id, token_hash, expires_at) VALUES ($1, $2, $3, $4)`,
		rt.ID, rt.UserID, rt.TokenHash, rt.ExpiresAt,
	)
	return err
}

func (r *RefreshRepo) GetByHash(ctx context.Context, hash string) (*domain.RefreshToken, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, user_id, token_hash, expires_at, revoked_at, created_at
		 FROM refresh_tokens WHERE token_hash = $1`,
		hash,
	)
	var rt domain.RefreshToken
	if err := row.Scan(&rt.ID, &rt.UserID, &rt.TokenHash, &rt.ExpiresAt, &rt.RevokedAt, &rt.CreatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrInvalidToken
		}
		return nil, err
	}
	return &rt, nil
}

func (r *RefreshRepo) Revoke(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE refresh_tokens SET revoked_at = $2 WHERE id = $1 AND revoked_at IS NULL`,
		id, time.Now(),
	)
	return err
}
