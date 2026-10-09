package service

import (
	"bytes"
	"context"
	"errors"
	"io"

	"github.com/google/uuid"

	"myS3/internal/domain"
	"myS3/internal/repository/grant"
	"myS3/internal/repository/link"
	"myS3/internal/repository/object"
	"myS3/internal/storage"
)

const indexLimit = 2 << 20

type ObjectGraph struct {
	Nodes []domain.Object
	Edges []domain.ObjectLink
}

type ObjectService struct {
	objects *object.ObjectRepo
	grants  *grant.GrantRepo
	links   *link.LinkRepo
	s3      *storage.S3
	perms   *PermissionService
	mode    string
}

func NewObjectService(objects *object.ObjectRepo, grants *grant.GrantRepo, links *link.LinkRepo, s3 *storage.S3, perms *PermissionService, mode string) *ObjectService {
	return &ObjectService{objects: objects, grants: grants, links: links, s3: s3, perms: perms, mode: mode}
}

func (s *ObjectService) requireWritable() error {
	if s.mode == "readonly" {
		return domain.ErrReadOnly
	}
	return nil
}

func (s *ObjectService) CreateFolder(ctx context.Context, actorID uuid.UUID, parentID *uuid.UUID, name string) (*domain.Object, error) {
	if err := s.requireWritable(); err != nil {
		return nil, err
	}
	if parentID != nil {
		if err := s.perms.requireWrite(ctx, actorID, *parentID); err != nil {
			return nil, err
		}
	}

	o := &domain.Object{
		ID:         uuid.New(),
		OwnerID:    actorID,
		ParentID:   parentID,
		Name:       name,
		Type:       domain.ObjectTypeFolder,
		Visibility: domain.VisibilityPrivate,
	}
	if err := s.objects.Create(ctx, o); err != nil {
		return nil, err
	}
	return o, nil
}

func (s *ObjectService) UploadFile(ctx context.Context, actorID uuid.UUID, parentID *uuid.UUID, name, contentType string, body io.Reader, size int64) (*domain.Object, error) {
	if err := s.requireWritable(); err != nil {
		return nil, err
	}
	if parentID != nil {
		if err := s.perms.requireWrite(ctx, actorID, *parentID); err != nil {
			return nil, err
		}
	}

	id := uuid.New()
	key := storageKey(actorID, id)

	shouldIndex := isLinkSource(name, contentType) && size > 0 && size <= indexLimit
	reader := io.Reader(body)
	var copyBuf bytes.Buffer
	if shouldIndex {
		data, err := io.ReadAll(body)
		if err != nil {
			return nil, err
		}
		copyBuf.Write(data)
		reader = bytes.NewReader(data)
	}

	etag, err := s.s3.Put(ctx, key, contentType, reader)
	if err != nil {
		return nil, err
	}

	o := &domain.Object{
		ID:          id,
		OwnerID:     actorID,
		ParentID:    parentID,
		Name:        name,
		Type:        domain.ObjectTypeFile,
		Visibility:  domain.VisibilityPrivate,
		StorageKey:  &key,
		SizeBytes:   &size,
		ContentType: &contentType,
		ETag:        &etag,
	}
	if err := s.objects.Create(ctx, o); err != nil {
		_ = s.s3.Delete(ctx, key)
		return nil, err
	}

	_ = s.links.ResolveDanglingByTarget(ctx, actorID, parentID, o.Name, o.ID)

	if shouldIndex {
		_ = s.indexLinks(ctx, o.ID, actorID, parentID, copyBuf.Bytes())
	}
	return o, nil
}

func (s *ObjectService) GetGraph(ctx context.Context, actorID uuid.UUID, folderID *uuid.UUID) (*ObjectGraph, error) {
	var nodes []domain.Object
	var err error

	if folderID != nil {
		if err := s.perms.requireRead(ctx, &actorID, *folderID); err != nil {
			return nil, err
		}
		nodes, err = s.objects.ListSubtree(ctx, *folderID)
	} else {
		nodes, err = s.objects.ListAllByOwner(ctx, actorID)
	}
	if err != nil {
		return nil, err
	}

	ids := make([]uuid.UUID, 0, len(nodes))
	for i := range nodes {
		ids = append(ids, nodes[i].ID)
	}
	edges, err := s.links.LinksWithin(ctx, ids)
	if err != nil {
		return nil, err
	}
	return &ObjectGraph{Nodes: nodes, Edges: edges}, nil
}

