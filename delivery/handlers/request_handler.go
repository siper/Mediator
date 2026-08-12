package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"stersh.ru/mediator/application"
	"stersh.ru/mediator/domain"
)

type RequestHandler struct {
	svc *application.MediaRequestService
}

func NewRequestHandler(svc *application.MediaRequestService) *RequestHandler {
	return &RequestHandler{svc: svc}
}

func requestUser(c *gin.Context) *domain.User {
	u, ok := c.Get("user")
	if !ok {
		return nil
	}
	user, ok := u.(*domain.User)
	if !ok {
		return nil
	}
	return user
}

type createRequestRequest struct {
	Provider         string           `json:"provider" binding:"required"`
	ExternalID       string           `json:"external_id" binding:"required"`
	Title            string           `json:"title"`
	Cover            string           `json:"cover"`
	Type             domain.MediaType `json:"type"`
	LibraryID        domain.ID        `json:"library_id" binding:"required"`
	QualityProfileID *domain.ID       `json:"quality_profile_id"`
	Folder           string           `json:"folder"`
}

type rejectRequest struct {
	Notes *string `json:"notes"`
}

func (h *RequestHandler) List(c *gin.Context) {
	page, limit := parsePagination(c)
	currentUser := requestUser(c)
	if currentUser == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}
	var items []domain.MediaRequest
	if currentUser.IsAdmin() {
		items, err := h.svc.ListAll(page, limit)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, items)
		return
	}
	items, err := h.svc.ListByUser(currentUser.Id, page, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, items)
}

func (h *RequestHandler) GetByID(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	user := requestUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}
	req, err := h.svc.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	if !user.IsAdmin() && req.UserId != user.Id {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	c.JSON(http.StatusOK, req)
}

func (h *RequestHandler) Create(c *gin.Context) {
	user := requestUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}
	var req createRequestRequest
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
	result, err := h.svc.Create(c.Request.Context(), user.Id, req.Provider, req.ExternalID, req.Title, req.Cover, req.Type, req.LibraryID, req.QualityProfileID, req.Folder)
	if err != nil {
		status := http.StatusUnprocessableEntity
		switch err {
		case domain.ErrMediaAlreadyExists, domain.ErrDuplicateRequest:
			status = http.StatusConflict
		case domain.ErrRequestsDisabled:
			status = http.StatusForbidden
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, result)
}

func (h *RequestHandler) Cancel(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	user := requestUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}
	result, err := h.svc.Cancel(c.Request.Context(), id, user.Id)
	if err != nil {
		switch err {
		case domain.ErrRequestNotFound:
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		case domain.ErrRequestNotAuthorized:
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		}
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *RequestHandler) Approve(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	user := requestUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}
	result, err := h.svc.Approve(c.Request.Context(), id, user.Id)
	if err != nil {
		switch err {
		case domain.ErrRequestNotFound:
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		case domain.ErrRequestNotPending:
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		}
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *RequestHandler) Reject(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	user := requestUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}
	var req rejectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	result, err := h.svc.Reject(c.Request.Context(), id, user.Id, req.Notes)
	if err != nil {
		switch err {
		case domain.ErrRequestNotFound:
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		case domain.ErrRequestNotPending:
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}
	c.JSON(http.StatusOK, result)
}
