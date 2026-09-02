package main

import (
	"context"
	"embed"
	"fmt"
	"log"
	"log/slog"
	"os"
	"time"

	"stersh.ru/mediator/application"
	"stersh.ru/mediator/config"
	"stersh.ru/mediator/delivery"
	"stersh.ru/mediator/delivery/handlers"
	"stersh.ru/mediator/domain"
	"stersh.ru/mediator/infrastructure/auth"
	"stersh.ru/mediator/infrastructure/authortoday"
	"stersh.ru/mediator/infrastructure/covercache"
	"stersh.ru/mediator/infrastructure/download"
	"stersh.ru/mediator/infrastructure/download/qbittorrent"
	"stersh.ru/mediator/infrastructure/grabber"
	direct "stersh.ru/mediator/infrastructure/grabber/http"
	"stersh.ru/mediator/infrastructure/grabber/torrent"
	"stersh.ru/mediator/infrastructure/indexer"
	"stersh.ru/mediator/infrastructure/musicbrainz"
	"stersh.ru/mediator/infrastructure/openlibrary"
	"stersh.ru/mediator/infrastructure/source"
	"stersh.ru/mediator/infrastructure/sqlite"
	"stersh.ru/mediator/infrastructure/storage"
	"stersh.ru/mediator/infrastructure/tmdb"
)

//go:embed all:web/dist
var webDist embed.FS

