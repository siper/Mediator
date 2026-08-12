package handlers

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"stersh.ru/mediator/application"
	"stersh.ru/mediator/domain"
)

type TaskHandler struct {
	sched *application.SchedulerService
	repo  domain.TaskRepository
}

func NewTaskHandler(sched *application.SchedulerService, repo domain.TaskRepository) *TaskHandler {
	return &TaskHandler{sched: sched, repo: repo}
}

func (h *TaskHandler) List(c *gin.Context) {
	tasks, err := h.repo.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, tasks)
}

type updateTaskRequest struct {
	Interval string            `json:"interval" binding:"required"`
	Enabled  bool              `json:"enabled"`
	Settings map[string]string `json:"settings"`
}

func (h *TaskHandler) Update(c *gin.Context) {
	name := c.Param("name")
	var req updateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.sched.SetConfig(name, req.Interval, req.Enabled, req.Settings); err != nil {
		if err == domain.ErrTaskNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	t, err := h.repo.Get(name)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, t)
}

func (h *TaskHandler) Run(c *gin.Context) {
	name := c.Param("name")
	if err := h.sched.Trigger(context.Background(), name); err != nil {
		if err == domain.ErrTaskNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusAccepted, gin.H{})
}
