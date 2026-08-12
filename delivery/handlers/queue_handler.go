package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"stersh.ru/mediator/application"
	"stersh.ru/mediator/domain"
)

type QueueHandler struct {
	queueSvc    *application.QueueService
	historySvc  *application.HistoryService
	historyRepo domain.HistoryRepository
}

func NewQueueHandler(
	queueSvc *application.QueueService,
	historySvc *application.HistoryService,
	historyRepo domain.HistoryRepository,
) *QueueHandler {
	return &QueueHandler{
		queueSvc:    queueSvc,
		historySvc:  historySvc,
		historyRepo: historyRepo,
	}
}

func (h *QueueHandler) Queue(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 50
	}
	items, err := h.queueSvc.ListItems(page, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, items)
}

func (h *QueueHandler) History(c *gin.Context) {
	if mid := c.Query("media_id"); mid != "" {
		id, err := strconv.ParseUint(mid, 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid media_id"})
			return
		}
		items, err := h.historyRepo.GetByMediaId(domain.ID(id))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, items)
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 50
	}
	items, err := h.historySvc.ListItems(page, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, items)
}
