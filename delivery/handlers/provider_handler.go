package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"stersh.ru/mediator/application"
	"stersh.ru/mediator/domain"
)

type ProviderHandler struct {
	svc      *application.ProviderService
	settings domain.SettingRepository
}

func NewProviderHandler(svc *application.ProviderService, settings domain.SettingRepository) *ProviderHandler {
	return &ProviderHandler{svc: svc, settings: settings}
}

func (h *ProviderHandler) requestsEnabled() bool {
	if h.settings == nil {
		return true
	}
	v, err := h.settings.Get(domain.SettingRequestsEnabled)
	if err != nil {
		return true
	}
	return v == "1" || v == "true"
}

func (h *ProviderHandler) Search(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "query parameter 'q' is required"})
		return
	}

	providerName := c.Query("provider")

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

	page := 1
	if p := c.Query("page"); p != "" {
		v, err := strconv.Atoi(p)
		if err != nil || v < 1 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid page"})
			return
		}
		page = v
	}
	limit := 25
	if l := c.Query("limit"); l != "" {
		v, err := strconv.Atoi(l)
		if err != nil || v < 1 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid limit"})
			return
		}
		limit = v
	}

	results, err := h.svc.Search(query, mediaType, providerName, page, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, results)
}

type importRequest struct {
	Provider         string           `json:"provider" binding:"required"`
	ExternalID       string           `json:"external_id" binding:"required"`
	Type             domain.MediaType `json:"type"`
	LibraryID        domain.ID        `json:"library_id" binding:"required"`
	QualityProfileID *domain.ID       `json:"quality_profile_id"`
	Folder           string           `json:"folder"`
	CoverURL         string           `json:"cover_url"`
}

func (h *ProviderHandler) Import(c *gin.Context) {
	if h.requestsEnabled() {
		u, _ := c.Get("user")
		user, _ := u.(*domain.User)
		if user == nil || !user.IsAdmin() {
			c.JSON(http.StatusForbidden, gin.H{"error": "media requests are enabled; submit a request instead"})
			return
		}
	}
	var req importRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	switch req.Type {
	case domain.MediaTypeBook, domain.MediaTypeSeries, domain.MediaTypeMovie, domain.MediaTypeMusicAlbum:
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid type"})
		return
	}

	media, err := h.svc.Import(c.Request.Context(), req.Provider, req.ExternalID, req.Type, req.LibraryID, req.QualityProfileID, req.Folder, req.CoverURL)
	if err != nil {
		status := http.StatusUnprocessableEntity
		switch err {
		case domain.ErrProfileNotFound:
			status = http.StatusNotFound
		case domain.ErrProfileTypeMismatch:
			status = http.StatusBadRequest
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, media)
}
