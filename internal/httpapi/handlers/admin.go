package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"myS3/internal/service"
)

type Admin struct {
	svc *service.ObjectService
}

func NewAdmin(svc *service.ObjectService) *Admin {
	return &Admin{svc: svc}
}

func (h *Admin) RepairLinks(c *gin.Context) {
	if err := h.svc.RepairLinks(c.Request.Context()); err != nil {
		respondDomainError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Admin) CleanupOrphans(c *gin.Context) {
	if err := h.svc.CleanupOrphans(c.Request.Context()); err != nil {
		respondDomainError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
