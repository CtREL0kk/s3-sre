package dto

import (
	"github.com/google/uuid"

	"myS3/internal/domain"
)

type GraphNode struct {
	ID         uuid.UUID         `json:"id"`
	Name       string            `json:"name"`
	Type       domain.ObjectType `json:"type"`
	Visibility domain.Visibility `json:"visibility"`
	ParentID   *uuid.UUID        `json:"parent_id"`
}

type GraphEdge struct {
	SourceID   uuid.UUID  `json:"source"`
	TargetID   *uuid.UUID `json:"target"`
	TargetName string     `json:"target_name"`
}

type GraphTreeEdge struct {
	ParentID *uuid.UUID `json:"parent"`
	ChildID  uuid.UUID  `json:"child"`
}

type GraphResponse struct {
	Nodes []GraphNode     `json:"nodes"`
	Edges []GraphEdge     `json:"edges"`
	Tree  []GraphTreeEdge `json:"tree"`
}
