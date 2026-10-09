package dto

import (
	"time"

	"github.com/google/uuid"

	"myS3/internal/domain"
)

type GrantRequest struct {
	GranteeID  uuid.UUID         `json:"grantee_id" binding:"required"`
	Permission domain.Permission `json:"permission" binding:"required,oneof=read write"`
}

type Grant struct {
	GranteeID  uuid.UUID         `json:"grantee_id"`
	Permission domain.Permission `json:"permission"`
	CreatedAt  time.Time         `json:"created_at"`
}

type GrantsResponse struct {
	Items []Grant `json:"items"`
}
