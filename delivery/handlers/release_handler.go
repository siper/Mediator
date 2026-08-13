package handlers

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"stersh.ru/mediator/application"
	"stersh.ru/mediator/domain"
)

type ReleaseHandler struct {
	releaseSvc  *application.ReleaseService
	grabSvc     *application.GrabService
	qualityRepo domain.QualityProfileRepository
	mediaRepo   domain.MediaRepository
}

func NewReleaseHandler(releaseSvc *application.ReleaseService, grabSvc *application.GrabService, qualityRepo domain.QualityProfileRepository, mediaRepo domain.MediaRepository) *ReleaseHandler {
	return &ReleaseHandler{releaseSvc: releaseSvc, grabSvc: grabSvc, qualityRepo: qualityRepo, mediaRepo: mediaRepo}
}

type searchRequest struct {
	Query            string           `json:"query" binding:"required"`
	MediaID          *uint64          `json:"media_id"`
	Type             domain.MediaType `json:"type"`
	QualityProfileID *uint64          `json:"quality_profile_id"`
	Season           *int             `json:"season"`
	Episode          *int             `json:"episode"`
}

func (h *ReleaseHandler) Search(c *gin.Context) {
	var req searchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var profile *domain.QualityProfile
	if req.QualityProfileID != nil {
		p, err := h.qualityRepo.GetById(domain.ID(*req.QualityProfileID))
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		profile = p
	}

	var target *application.SeriesTarget
	if req.Type == domain.MediaTypeSeries && req.Season != nil {
		target = &application.SeriesTarget{Season: *req.Season, Episode: req.Episode}
	}

	var titles []string
	if req.MediaID != nil {
		media, err := h.mediaRepo.GetById(domain.ID(*req.MediaID))
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		titles = application.MediaMatchTitles(*media)
	}

	results, err := h.releaseSvc.Search(c.Request.Context(), req.Query, req.Type, profile, target, titles...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, results)
}

type grabRequest struct {
	MediaID uint64          `json:"media_id" binding:"required"`
	PartIDs []uint64        `json:"part_ids"`
	Release *domain.Release `json:"release"`
	URL     string          `json:"url"`
}

func (h *ReleaseHandler) Grab(c *gin.Context) {
	var req grabRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	target := domain.GrabTarget{
		MediaID: domain.ID(req.MediaID),
		Release: req.Release,
		URL:     req.URL,
	}
	for _, pid := range req.PartIDs {
		target.PartIds = append(target.PartIds, domain.ID(pid))
	}
	if req.Release == nil && req.URL == "" {
		media, err := h.mediaRepo.GetById(domain.ID(req.MediaID))
		if err != nil {
			slog.Warn("release handler: media not found for grab", "media_id", req.MediaID, "err", err)
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		target.ProviderName = media.ProviderID
		slog.Debug("release handler: grab target resolved", "media_id", req.MediaID, "provider_name", target.ProviderName)
		if target.Release == nil && media.Name != "" {
			target.Release = &domain.Release{Title: media.Name}
		}
	}

	slog.Debug("release handler: grab called", "media_id", req.MediaID, "provider_name", target.ProviderName, "url", target.URL, "release", target.Release)
	item, err := h.grabSvc.Grab(c.Request.Context(), target)
	if err != nil {
		slog.Warn("release handler: grab failed", "media_id", req.MediaID, "provider_name", target.ProviderName, "err", err)
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}
	slog.Info("release handler: grab submitted", "media_id", req.MediaID, "job_id", item.JobID, "grabber", item.GrabberName)
	c.JSON(http.StatusCreated, item)
}
