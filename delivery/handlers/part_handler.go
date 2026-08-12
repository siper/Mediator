package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"stersh.ru/mediator/application"
	"stersh.ru/mediator/domain"
)

type PartHandler struct {
	svc       *application.PartService
	groupRepo domain.PartGroupRepository
}

func NewPartHandler(svc *application.PartService, groupRepo domain.PartGroupRepository) *PartHandler {
	return &PartHandler{svc: svc, groupRepo: groupRepo}
}

type addPartRequest struct {
	Name       *string `json:"name"`
	GroupOrder *int    `json:"group_order"`
	GroupId    *uint64 `json:"group_id"`
	MediaId    uint64  `json:"media_id" binding:"required"`
	Path       *string `json:"path"`
}

func (h *PartHandler) Add(c *gin.Context) {
	var req addPartRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var groupID *domain.ID
	if req.GroupId != nil {
		v := domain.ID(*req.GroupId)
		groupID = &v
	}
	part := &domain.Part{
		Name:       req.Name,
		GroupOrder: req.GroupOrder,
		GroupId:    groupID,
		MediaId:    domain.ID(req.MediaId),
		Path:       req.Path,
	}
	if err := h.svc.Add(part); err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, part)
}

func (h *PartHandler) GetByID(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	part, err := h.svc.GetByID(domain.ID(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, part)
}

func (h *PartHandler) GetByMediaID(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("mediaId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid media id"})
		return
	}
	parts, err := h.svc.GetByMediaID(domain.ID(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, parts)
}

func (h *PartHandler) Groups(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	if h.groupRepo == nil {
		c.JSON(http.StatusOK, []domain.PartGroup{})
		return
	}
	groups, err := h.groupRepo.GetByMediaID(domain.ID(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, groups)
}

func (h *PartHandler) Remove(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	if err := h.svc.Remove(domain.ID(id)); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *PartHandler) Wanted(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	items, err := h.svc.WantedItems(page, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, items)
}

type updatePartRequest struct {
	Name       *string `json:"name"`
	GroupOrder *int    `json:"group_order"`
	GroupId    *uint64 `json:"group_id"`
	Path       *string `json:"path"`
	Monitored  *bool   `json:"monitored"`
}

func (h *PartHandler) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var req updatePartRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	patch := application.PartPatch{
		Name:       req.Name,
		GroupOrder: req.GroupOrder,
		Path:       req.Path,
		Monitored:  req.Monitored,
	}
	if req.GroupId != nil {
		v := domain.ID(*req.GroupId)
		patch.GroupId = &v
	}

	updated, err := h.svc.Update(domain.ID(id), patch)
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, updated)
}
