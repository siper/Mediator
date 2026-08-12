package delivery

import (
	"io/fs"

	"github.com/gin-gonic/gin"
	"stersh.ru/mediator/application"
	"stersh.ru/mediator/delivery/handlers"
	"stersh.ru/mediator/domain"
)

type Deps struct {
	MediaSvc         *application.MediaService
	PartSvc          *application.PartService
	ProviderSvc      *application.ProviderService
	MediaRepo        domain.MediaRepository
	Providers        []domain.MediaProvider
	CoverStore       domain.CoverStore
	CoverDir         string
	ReleaseSvc       *application.ReleaseService
	GrabSvc          *application.GrabService
	GrabMissingSvc   *application.GrabMissingService
	QueueRepo        domain.QueueRepository
	QueueSvc         *application.QueueService
	HistorySvc       *application.HistoryService
	HistoryRepo      domain.HistoryRepository
	IndexerRepo      domain.IndexerRepository
	IndexerReloader  domain.IndexerReloader
	IndexerTester    domain.IndexerTester
	ClientRepo       domain.DownloadClientRepository
	ClientReloader   domain.ClientReloader
	ClientTester     domain.DownloadClientTester
	LibraryRepo      domain.LibraryRepository
	QualityRepo      domain.QualityProfileRepository
	ProxyRepo        domain.ProxyRepository
	TaskRepo         domain.TaskRepository
	SchedulerSvc     *application.SchedulerService
	SourceRepo       domain.SourceRepository
	MetadataReloader domain.SourceReloader
	SourceTester    domain.SourceTester
	GroupRepo       domain.PartGroupRepository
	RequestRepo     domain.MediaRequestRepository
	RequestSvc      *application.MediaRequestService
	RequestHandler  *handlers.RequestHandler
	SettingRepo     domain.SettingRepository
	WebFS            fs.FS

	AuthMiddleware *AuthMiddleware
	AuthHandler    *handlers.AuthHandler
	UserHandler    *handlers.UserHandler
}

