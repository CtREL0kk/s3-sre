package service

import (
	"context"

	"github.com/google/uuid"

	"myS3/internal/domain"
	"myS3/internal/repository/grant"
	"myS3/internal/repository/object"
)

type PermissionService struct {
	objects *object.ObjectRepo
	grants  *grant.GrantRepo
}

func NewPermissionService(objects *object.ObjectRepo, grants *grant.GrantRepo) *PermissionService {
	return &PermissionService{objects: objects, grants: grants}
}

func (s *PermissionService) CanRead(ctx context.Context, actorID *uuid.UUID, objectID uuid.UUID) (bool, error) {
	if actorID == nil {
		return s.objects.IsPublicInChain(ctx, objectID)
	}

	o, err := s.objects.GetByID(ctx, objectID)
	if err != nil {
		return false, err
	}
	if o.OwnerID == *actorID {
		return true, nil
	}
	public, err := s.objects.IsPublicInChain(ctx, objectID)
	if err != nil {
		return false, err
	}
	if public {
		return true, nil
	}
	return s.grants.HasGrantInChain(ctx, objectID, *actorID, domain.PermissionRead)
}

func (s *PermissionService) CanWrite(ctx context.Context, actorID uuid.UUID, objectID uuid.UUID) (bool, error) {
	o, err := s.objects.GetByID(ctx, objectID)
	if err != nil {
		return false, err
	}
	if o.OwnerID == actorID {
		return true, nil
	}
	return s.grants.HasGrantInChain(ctx, objectID, actorID, domain.PermissionWrite)
}

func (s *PermissionService) CanManage(ctx context.Context, actorID, objectID uuid.UUID) (bool, error) {
	o, err := s.objects.GetByID(ctx, objectID)
	if err != nil {
		return false, err
	}
	return o.OwnerID == actorID, nil
}

func (s *PermissionService) requireRead(ctx context.Context, actorID *uuid.UUID, objectID uuid.UUID) error {
	ok, err := s.CanRead(ctx, actorID, objectID)
	if err != nil {
		return err
	}
	if !ok {
		return domain.ErrForbidden
	}
	return nil
}

func (s *PermissionService) requireWrite(ctx context.Context, actorID, objectID uuid.UUID) error {
	ok, err := s.CanWrite(ctx, actorID, objectID)
	if err != nil {
		return err
	}
	if !ok {
		return domain.ErrForbidden
	}
	return nil
}

func (s *PermissionService) requireManage(ctx context.Context, actorID, objectID uuid.UUID) error {
	ok, err := s.CanManage(ctx, actorID, objectID)
	if err != nil {
		return err
	}
	if !ok {
		return domain.ErrForbidden
	}
	return nil
}
