package domain

import (
	"time"

	"github.com/google/uuid"
)

type Permission string

const (
	PermissionRead  Permission = "read"
	PermissionWrite Permission = "write"
)

type Grant struct {
	ID         uuid.UUID
	ObjectID   uuid.UUID
	GranteeID  uuid.UUID
	Permission Permission
	CreatedAt  time.Time
}
