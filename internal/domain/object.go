package domain

import (
	"time"

	"github.com/google/uuid"
)

type ObjectType string

const (
	ObjectTypeFolder ObjectType = "folder"
	ObjectTypeFile   ObjectType = "file"
)

type Visibility string

const (
	VisibilityPrivate Visibility = "private"
	VisibilityPublic  Visibility = "public"
)

type Object struct {
	ID          uuid.UUID
	OwnerID     uuid.UUID
	ParentID    *uuid.UUID
	Name        string
	Type        ObjectType
	Visibility  Visibility
	StorageKey  *string
	SizeBytes   *int64
	ContentType *string
	ETag        *string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
