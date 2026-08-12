package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"stersh.ru/mediator/domain"
	"stersh.ru/mediator/infrastructure/auth"
)

type UserHandler struct {
	users domain.UserRepository
	jwt   *auth.JWTService
}

func NewUserHandler(users domain.UserRepository, j *auth.JWTService) *UserHandler {
	return &UserHandler{users: users, jwt: j}
}

type createUserRequest struct {
	Name     string `json:"Name"`
	Email    string `json:"Email"`
	Password string `json:"Password"`
	Role     string `json:"Role"`
}

type updateUserRequest struct {
	Name     string `json:"Name"`
	Email    string `json:"Email"`
	Password string `json:"Password"`
	Role     string `json:"Role"`
}

func (h *UserHandler) List(c *gin.Context) {
	page, limit := parsePagination(c)
	users, err := h.users.List(page, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	dtos := make([]userDTO, len(users))
	for i, u := range users {
		dtos[i] = toUserDTO(&u)
	}
	c.JSON(http.StatusOK, dtos)
}

func (h *UserHandler) GetByID(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	u, err := h.users.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, toUserDTO(u))
}

func (h *UserHandler) Create(c *gin.Context) {
	var req createUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Name == "" || req.Email == "" || req.Password == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name, email and password are required"})
		return
	}
	if len(req.Password) < 6 {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrPasswordTooShort.Error()})
		return
	}

	role := domain.RoleUser
	if req.Role == string(domain.RoleAdmin) {
		role = domain.RoleAdmin
	}

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	u := &domain.User{
		Name:         req.Name,
		Email:        req.Email,
		PasswordHash: hash,
		Role:         role,
		CreatedAt:    nowRFC3339(),
	}
	if err := h.users.Add(u); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, toUserDTO(u))
}

func (h *UserHandler) Update(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}

	existing, err := h.users.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	var req updateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Name != "" {
		existing.Name = req.Name
	}
	if req.Email != "" {
		existing.Email = req.Email
	}
	if req.Role == string(domain.RoleAdmin) || req.Role == string(domain.RoleUser) {
		existing.Role = domain.Role(req.Role)
	}
	if req.Password != "" {
		if len(req.Password) < 6 {
			c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrPasswordTooShort.Error()})
			return
		}
		hash, err := auth.HashPassword(req.Password)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		existing.PasswordHash = hash
	}

	if err := h.users.Update(existing); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, toUserDTO(existing))
}

func (h *UserHandler) Delete(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if err := h.users.Remove(id); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

func parsePagination(c *gin.Context) (int, int) {
	page := 1
	limit := 20
	if v := c.Query("page"); v != "" {
		if n, err := parseInt(v); err == nil && n > 0 {
			page = n
		}
	}
	if v := c.Query("limit"); v != "" {
		if n, err := parseInt(v); err == nil && n > 0 {
			limit = n
		}
	}
	return page, limit
}

func parseInt(s string) (int, error) {
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, domain.ErrInvalidMediaType
		}
		n = n*10 + int(c-'0')
	}
	return n, nil
}

func nowRFC3339() string {
	return timeNowUTC()
}
