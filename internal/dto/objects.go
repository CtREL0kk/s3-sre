package dto

import (
	"time"

	"github.com/google/uuid"

	"myS3/internal/domain"
)

type CreateFolderRequest struct {
	ParentID *uuid.UUID `json:"parent_id"`
	Name     string     `json:"name" binding:"required,min=1,max=255"`
}

type UpdateObjectRequest struct {
	Name     *string    `json:"name"`
	ParentID *uuid.UUID `json:"parent_id"`
}

type VisibilityRequest struct {
	Visibility domain.Visibility `json:"visibility" binding:"required,oneof=private public"`
}

type Object struct {
	ID          uuid.UUID         `json:"id"`
	OwnerID     uuid.UUID         `json:"owner_id"`
	ParentID    *uuid.UUID        `json:"parent_id"`
	Name        string            `json:"name"`
	Type        domain.ObjectType `json:"type"`
	Visibility  domain.Visibility `json:"visibility"`
	SizeBytes   *int64            `json:"size_bytes"`
	ContentType *string           `json:"content_type"`
	ETag        *string           `json:"etag"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

type ObjectItem struct {
	ID          uuid.UUID         `json:"id"`
	ParentID    *uuid.UUID        `json:"parent_id"`
	Name        string            `json:"name"`
	Type        domain.ObjectType `json:"type"`
	Visibility  domain.Visibility `json:"visibility"`
	SizeBytes   *int64            `json:"size_bytes"`
	ContentType *string           `json:"content_type"`
	ETag        *string           `json:"etag"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

type ItemsResponse struct {
	Items []ObjectItem `json:"items"`
	Total int          `json:"total"`
}
