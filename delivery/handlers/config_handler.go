package handlers

import (
	"context"
	"log/slog"
	"net/http"
	"path/filepath"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"stersh.ru/mediator/domain"
)

type ConfigHandler struct {
	indexerRepo     domain.IndexerRepository
	indexerReloader domain.IndexerReloader
	indexerTester   domain.IndexerTester
	clientRepo      domain.DownloadClientRepository
	clientReloader  domain.ClientReloader
	clientTester    domain.DownloadClientTester
	libraryRepo     domain.LibraryRepository
	qualityRepo     domain.QualityProfileRepository
	proxyRepo       domain.ProxyRepository
	sourceReloader  domain.SourceReloader
	settingRepo     domain.SettingRepository
}

func NewConfigHandler(
	indexerRepo domain.IndexerRepository,
	indexerReloader domain.IndexerReloader,
	indexerTester domain.IndexerTester,
	clientRepo domain.DownloadClientRepository,
	clientReloader domain.ClientReloader,
	clientTester domain.DownloadClientTester,
	libraryRepo domain.LibraryRepository,
	qualityRepo domain.QualityProfileRepository,
	proxyRepo domain.ProxyRepository,
	sourceReloader domain.SourceReloader,
	settingRepo domain.SettingRepository,
) *ConfigHandler {
	return &ConfigHandler{
		indexerRepo:     indexerRepo,
		indexerReloader: indexerReloader,
		indexerTester:   indexerTester,
		clientRepo:      clientRepo,
		clientReloader:  clientReloader,
		clientTester:    clientTester,
		libraryRepo:     libraryRepo,
		qualityRepo:     qualityRepo,
		proxyRepo:       proxyRepo,
		sourceReloader:  sourceReloader,
		settingRepo:     settingRepo,
	}
}

func parseID(c *gin.Context) (domain.ID, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return 0, false
	}
	return domain.ID(id), true
}

type enabledRequest struct {
	Enabled bool `json:"enabled"`
}

func (h *ConfigHandler) UpdateIndexerEnabled(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var req enabledRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ix, err := h.indexerRepo.GetById(id)
	if err != nil {
		if err == domain.ErrIndexerNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ix.Enabled = req.Enabled
	if err := h.indexerRepo.Update(ix); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	h.reloadIndexers()
	c.JSON(http.StatusOK, ix)
}

func (h *ConfigHandler) UpdateClientEnabled(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var req enabledRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	cl, err := h.clientRepo.GetById(id)
	if err != nil {
		if err == domain.ErrDownloadClientNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	cl.Enabled = req.Enabled
	if err := h.clientRepo.Update(cl); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	h.reloadClients()
	c.JSON(http.StatusOK, cl)
}

func (h *ConfigHandler) UpdateProxyEnabled(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var req enabledRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	p, err := h.proxyRepo.GetByID(id)
	if err != nil {
		if err == domain.ErrProxyNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	p.Enabled = req.Enabled
	if err := h.proxyRepo.Update(p); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	h.reloadSources()
	c.JSON(http.StatusOK, p)
}

func normalizeLibraryPath(raw string) (string, error) {
	if raw == "" {
		return "", domain.ErrEmptyPath
	}
	cleaned := filepath.Clean(raw)
	if !filepath.IsAbs(cleaned) {
		return "", domain.ErrPathNotAbsolute
	}
	return cleaned, nil
}

func (h *ConfigHandler) ListIndexers(c *gin.Context) {
	items, err := h.indexerRepo.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, items)
}

func (h *ConfigHandler) CreateIndexer(c *gin.Context) {
	var ix domain.Indexer
	if err := c.ShouldBindJSON(&ix); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if ix.Settings == nil {
		ix.Settings = map[string]string{}
	}
	if err := ix.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.indexerRepo.Add(&ix); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	h.reloadIndexers()
	c.JSON(http.StatusCreated, ix)
}

func (h *ConfigHandler) UpdateIndexer(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var ix domain.Indexer
	if err := c.ShouldBindJSON(&ix); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if ix.Settings == nil {
		ix.Settings = map[string]string{}
	}
	ix.Id = id
	if err := ix.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.indexerRepo.Update(&ix); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	h.reloadIndexers()
	c.JSON(http.StatusOK, ix)
}

func (h *ConfigHandler) DeleteIndexer(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if err := h.indexerRepo.Remove(id); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	h.reloadIndexers()
	c.Status(http.StatusNoContent)
}

func (h *ConfigHandler) reloadIndexers() {
	if h.indexerReloader == nil {
		return
	}
	if err := h.indexerReloader.Reload(); err != nil {
		slog.Warn("failed to reload indexers", "err", err)
	}
}

func (h *ConfigHandler) TestIndexer(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Second)
	defer cancel()
	var ix domain.Indexer
	if err := c.ShouldBindJSON(&ix); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if ix.Type == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "indexer type is required"})
		return
	}
	if ix.Settings == nil {
		ix.Settings = map[string]string{}
	}
	if err := h.indexerTester.Test(ctx, ix); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *ConfigHandler) reloadClients() {
	if h.clientReloader == nil {
		return
	}
	if err := h.clientReloader.Reload(); err != nil {
		slog.Warn("failed to reload download clients", "err", err)
	}
}

func (h *ConfigHandler) reloadSources() {
	if h.sourceReloader == nil {
		return
	}
	if err := h.sourceReloader.Reload(); err != nil {
		slog.Warn("failed to reload sources", "err", err)
	}
}

func (h *ConfigHandler) ListClients(c *gin.Context) {
	items, err := h.clientRepo.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, items)
}

