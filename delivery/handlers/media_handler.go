package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"stersh.ru/mediator/application"
	"stersh.ru/mediator/domain"
)

type MediaHandler struct {
	svc            *application.MediaService
	pSvc           *application.ProviderService
	qualityRepo    domain.QualityProfileRepository
	grabMissingSvc *application.GrabMissingService
}

func NewMediaHandler(svc *application.MediaService, pSvc *application.ProviderService, qualityRepo domain.QualityProfileRepository, grabMissingSvc *application.GrabMissingService) *MediaHandler {
	return &MediaHandler{svc: svc, pSvc: pSvc, qualityRepo: qualityRepo, grabMissingSvc: grabMissingSvc}
}

func (h *MediaHandler) GetByID(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	m, err := h.svc.GetByID(domain.ID(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, m)
}

func (h *MediaHandler) Remove(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	deleteFiles := c.Query("delete_files") == "true" || c.Query("delete_files") == "1"
	if err := h.svc.Remove(domain.ID(id), deleteFiles); err != nil {
		if errors.Is(err, domain.ErrMediaNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

type updateProfileRequest struct {
	QualityProfileID *uint64 `json:"quality_profile_id"`
}

func (h *MediaHandler) UpdateProfile(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var req updateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var profileID *domain.ID
	if req.QualityProfileID != nil {
		media, err := h.svc.GetByID(domain.ID(id))
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		qp, err := h.qualityRepo.GetById(domain.ID(*req.QualityProfileID))
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": domain.ErrProfileNotFound.Error()})
			return
		}
		if qp.Type != media.Type {
			c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrProfileTypeMismatch.Error()})
			return
		}
		pid := domain.ID(*req.QualityProfileID)
		profileID = &pid
	}

	if err := h.svc.UpdateProfile(domain.ID(id), profileID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	m, err := h.svc.GetByID(domain.ID(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, m)
}

func (h *MediaHandler) Refresh(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	if err := h.pSvc.RefreshMedia(c.Request.Context(), domain.ID(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	m, err := h.svc.GetByID(domain.ID(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, m)
}

func (h *MediaHandler) GrabMissing(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	count, err := h.grabMissingSvc.ProcessMedia(c.Request.Context(), domain.ID(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"grabbed": count})
}

func (h *MediaHandler) GetPaged(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	var mediaType *domain.MediaType
	if t := c.Query("type"); t != "" {
		v, err := strconv.ParseUint(t, 10, 8)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid type"})
			return
		}
		mt := domain.MediaType(v)
		mediaType = &mt
	}

	result, err := h.svc.GetPaged(page, limit, mediaType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}