func SetupRouter(d Deps) *gin.Engine {
	r := gin.Default()

	mediaH := handlers.NewMediaHandler(d.MediaSvc, d.ProviderSvc, d.QualityRepo, d.GrabMissingSvc)
	partH := handlers.NewPartHandler(d.PartSvc, d.GroupRepo)
	providerH := handlers.NewProviderHandler(d.ProviderSvc, d.SettingRepo)
	releaseH := handlers.NewReleaseHandler(d.ReleaseSvc, d.GrabSvc, d.QualityRepo, d.MediaRepo)
	queueH := handlers.NewQueueHandler(d.QueueSvc, d.HistorySvc, d.HistoryRepo)
	configH := handlers.NewConfigHandler(d.IndexerRepo, d.IndexerReloader, d.IndexerTester, d.ClientRepo, d.ClientReloader, d.ClientTester, d.LibraryRepo, d.QualityRepo, d.ProxyRepo, d.MetadataReloader, d.SettingRepo)
	taskH := handlers.NewTaskHandler(d.SchedulerSvc, d.TaskRepo)
	sourceH := handlers.NewSourceHandler(d.SourceRepo, d.MetadataReloader, d.SourceTester)

	r.Static("/covers", d.CoverDir)

	if d.AuthHandler != nil {
		authGrp := r.Group("/auth")
		{
			authGrp.POST("/register", d.AuthHandler.Register)
			authGrp.POST("/login", d.AuthHandler.Login)
			authGrp.POST("/refresh", d.AuthHandler.Refresh)
			authGrp.POST("/logout", d.AuthHandler.Logout)
			authGrp.GET("/status", d.AuthHandler.AuthStatus)
			authGrp.GET("/oidc/login", d.AuthHandler.OIDCLogin)
		}
		r.GET(handlers.AuthOIDCCallbackPath, d.AuthHandler.OIDCCallback)
		if d.AuthMiddleware != nil {
			r.GET("/auth/me", d.AuthMiddleware.RequireAuth(), d.AuthHandler.Me)
		}
	}

	api := r.Group("/api")

	if d.AuthMiddleware != nil {
		api.Use(d.AuthMiddleware.RequireAuth())
	}
	{
		api.GET("/media", mediaH.GetPaged)
		api.GET("/media/:id", mediaH.GetByID)
		api.PATCH("/media/:id/profile", mediaH.UpdateProfile)
		api.GET("/media/:id/groups", partH.Groups)
		api.POST("/media/:id/refresh", mediaH.Refresh)
		api.POST("/media/:id/grab-missing", mediaH.GrabMissing)
		api.DELETE("/media/:id", mediaH.Remove)

		api.GET("/providers/search", providerH.Search)
		api.POST("/providers/import", providerH.Import)

		api.POST("/parts", partH.Add)
		api.GET("/parts/wanted", partH.Wanted)
		api.GET("/parts/media/:mediaId", partH.GetByMediaID)
		api.GET("/parts/:id", partH.GetByID)
		api.PUT("/parts/:id", partH.Update)
		api.DELETE("/parts/:id", partH.Remove)

		api.POST("/releases/search", releaseH.Search)
		api.POST("/releases/grab", releaseH.Grab)

		api.GET("/queue", queueH.Queue)
		api.GET("/history", queueH.History)

		api.GET("/libraries", configH.ListLibraries)
		api.GET("/quality-profiles", configH.ListQualityProfiles)
		api.GET("/settings/public", configH.GetPublicSettings)

		if d.RequestHandler != nil {
			api.POST("/requests", d.RequestHandler.Create)
			api.GET("/requests", d.RequestHandler.List)
			api.GET("/requests/:id", d.RequestHandler.GetByID)
			api.POST("/requests/:id/cancel", d.RequestHandler.Cancel)
		}
	}

	admin := r.Group("/api")
	if d.AuthMiddleware != nil {
		admin.Use(d.AuthMiddleware.RequireAuth(), d.AuthMiddleware.RequireAdmin())
	}
	{
		admin.POST("/libraries", configH.CreateLibrary)
		admin.PUT("/libraries/:id", configH.UpdateLibrary)
		admin.DELETE("/libraries/:id", configH.DeleteLibrary)

		admin.POST("/quality-profiles", configH.CreateQualityProfile)
		admin.PUT("/quality-profiles/:id", configH.UpdateQualityProfile)
		admin.DELETE("/quality-profiles/:id", configH.DeleteQualityProfile)

		admin.GET("/indexers", configH.ListIndexers)
		admin.POST("/indexers", configH.CreateIndexer)
		admin.PUT("/indexers/:id", configH.UpdateIndexer)
		admin.PATCH("/indexers/:id/enabled", configH.UpdateIndexerEnabled)
		admin.DELETE("/indexers/:id", configH.DeleteIndexer)
		admin.POST("/indexers/test", configH.TestIndexer)

		admin.GET("/download-clients", configH.ListClients)
		admin.POST("/download-clients", configH.CreateClient)
		admin.PUT("/download-clients/:id", configH.UpdateClient)
		admin.PATCH("/download-clients/:id/enabled", configH.UpdateClientEnabled)
		admin.DELETE("/download-clients/:id", configH.DeleteClient)
		admin.POST("/download-clients/test", configH.TestClient)

		admin.GET("/tasks", taskH.List)
		admin.PUT("/tasks/:name", taskH.Update)
		admin.POST("/tasks/:name/run", taskH.Run)

		admin.GET("/sources", sourceH.List)
		admin.POST("/sources", sourceH.Create)
		admin.GET("/sources/:id", sourceH.GetByID)
		admin.PUT("/sources/:id", sourceH.Update)
		admin.POST("/sources/test", sourceH.TestSource)
		admin.PATCH("/sources/:id/enabled", sourceH.SetEnabled)
		admin.DELETE("/sources/:id", sourceH.Remove)

		admin.GET("/proxies", configH.ListProxies)
		admin.POST("/proxies", configH.CreateProxy)
		admin.GET("/proxies/:id", configH.GetProxyByID)
		admin.PUT("/proxies/:id", configH.UpdateProxy)
		admin.PATCH("/proxies/:id/enabled", configH.UpdateProxyEnabled)
		admin.DELETE("/proxies/:id", configH.DeleteProxy)

		if d.RequestHandler != nil {
			admin.POST("/requests/:id/approve", d.RequestHandler.Approve)
			admin.POST("/requests/:id/reject", d.RequestHandler.Reject)
		}
		if d.SettingRepo != nil {
			admin.GET("/settings", configH.ListSettings)
			admin.PUT("/settings/:key", configH.UpdateSetting)
		}
	}

	if d.UserHandler != nil && d.AuthMiddleware != nil {
		adminGrp := r.Group("/api/users")
		adminGrp.Use(d.AuthMiddleware.RequireAuth(), d.AuthMiddleware.RequireAdmin())
		{
			adminGrp.GET("", d.UserHandler.List)
			adminGrp.POST("", d.UserHandler.Create)
			adminGrp.GET("/:id", d.UserHandler.GetByID)
			adminGrp.PUT("/:id", d.UserHandler.Update)
			adminGrp.DELETE("/:id", d.UserHandler.Delete)
		}
	}

	if d.WebFS != nil {
		registerWeb(r, d.WebFS)
	}

	return r
}