func (s *ObjectService) indexLinks(ctx context.Context, sourceID, ownerID uuid.UUID, parentID *uuid.UUID, content []byte) error {
	names := parseWikiLinks(content)
	links := make([]domain.ObjectLink, 0, len(names))
	for _, name := range names {
		targetID := s.resolveLink(ctx, ownerID, parentID, name)
		links = append(links, domain.ObjectLink{
			ID:         uuid.New(),
			SourceID:   sourceID,
			TargetID:   targetID,
			TargetName: name,
		})
	}
	return s.links.ReplaceForObject(ctx, sourceID, links)
}

func (s *ObjectService) resolveLink(ctx context.Context, ownerID uuid.UUID, parentID *uuid.UUID, name string) *uuid.UUID {
	if o, err := s.objects.GetByNameInFolder(ctx, ownerID, parentID, name); err == nil {
		return &o.ID
	}
	if o, err := s.objects.GetByNameAnywhere(ctx, ownerID, name); err == nil {
		return &o.ID
	}
	return nil
}

func (s *ObjectService) Get(ctx context.Context, actorID, id uuid.UUID) (*domain.Object, error) {
	o, err := s.objects.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := s.perms.requireRead(ctx, &actorID, id); err != nil {
		return nil, err
	}
	return o, nil
}

