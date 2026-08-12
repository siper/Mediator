package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"stersh.ru/mediator/domain"
)

type SourceHandler struct {
	repo     domain.SourceRepository
	reloader domain.SourceReloader
	tester   domain.SourceTester
}

func NewSourceHandler(repo domain.SourceRepository, reloader domain.SourceReloader, tester domain.SourceTester) *SourceHandler {
	return &SourceHandler{repo: repo, reloader: reloader, tester: tester}
}

func (h *SourceHandler) List(c *gin.Context) {
	items, err := h.repo.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, items)
}

func (h *SourceHandler) GetByID(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	s, err := h.repo.GetByID(id)
	if err != nil {
		if err == domain.ErrSourceNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, s)
}

type upsertSourceRequest struct {
	Type     string            `json:"type" binding:"required"`
	Name     string            `json:"name"`
	Settings map[string]string `json:"settings"`
	Enabled  bool              `json:"enabled"`
	ProxyID  *domain.ID        `json:"ProxyID"`
}

func (h *SourceHandler) Create(c *gin.Context) {
	var req upsertSourceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	s := &domain.Source{
		Type:     req.Type,
		Name:     req.Name,
		Settings: req.Settings,
		Enabled:  req.Enabled,
		ProxyID:  req.ProxyID,
	}
	if s.Settings == nil {
		s.Settings = map[string]string{}
	}
	if err := s.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.repo.Add(s); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	_ = h.reloader.Reload()
	c.JSON(http.StatusCreated, s)
}

func (h *SourceHandler) Update(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var req upsertSourceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	s := &domain.Source{
		Id:       id,
		Type:     req.Type,
		Name:     req.Name,
		Settings: req.Settings,
		Enabled:  req.Enabled,
		ProxyID:  req.ProxyID,
	}
	if s.Settings == nil {
		s.Settings = map[string]string{}
	}
	if err := s.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.repo.Update(s); err != nil {
		if err == domain.ErrSourceNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	_ = h.reloader.Reload()
	c.JSON(http.StatusOK, s)
}

func (h *SourceHandler) SetEnabled(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var req enabledRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	s, err := h.repo.GetByID(id)
	if err != nil {
		if err == domain.ErrSourceNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	s.Enabled = req.Enabled
	if err := h.repo.Update(s); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	_ = h.reloader.Reload()
	c.JSON(http.StatusOK, s)
}

func (h *SourceHandler) Remove(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if err := h.repo.Remove(id); err != nil {
		if err == domain.ErrSourceNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	_ = h.reloader.Reload()
	c.Status(http.StatusNoContent)
}

func (h *SourceHandler) TestSource(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Second)
	defer cancel()
	var req upsertSourceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	s := domain.Source{
		Type:     req.Type,
		Name:     req.Name,
		Settings: req.Settings,
		ProxyID:  req.ProxyID,
	}
	if s.Settings == nil {
		s.Settings = map[string]string{}
	}
	if err := h.tester.Test(ctx, s); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
