package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"myS3/internal/domain"
	"myS3/internal/dto"
	"myS3/internal/httpapi/middleware"
	"myS3/internal/service"
)

type Objects struct {
	svc    *service.ObjectService
	logger *zap.SugaredLogger
}

func NewObjects(svc *service.ObjectService, logger *zap.SugaredLogger) *Objects {
	return &Objects{svc: svc, logger: logger}
}

func (h *Objects) CreateFolder(c *gin.Context) {
	var req dto.CreateFolderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondValidationError(c, err)
		return
	}

	o, err := h.svc.CreateFolder(c.Request.Context(), userID(c), req.ParentID, req.Name)
	if err != nil {
		respondDomainError(c, err)
		return
	}
	respondObject(c, http.StatusCreated, o)
}

func (h *Objects) Update(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		return
	}

	var req dto.UpdateObjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondValidationError(c, err)
		return
	}
	if req.Name == nil && req.ParentID == nil {
		respondError(c, http.StatusBadRequest, "validation_error", "укажите name или parent_id")
		return
	}

	ctx := c.Request.Context()
	if req.Name != nil {
		if err := h.svc.Rename(ctx, userID(c), id, *req.Name); err != nil {
			respondDomainError(c, err)
			return
		}
	}
	if req.ParentID != nil {
		if err := h.svc.Move(ctx, userID(c), id, req.ParentID); err != nil {
			respondDomainError(c, err)
			return
		}
	}

	o, err := h.svc.Get(ctx, userID(c), id)
	if err != nil {
		respondDomainError(c, err)
		return
	}
	respondObject(c, http.StatusOK, o)
}

func (h *Objects) Upload(c *gin.Context) {
	parentID, ok := parseID(c, "id")
	if !ok {
		return
	}

	fileHeader, err := c.FormFile("file")
	if err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			respondError(c, http.StatusRequestEntityTooLarge, "file_too_large", "файл превышает лимит загрузки")
			return
		}
		respondError(c, http.StatusBadRequest, "validation_error", "файл не передан (поле file)")
		return
	}

	f, err := fileHeader.Open()
	if err != nil {
		respondError(c, http.StatusBadRequest, "validation_error", "не удалось открыть файл")
		return
	}
	defer f.Close()

	o, err := h.svc.UploadFile(c.Request.Context(), userID(c), &parentID,
		fileHeader.Filename, fileHeader.Header.Get("Content-Type"), f, fileHeader.Size)
	if err != nil {
		respondDomainError(c, err)
		return
	}
	respondObject(c, http.StatusCreated, o)
}

func (h *Objects) ListRoot(c *gin.Context) {
	limit, offset := parsePagination(c)
	items, total, err := h.svc.ListChildren(c.Request.Context(), userID(c), nil, limit, offset)
	if err != nil {
		respondDomainError(c, err)
		return
	}
	respondItems(c, items, total)
}

func (h *Objects) ListChildren(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	limit, offset := parsePagination(c)
	items, total, err := h.svc.ListChildren(c.Request.Context(), userID(c), &id, limit, offset)
	if err != nil {
		respondDomainError(c, err)
		return
	}
	respondItems(c, items, total)
}

func (h *Objects) Get(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	o, err := h.svc.Get(c.Request.Context(), userID(c), id)
	if err != nil {
		respondDomainError(c, err)
		return
	}
	respondObject(c, http.StatusOK, o)
}

func (h *Objects) Graph(c *gin.Context) {
	var folderID *uuid.UUID
	if raw := c.Query("folder"); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			respondError(c, http.StatusBadRequest, "validation_error", "некорректный folder")
			return
		}
		folderID = &id
	}

	g, err := h.svc.GetGraph(c.Request.Context(), userID(c), folderID)
	if err != nil {
		respondDomainError(c, err)
		return
	}

	nodes := make([]dto.GraphNode, 0, len(g.Nodes))
	for i := range g.Nodes {
		o := &g.Nodes[i]
		nodes = append(nodes, dto.GraphNode{
			ID:         o.ID,
			Name:       o.Name,
			Type:       o.Type,
			Visibility: o.Visibility,
			ParentID:   o.ParentID,
		})
	}
	edges := make([]dto.GraphEdge, 0, len(g.Edges))
	for i := range g.Edges {
		e := &g.Edges[i]
		edges = append(edges, dto.GraphEdge{
			SourceID:   e.SourceID,
			TargetID:   e.TargetID,
			TargetName: e.TargetName,
		})
	}

	inSubtree := make(map[uuid.UUID]bool, len(g.Nodes))
	for i := range g.Nodes {
		inSubtree[g.Nodes[i].ID] = true
	}
	tree := []dto.GraphTreeEdge{}
	for i := range g.Nodes {
		n := &g.Nodes[i]
		if n.ParentID != nil && inSubtree[*n.ParentID] {
			tree = append(tree, dto.GraphTreeEdge{ParentID: n.ParentID, ChildID: n.ID})
		}
	}

	respondJSON(c, http.StatusOK, dto.GraphResponse{Nodes: nodes, Edges: edges, Tree: tree})
}