func (s *ObjectService) ListRoot(ctx context.Context, actorID uuid.UUID, limit, offset int) ([]domain.Object, int, error) {
	items, err := s.objects.GetRootObjects(ctx, actorID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	total, err := s.objects.CountRootObjects(ctx, actorID)
	if err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (s *ObjectService) ListChildren(ctx context.Context, actorID uuid.UUID, parentID *uuid.UUID, limit, offset int) ([]domain.Object, int, error) {
	if parentID == nil {
		return s.ListRoot(ctx, actorID, limit, offset)
	}
	if err := s.perms.requireRead(ctx, &actorID, *parentID); err != nil {
		return nil, 0, err
	}

	parent, err := s.objects.GetByID(ctx, *parentID)
	if err != nil {
		return nil, 0, err
	}
	if parent.OwnerID != actorID {
		hasGrant, err := s.grants.HasGrantInChain(ctx, *parentID, actorID, domain.PermissionRead)
		if err != nil {
			return nil, 0, err
		}
		if !hasGrant {
			return s.listChildrenAccessible(ctx, actorID, parentID, limit, offset)
		}
	}

	items, err := s.objects.GetChildren(ctx, parentID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	total, err := s.objects.CountChildren(ctx, parentID)
	if err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (s *ObjectService) listChildrenAccessible(ctx context.Context, actorID uuid.UUID, parentID *uuid.UUID, limit, offset int) ([]domain.Object, int, error) {
	items, err := s.objects.GetChildrenAccessible(ctx, actorID, parentID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	total, err := s.objects.CountChildrenAccessible(ctx, actorID, parentID)
	if err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (s *ObjectService) Download(ctx context.Context, actorID, id uuid.UUID) (io.ReadCloser, *domain.Object, error) {
	o, err := s.objects.GetByID(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	if err := s.perms.requireRead(ctx, &actorID, id); err != nil {
		return nil, nil, err
	}
	return s.openFile(ctx, o)
}

func (s *ObjectService) DownloadPublic(ctx context.Context, id uuid.UUID) (io.ReadCloser, *domain.Object, error) {
	o, err := s.objects.GetByID(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	if err := s.perms.requireRead(ctx, nil, id); err != nil {
		return nil, nil, err
	}
	return s.openFile(ctx, o)
}

func (s *ObjectService) openFile(ctx context.Context, o *domain.Object) (io.ReadCloser, *domain.Object, error) {
	if o.Type != domain.ObjectTypeFile || o.StorageKey == nil {
		return nil, nil, domain.ErrNotFile
	}
	rc, err := s.s3.Get(ctx, *o.StorageKey)
	if err != nil {
		return nil, nil, err
	}
	return rc, o, nil
}

func (s *ObjectService) Rename(ctx context.Context, actorID, id uuid.UUID, newName string) error {
	if err := s.requireWritable(); err != nil {
		return err
	}
	if err := s.perms.requireWrite(ctx, actorID, id); err != nil {
		return err
	}
	return s.objects.UpdateName(ctx, id, newName)
}

func (s *ObjectService) Move(ctx context.Context, actorID, id uuid.UUID, newParentID *uuid.UUID) error {
	if err := s.requireWritable(); err != nil {
		return err
	}
	o, err := s.objects.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if err := s.perms.requireWrite(ctx, actorID, id); err != nil {
		return err
	}

	if newParentID == nil {
		if o.OwnerID != actorID {
			return domain.ErrForbidden
		}
		if o.ParentID == nil {
			return nil
		}
		return s.objects.UpdateParent(ctx, id, nil)
	}

	if *newParentID == o.ID {
		return domain.ErrInvalidMove
	}
	if err := s.perms.requireWrite(ctx, actorID, *newParentID); err != nil {
		return err
	}
	if o.Type == domain.ObjectTypeFolder {
		isAncestor, err := s.objects.IsAncestor(ctx, o.ID, *newParentID)
		if err != nil {
			return err
		}
		if isAncestor {
			return domain.ErrInvalidMove
		}
	}
	return s.objects.UpdateParent(ctx, id, newParentID)
}

func (s *ObjectService) Delete(ctx context.Context, actorID, id uuid.UUID) error {
	if err := s.requireWritable(); err != nil {
		return err
	}
	if err := s.perms.requireManage(ctx, actorID, id); err != nil {
		return err
	}

	keys, err := s.objects.SubtreeStorageKeys(ctx, id)
	if err != nil {
		return err
	}
	if len(keys) > 0 {
		if err := s.s3.DeleteMany(ctx, keys); err != nil {
			return err
		}
	}
	return s.objects.Delete(ctx, id)
}

func (s *ObjectService) SetVisibility(ctx context.Context, actorID, id uuid.UUID, visibility domain.Visibility) (*domain.Object, error) {
	if err := s.requireWritable(); err != nil {
		return nil, err
	}
	if err := s.perms.requireManage(ctx, actorID, id); err != nil {
		return nil, err
	}
	if err := s.objects.SetVisibility(ctx, id, visibility); err != nil {
		return nil, err
	}
	return s.objects.GetByID(ctx, id)
}

func (s *ObjectService) ListGrants(ctx context.Context, actorID, id uuid.UUID) ([]domain.Grant, error) {
	if err := s.perms.requireManage(ctx, actorID, id); err != nil {
		return nil, err
	}
	return s.grants.ListByObject(ctx, id)
}

func (s *ObjectService) SetGrant(ctx context.Context, actorID, id, granteeID uuid.UUID, permission domain.Permission) (*domain.Grant, error) {
	if err := s.requireWritable(); err != nil {
		return nil, err
	}
	if err := s.perms.requireManage(ctx, actorID, id); err != nil {
		return nil, err
	}
	g := &domain.Grant{
		ID:         uuid.New(),
		ObjectID:   id,
		GranteeID:  granteeID,
		Permission: permission,
	}
	if err := s.grants.Upsert(ctx, g); err != nil {
		return nil, err
	}
	return g, nil
}

func (s *ObjectService) DeleteGrant(ctx context.Context, actorID, id, granteeID uuid.UUID) error {
	if err := s.requireWritable(); err != nil {
		return err
	}
	if err := s.perms.requireManage(ctx, actorID, id); err != nil {
		return err
	}
	return s.grants.Delete(ctx, id, granteeID)
}

func (s *ObjectService) RepairLinks(ctx context.Context) error {
	files, err := s.objects.ListAllFiles(ctx)
	if err != nil {
		return err
	}
	for i := range files {
		f := &files[i]
		if f.StorageKey == nil {
			continue
		}
		if !isLinkSource(f.Name, derefString(f.ContentType)) {
			continue
		}
		if f.SizeBytes != nil && *f.SizeBytes > indexLimit {
			continue
		}

		rc, err := s.s3.Get(ctx, *f.StorageKey)
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				continue
			}
			return err
		}
		data, readErr := io.ReadAll(rc)
		rc.Close()
		if readErr != nil {
			return readErr
		}
		if err := s.indexLinks(ctx, f.ID, f.OwnerID, f.ParentID, data); err != nil {
			return err
		}
	}
	return nil
}

func (s *ObjectService) CleanupOrphans(ctx context.Context) error {
	dbKeys, err := s.objects.ListAllStorageKeys(ctx)
	if err != nil {
		return err
	}
	dbSet := make(map[string]bool, len(dbKeys))
	for _, k := range dbKeys {
		dbSet[k] = true
	}

	s3Keys, err := s.s3.ListKeys(ctx)
	if err != nil {
		return err
	}
	s3Set := make(map[string]bool, len(s3Keys))
	var orphansInS3 []string
	for _, k := range s3Keys {
		s3Set[k] = true
		if !dbSet[k] {
			orphansInS3 = append(orphansInS3, k)
		}
	}
	if len(orphansInS3) > 0 {
		if err := s.s3.DeleteMany(ctx, orphansInS3); err != nil {
			return err
		}
	}

	for _, k := range dbKeys {
		if s3Set[k] {
			continue
		}
		obj, err := s.objects.GetByStorageKey(ctx, k)
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				continue
			}
			return err
		}
		if err := s.objects.Delete(ctx, obj.ID); err != nil {
			return err
		}
	}
	return nil
}

func derefString(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}

func storageKey(ownerID, objectID uuid.UUID) string {
	return ownerID.String() + "/" + objectID.String()
}
