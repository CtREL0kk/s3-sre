package domain

import (
	"time"

	"github.com/google/uuid"
)

type ObjectLink struct {
	ID         uuid.UUID
	SourceID   uuid.UUID
	TargetID   *uuid.UUID
	TargetName string
	CreatedAt  time.Time
}