func (h *Objects) Content(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		return
	}

	rc, o, err := h.svc.Download(c.Request.Context(), userID(c), id)
	if err != nil {
		respondDomainError(c, err)
		return
	}
	defer rc.Close()

	contentType := "application/octet-stream"
	if o.ContentType != nil && *o.ContentType != "" {
		contentType = *o.ContentType
	}
	c.Header("Content-Type", contentType)
	c.Header("Content-Disposition", `attachment; filename="`+o.Name+`"`)
	c.DataFromReader(http.StatusOK, derefInt64(o.SizeBytes), contentType, rc, nil)
}

func (h *Objects) Delete(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	if err := h.svc.Delete(c.Request.Context(), userID(c), id); err != nil {
		h.logger.Errorw("delete object failed", "id", id, "error", err)
		respondDomainError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Objects) SetVisibility(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	var req dto.VisibilityRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondValidationError(c, err)
		return
	}

	o, err := h.svc.SetVisibility(c.Request.Context(), userID(c), id, req.Visibility)
	if err != nil {
		respondDomainError(c, err)
		return
	}
	respondObject(c, http.StatusOK, o)
}

func (h *Objects) ListGrants(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	grants, err := h.svc.ListGrants(c.Request.Context(), userID(c), id)
	if err != nil {
		respondDomainError(c, err)
		return
	}
	out := make([]dto.Grant, 0, len(grants))
	for _, g := range grants {
		out = append(out, dto.Grant{
			GranteeID:  g.GranteeID,
			Permission: g.Permission,
			CreatedAt:  g.CreatedAt,
		})
	}
	respondJSON(c, http.StatusOK, dto.GrantsResponse{Items: out})
}

func (h *Objects) SetGrant(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	var req dto.GrantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondValidationError(c, err)
		return
	}

	g, err := h.svc.SetGrant(c.Request.Context(), userID(c), id, req.GranteeID, req.Permission)
	if err != nil {
		respondDomainError(c, err)
		return
	}
	respondJSON(c, http.StatusCreated, dto.Grant{
		GranteeID:  g.GranteeID,
		Permission: g.Permission,
		CreatedAt:  g.CreatedAt,
	})
}

func (h *Objects) DeleteGrant(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	granteeID, ok := parseID(c, "userId")
	if !ok {
		return
	}
	if err := h.svc.DeleteGrant(c.Request.Context(), userID(c), id, granteeID); err != nil {
		respondDomainError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Objects) PublicContent(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		return
	}

	rc, o, err := h.svc.DownloadPublic(c.Request.Context(), id)
	if err != nil {
		respondDomainError(c, err)
		return
	}
	defer rc.Close()

	contentType := "application/octet-stream"
	if o.ContentType != nil && *o.ContentType != "" {
		contentType = *o.ContentType
	}
	c.Header("Content-Type", contentType)
	c.Header("Content-Disposition", `attachment; filename="`+o.Name+`"`)
	c.DataFromReader(http.StatusOK, derefInt64(o.SizeBytes), contentType, rc, nil)
}

func userID(c *gin.Context) uuid.UUID {
	return c.MustGet(middleware.ContextUserID).(uuid.UUID)
}

func parseID(c *gin.Context, param string) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param(param))
	if err != nil {
		respondError(c, http.StatusBadRequest, "validation_error", "некорректный id")
		return uuid.Nil, false
	}
	return id, true
}

const (
	defaultPageSize = 50
	maxPageSize     = 1000
)

func parsePagination(c *gin.Context) (limit, offset int) {
	limit = defaultPageSize
	if raw := c.Query("limit"); raw != "" {
		if v, err := strconv.Atoi(raw); err == nil && v > 0 {
			limit = min(v, maxPageSize)
		}
	}
	offset = 0
	if raw := c.Query("offset"); raw != "" {
		if v, err := strconv.Atoi(raw); err == nil && v >= 0 {
			offset = v
		}
	}
	return limit, offset
}

func respondObject(c *gin.Context, status int, o *domain.Object) {
	c.JSON(status, dto.Object{
		ID:          o.ID,
		OwnerID:     o.OwnerID,
		ParentID:    o.ParentID,
		Name:        o.Name,
		Type:        o.Type,
		Visibility:  o.Visibility,
		SizeBytes:   o.SizeBytes,
		ContentType: o.ContentType,
		ETag:        o.ETag,
		CreatedAt:   o.CreatedAt,
		UpdatedAt:   o.UpdatedAt,
	})
}

func respondItems(c *gin.Context, items []domain.Object, total int) {
	out := make([]dto.ObjectItem, 0, len(items))
	for i := range items {
		o := &items[i]
		out = append(out, dto.ObjectItem{
			ID:          o.ID,
			ParentID:    o.ParentID,
			Name:        o.Name,
			Type:        o.Type,
			Visibility:  o.Visibility,
			SizeBytes:   o.SizeBytes,
			ContentType: o.ContentType,
			ETag:        o.ETag,
			CreatedAt:   o.CreatedAt,
			UpdatedAt:   o.UpdatedAt,
		})
	}
	c.JSON(http.StatusOK, dto.ItemsResponse{Items: out, Total: total})
}

func derefInt64(v *int64) int64 {
	if v == nil {
		return 0
	}
	return *v
}