func main() {
	cfg := config.Load()

	var level slog.Level
	if err := level.UnmarshalText([]byte(cfg.LogLevel)); err != nil {
		level = slog.LevelInfo
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level})))

	for _, dir := range []string{cfg.ConfigDir, cfg.StagingDir, cfg.CoverDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			log.Fatalf("failed to create dir %s: %v", dir, err)
		}
	}

	db, err := sqlite.NewDB(cfg.DBPath)
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	if err := sqlite.RunMigrations(db); err != nil {
		log.Fatalf("failed to migrate database: %v", err)
	}

	mediaRepo := sqlite.NewSQLiteMediaRepository(db)
	partRepo := sqlite.NewSQLitePartRepository(db)
	groupRepo := sqlite.NewSQLitePartGroupRepository(db)
	queueRepo := sqlite.NewSQLiteQueueRepository(db)
	historyRepo := sqlite.NewSQLiteHistoryRepository(db)
	indexerRepo := sqlite.NewSQLiteIndexerRepository(db)
	clientRepo := sqlite.NewSQLiteDownloadClientRepository(db)
	libraryRepo := sqlite.NewSQLiteLibraryRepository(db)
	qualityRepo := sqlite.NewSQLiteQualityProfileRepository(db)
	taskRepo := sqlite.NewSQLiteTaskRepository(db)
	sourceRepo := sqlite.NewSQLiteSourceRepository(db)
	proxyRepo := sqlite.NewSQLiteProxyRepository(db)
	userRepo := sqlite.NewSQLiteUserRepository(db)
	sessionRepo := sqlite.NewSQLiteSessionRepository(db)
	oidcIdentRepo := sqlite.NewSQLiteOIDCIdentityRepository(db)
	settingRepo := sqlite.NewSQLiteSettingRepository(db)
	requestRepo := sqlite.NewSQLiteMediaRequestRepository(db)

	txm := sqlite.NewTxManager(db)
	coverStore := covercache.NewFileCoverStore(cfg.CoverDir, "/covers")

	providerBuilder := func(s domain.Source) (domain.MediaProvider, error) {
		var proxy *domain.Proxy
		if s.ProxyID != nil {
			p, err := proxyRepo.GetByID(*s.ProxyID)
			if err == nil && p.Enabled {
				proxy = p
			}
		}
		switch s.Type {
		case string(domain.SourceTMDB):
			lang := s.Get("language")
			if lang == "" {
				lang = cfg.TMDBLanguage
			}
			return tmdb.NewTMDBProvider(s.Get("api_key"), lang, proxy), nil
		case string(domain.SourceAuthorToday):
			return authortoday.NewProvider(authortoday.NewScraper(proxy), nil), nil
		case string(domain.SourceMusicBrainz):
			return musicbrainz.NewProvider(proxy), nil
		case string(domain.SourceOpenLibrary):
			return openlibrary.NewProvider(proxy), nil
		}
		return nil, fmt.Errorf("unknown source type: %s", s.Type)
	}
	providerSource := source.New(sourceRepo, providerBuilder)
	if err := providerSource.Reload(); err != nil {
		slog.Warn("failed to load metadata sources", "err", err)
	}

	indexerSource := indexer.NewSource(indexerRepo, indexer.Build)
	if err := indexerSource.Reload(); err != nil {
		slog.Warn("failed to load indexers", "err", err)
	}
	parser := application.NewReleaseParser()

	osFS := storage.NewOSService()

	mediaSvc := application.NewMediaService(mediaRepo, partRepo, libraryRepo, osFS)
	partSvc := application.NewPartService(partRepo, mediaRepo, groupRepo)
	providerSvc := application.NewProviderService(providerSource, mediaSvc, partSvc, groupRepo, partRepo, coverStore, qualityRepo)

	releaseSvc := application.NewReleaseService(indexerSource, parser)

	buildGrabbers := func() []domain.Grabber {
		next := []domain.Grabber{
			direct.New("http", cfg.StagingDir),
		}
		clients, err := clientRepo.List()
		if err != nil {
			slog.Warn("failed to list download clients", "err", err)
			return next
		}
		for _, c := range clients {
			if !c.Enabled {
				continue
			}
			switch c.Type {
			case domain.DownloadClientQBittorrent:
				runner, err := qbittorrent.New(c.Name, c.Get("host"), c.Get("username"), c.Get("password"))
				if err != nil {
					slog.Error("failed to init qbittorrent client", "name", c.Name, "err", err)
					continue
				}
				next = append(next, torrent.New("torrent", runner, osFS, "media"))
				slog.Info("grabber registered", "name", "torrent")
			case domain.DownloadClientAuthorToday:
				next = append(next, authortoday.NewGrabber(c.Get("token"), mediaRepo, cfg.StagingDir))
				slog.Info("grabber registered", "name", "author_today")
			}
		}
		return next
	}
	initialGrabbers := buildGrabbers()
	for _, g := range initialGrabbers {
		slog.Info("grabber registered", "name", g.Name())
	}

	grabSvc := application.NewGrabService(initialGrabbers, queueRepo, txm)
	queueSvc := application.NewQueueService(queueRepo, mediaRepo, partRepo, groupRepo)
	historySvc := application.NewHistoryService(historyRepo, mediaRepo, partRepo, groupRepo)
	importSvc := application.NewImportService(mediaRepo, partRepo, groupRepo, libraryRepo, txm, osFS, application.NewNamingService(), parser)
	monitor := application.NewGrabMonitor(initialGrabbers, queueRepo, historyRepo)
	monitor.OnCompleted = importSvc.Import
	cleanupStuckSvc := application.NewCleanupStuckService(initialGrabbers, taskRepo, queueRepo, historyRepo)

	clientReloader := grabber.NewClientReloader(buildGrabbers, grabSvc, monitor, cleanupStuckSvc)
	if err := clientReloader.Reload(); err != nil {
		slog.Warn("failed to load download clients", "err", err)
	}

	clientBuilder := func(c domain.DownloadClient) (domain.TestableClient, error) {
		switch c.Type {
		case domain.DownloadClientQBittorrent:
			return qbittorrent.New(c.Name, c.Get("host"), c.Get("username"), c.Get("password"))
		case domain.DownloadClientAuthorToday:
			return authortoday.NewAPIClient(func() string { return c.Get("token") }), nil
		}
		return nil, fmt.Errorf("unknown download client type: %s", c.Type)
	}
	clientTester := download.NewClientTester(clientBuilder)

	appCtx, appCancel := context.WithCancel(context.Background())
	defer appCancel()

	sched := application.NewSchedulerService(taskRepo).WithContext(appCtx)
	if err := sched.Register("refresh-metadata", application.NewRefreshMetadataJob(providerSvc, mediaSvc), "24h"); err != nil {
		log.Fatalf("failed to register refresh-metadata task: %v", err)
	}
	grabMissingSvc := application.NewGrabMissingService(grabSvc, partSvc, mediaSvc, releaseSvc, qualityRepo, groupRepo)
	if err := sched.Register("grab-missing", grabMissingSvc.Run, "1h"); err != nil {
		log.Fatalf("failed to register grab-missing task: %v", err)
	}
	requestSvc := application.NewMediaRequestService(requestRepo, mediaRepo, providerSvc, settingRepo, providerSvc, coverStore)
	if err := sched.Register("check-content", application.NewCheckContentJob(providerSvc), "6h"); err != nil {
		log.Fatalf("failed to register check-content task: %v", err)
	}

	rssSvc := application.NewRssService(grabSvc, mediaRepo, partRepo, groupRepo, qualityRepo, parser, indexerSource)
	if err := sched.Register("rss", rssSvc.Run, "15m"); err != nil {
		log.Fatalf("failed to register rss task: %v", err)
	}
	if err := sched.RegisterTask(domain.Task{
		Name:     domain.TaskCleanupStuck,
		Interval: "10m",
		Enabled:  false,
		Settings: map[string]string{domain.TaskSettingPenalty: "3"},
		Data:     "{}",
	}, cleanupStuckSvc.Run); err != nil {
		log.Fatalf("failed to register cleanup-stuck task: %v", err)
	}

	interval := 30 * time.Second
	go application.NewScheduler(interval, monitor.Tick).Run(appCtx)
	go sched.Run(appCtx)

	jwtSecret, err := auth.LoadOrCreateJWTSecret(settingRepo)
	if err != nil {
		log.Fatalf("failed to load jwt secret: %v", err)
	}

	jwtSvc := auth.NewJWTService(jwtSecret, cfg.AccessTTL)

	var authProvider domain.AuthProvider
	if cfg.OIDCIssuer != "" && cfg.OIDCClientID != "" {
		lazy := auth.NewLazyOIDCProvider(
			cfg.OIDCIssuer, cfg.OIDCClientID, cfg.OIDCClientSecret,
			cfg.OIDCRoleClaim, cfg.OIDCAdminRole,
		)
		authProvider = lazy
		if !lazy.Enabled() {
			slog.Warn("OIDC provider not ready at startup, will retry on demand", "issuer", cfg.OIDCIssuer)
		}
	}

	authSvc := auth.NewAuthService(
		userRepo, sessionRepo, oidcIdentRepo,
		jwtSvc, authProvider,
		cfg.AccessTTL, cfg.RefreshTTL,
		cfg.LoginEnabled, cfg.RegistrationEnabled,
		cfg.OIDCIssuer, cfg.OIDCAdminRole,
	)

	authMW := delivery.NewAuthMiddleware(jwtSvc, userRepo)
	authH := handlers.NewAuthHandler(authSvc, jwtSvc, userRepo, authProvider, cfg.LoginEnabled, cfg.RegistrationEnabled)
	userH := handlers.NewUserHandler(userRepo, jwtSvc)
	reqH := handlers.NewRequestHandler(requestSvc)

	r := delivery.SetupRouter(delivery.Deps{
		MediaSvc:         mediaSvc,
		PartSvc:          partSvc,
		ProviderSvc:      providerSvc,
		MediaRepo:        mediaRepo,
		CoverStore:       coverStore,
		CoverDir:         cfg.CoverDir,
		ReleaseSvc:       releaseSvc,
		GrabSvc:          grabSvc,
		GrabMissingSvc:   grabMissingSvc,
		QueueRepo:        queueRepo,
		QueueSvc:         queueSvc,
		HistorySvc:       historySvc,
		HistoryRepo:      historyRepo,
		IndexerRepo:      indexerRepo,
		IndexerReloader:  indexerSource,
		IndexerTester:    indexerSource,
		ClientRepo:       clientRepo,
		ClientReloader:   clientReloader,
		ClientTester:     clientTester,
		LibraryRepo:      libraryRepo,
		QualityRepo:      qualityRepo,
		ProxyRepo:        proxyRepo,
		TaskRepo:         taskRepo,
		SchedulerSvc:     sched,
		SourceRepo:       sourceRepo,
		MetadataReloader: providerSource,
		SourceTester:     providerSource,
		GroupRepo:        groupRepo,
		WebFS:            webDist,

		RequestRepo:    requestRepo,
		RequestSvc:     requestSvc,
		RequestHandler: reqH,
		SettingRepo:    settingRepo,

		AuthMiddleware: authMW,
		AuthHandler:    authH,
		UserHandler:    userH,
	})
	r.Run(cfg.Port)
}