func (h *ConfigHandler) CreateClient(c *gin.Context) {
	var cl domain.DownloadClient
	if err := c.ShouldBindJSON(&cl); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if cl.Settings == nil {
		cl.Settings = map[string]string{}
	}
	if err := cl.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.clientRepo.Add(&cl); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	h.reloadClients()
	c.JSON(http.StatusCreated, cl)
}

func (h *ConfigHandler) UpdateClient(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var cl domain.DownloadClient
	if err := c.ShouldBindJSON(&cl); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if cl.Settings == nil {
		cl.Settings = map[string]string{}
	}
	cl.Id = id
	if err := cl.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.clientRepo.Update(&cl); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	h.reloadClients()
	c.JSON(http.StatusOK, cl)
}

func (h *ConfigHandler) DeleteClient(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if err := h.clientRepo.Remove(id); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	h.reloadClients()
	c.Status(http.StatusNoContent)
}

func (h *ConfigHandler) TestClient(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Second)
	defer cancel()
	var cl domain.DownloadClient
	if err := c.ShouldBindJSON(&cl); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if cl.Type == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "download client type is required"})
		return
	}
	if cl.Settings == nil {
		cl.Settings = map[string]string{}
	}
	if err := h.clientTester.Test(ctx, cl); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *ConfigHandler) ListLibraries(c *gin.Context) {
	items, err := h.libraryRepo.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, items)
}

func (h *ConfigHandler) CreateLibrary(c *gin.Context) {
	var l domain.Library
	if err := c.ShouldBindJSON(&l); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	path, err := normalizeLibraryPath(l.Path)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	l.Path = path
	if err := l.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.libraryRepo.Add(&l); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, l)
}

func (h *ConfigHandler) UpdateLibrary(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var l domain.Library
	if err := c.ShouldBindJSON(&l); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	path, err := normalizeLibraryPath(l.Path)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	l.Path = path
	l.Id = id
	if err := l.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.libraryRepo.Update(&l); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, l)
}

func (h *ConfigHandler) DeleteLibrary(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if err := h.libraryRepo.Remove(id); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *ConfigHandler) ListQualityProfiles(c *gin.Context) {
	items, err := h.qualityRepo.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, items)
}

func (h *ConfigHandler) CreateQualityProfile(c *gin.Context) {
	var qp domain.QualityProfile
	if err := c.ShouldBindJSON(&qp); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := qp.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.qualityRepo.Add(&qp); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
  c.JSON(http.StatusCreated, qp)
}

func (h *ConfigHandler) UpdateQualityProfile(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var qp domain.QualityProfile
	if err := c.ShouldBindJSON(&qp); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	qp.Id = id
	if err := qp.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.qualityRepo.Update(&qp); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, qp)
}

func (h *ConfigHandler) DeleteQualityProfile(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if err := h.qualityRepo.Remove(id); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *ConfigHandler) ListProxies(c *gin.Context) {
	items, err := h.proxyRepo.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, items)
}

func (h *ConfigHandler) CreateProxy(c *gin.Context) {
	var p domain.Proxy
	if err := c.ShouldBindJSON(&p); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := p.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.proxyRepo.Add(&p); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	h.reloadSources()
	c.JSON(http.StatusCreated, p)
}

func (h *ConfigHandler) GetProxyByID(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	p, err := h.proxyRepo.GetByID(id)
	if err != nil {
		if err == domain.ErrProxyNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, p)
}

func (h *ConfigHandler) UpdateProxy(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var p domain.Proxy
	if err := c.ShouldBindJSON(&p); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	p.Id = id
	if err := p.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.proxyRepo.Update(&p); err != nil {
		if err == domain.ErrProxyNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	h.reloadSources()
	c.JSON(http.StatusOK, p)
}

func (h *ConfigHandler) DeleteProxy(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if err := h.proxyRepo.Remove(id); err != nil {
		if err == domain.ErrProxyNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	h.reloadSources()
	c.Status(http.StatusNoContent)
}

type settingValueRequest struct {
	Value string `json:"value" binding:"required"`
}

func (h *ConfigHandler) ListSettings(c *gin.Context) {
	items, err := h.settingRepo.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	out := make([]domain.Setting, 0, len(items))
	for _, s := range items {
		if s.Key == domain.SettingJWTSecret {
			continue
		}
		out = append(out, s)
	}
	c.JSON(http.StatusOK, out)
}

func (h *ConfigHandler) UpdateSetting(c *gin.Context) {
	key := c.Param("key")
	if key == domain.SettingJWTSecret {
		c.JSON(http.StatusForbidden, gin.H{"error": "setting is not writable"})
		return
	}
	var req settingValueRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.settingRepo.Set(key, req.Value); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, domain.Setting{Key: key, Value: req.Value})
}

func (h *ConfigHandler) GetPublicSettings(c *gin.Context) {
	defaults := map[string]string{
		domain.SettingRequestsEnabled: "true",
	}
	result := gin.H{}
	if h.settingRepo == nil {
		for _, key := range domain.PublicSettingKeys {
			result[key] = defaults[key]
		}
		c.JSON(http.StatusOK, result)
		return
	}
	for _, key := range domain.PublicSettingKeys {
		v, err := h.settingRepo.Get(key)
		if err != nil {
			v = defaults[key]
		}
		result[key] = v
	}
	c.JSON(http.StatusOK, result)
}

func (h *ConfigHandler) reloadSettings() {
	if h.settingRepo == nil {
		return
	}
}
