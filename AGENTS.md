# AGENTS.md вЂ” Project Conventions

## Service Overview

**media** is a self-hosted media manager in the style of the \*arr stack
(Sonarr / Radarr / Lidarr / Readarr) combined into one service. It catalogs the
user's movies, series, books, and music albums; tracks what is "wanted";
searches for releases (or uses direct/platform sources); downloads them via a
download client or custom grabber; and imports the finished files into a
structured library, updating the catalog with file paths.

It is a **catalog + automation engine**, not a media streamer. The service
stores metadata and file locations; the actual media files live on the user's
filesystem.

## Domain Model

Entities (all in `domain/*.go`):

- `Media` вЂ” a catalog entry (movie / series / book / music album). Fields:
  `Id`, `Name`, `Cover *string`, `Type MediaType`, `LibraryID *ID`,
  `ProviderID string` (e.g. `"tmdb"`, `"author_today"`), `ExternalID string`,
  `Status MediaStatus`, `LastModified string`.
- `PartGroup` вЂ” a grouping inside a Media (a season of a series, a disc of an album).
  Fields: `Id`, `Name`, `Order int`, `MediaId`.
- `Part` вЂ” an individual unit that maps to a file: an episode, a chapter, a
  track, **or the single file of a movie/book**. Fields: `Id`, `Name *string`,
  `GroupOrder *int`, `GroupId *ID`, `MediaId`, `Path *string` (points to the local
  file once imported; nil = wanted/missing), `Monitored bool`.
- `Library` вЂ” `{Id, Name, Path, Type}`; one or more per media type,
  `Media.LibraryID` points to the one that stores the media's files. Replaced
  `RootFolder`. `Path` is an absolute filesystem path.
 - `Source` вЂ” a metadata provider config row in the `sources` table. Fields:
   `Id`, `Type` (`"tmdb"` / `"author_today"`), `Name`, `Settings map[string]string`,
   `Enabled`, `ProxyID *ID` (optional reference to a `Proxy`). Table has
   `UNIQUE(type)` вЂ” only one source per type.
 - `Proxy` вЂ” an outbound HTTP proxy config row in the `proxies` table. Fields:
   `Id`, `Name`, `Type` (`"http"` / `"socks5"`), `Endpoint` (host:port or URL,
   optionally `user:pass@` for SOCKS5), `Enabled`. A source's metadata provider
   routes its requests through the selected `Proxy` (if enabled); when unset the
   provider connects directly.
 - `Indexer` вЂ” a release-search config row in the `indexers` table. Fields:
  `Id`, `Name`, `Type` (`"prowlarr"` / `"jackett"`), `Settings map[string,string`
  (`endpoint`, `api_key`), `Enabled`.
- `DownloadClient` вЂ” a file-acquisition config row in the `download_clients`
  table. Fields: `Id`, `Name`, `Type` (`"qbittorrent"` / `"author_today"`),
  `Settings map[string]string` (`host`/`username`/`password` for qBittorrent,
  `token` for Author.today), `Enabled`.
- `QualityProfile` вЂ” `{Id, Name, Type MediaType, Allowed []Quality, Cutoff Quality}`.
  Bound to one `MediaType`; its `Allowed` set and `Cutoff` are drawn from that
  type's quality catalog.
- `Quality` вЂ” `{Kind QualityKind, Name string}`. Video: resolution (`480p`вЂ¦`2160p`).
  Book: file format (`txt`/`pdf`/`djvu`/`fb2`/`epub`/`mobi`/`azw3`/`docx`).
  Audio: codec (`mp3`/`aac`/`ogg`/`opus`/`alac`/`flac`/`wav`).
- `Release` вЂ” data from indexers: `Title`, `DownloadURL`, `MagnetURI`, `Size int64`,
  `Seeders int`, `Indexer string`, `PublishDate`.
- `ParsedRelease` вЂ” `Quality`, `Source`, `Resolution`, `Codec`, `Year *int`,
  `Season *int`, `Episodes []int`, `Group string`, `IsRepack bool`.
- `QueueItem` вЂ” `{Id, MediaId, PartIds []ID, GrabberName, JobID, DownloadID,
  ReleaseTitle, State GrabState, Progress float64, AddedAt}`.
- `History` вЂ” `{Id, MediaId, PartId *ID, EventType, ReleaseTitle, Data, CreatedAt}`.
- `Task` вЂ” `{Name, Interval, Enabled, LastRun *time.Time, NextRun *time.Time,
  LastStatus, LastError}`. Seeded tasks: `refresh-metadata`, `grab-missing`,
  `check-content`.

Enums:

- `MediaType`: `MediaTypeBook`, `MediaTypeSeries`, `MediaTypeMovie`,
  `MediaTypeMusicAlbum`.
- `MediaStatus`: `MediaStatusContinuing`, `MediaStatusCompleted`.
- `GrabState`: `GrabQueued`, `GrabRunning`, `GrabCompleted`, `GrabFailed`,
  `GrabCanceled`. `Active()` returns true for queued/running.
- `DownloadStatus`: `DownloadQueued`, `Downloading`, `DownloadCompleted`,
  `DownloadSeeding`, `DownloadFailed`. `Active()` returns true for queued/downloading/seeding.
- `HistoryEventType`: `HistoryGrabbed`, `HistoryImported`, `HistoryFailed`.
- `TaskStatus`: `TaskIdle`, `TaskRunning`, `TaskOK`, `TaskError`.
- `QualityKind`: `QualityKindVideo`, `QualityKindBook`, `QualityKindAudio`.

File location is **always on `Part.Path`**, uniformly for all media types.
Movies and books have exactly one Part. Do not introduce `Media.Path` /
`MovieFile`.

## Acquisition Pipeline

The end-to-end loop the service performs:

1. **Add** вЂ” import metadata from a `MediaProvider` (TMDB for movies/series).
   Creating a Media also creates placeholder `Part`s (`Monitored=true`,
   `Path=nil`): one Part for a movie/book, episodes/seasons for a series.
2. **Want** вЂ” a `Part` with `Monitored=true AND Path IS NULL` is "wanted/missing".
3. **Find** вЂ” `ReleaseService` queries active `ReleaseIndexer`es (Torznab via
   Prowlarr/Jackett) and returns `Release`s; `ReleaseParser.Parse(title,
   mediaType)` extracts `ParsedRelease` (quality/source/season/episode/etc.).
4. **Grab** вЂ” `GrabService` routes a `GrabTarget` to a matching `Grabber`:
   - `torrent` в†’ sends `.magnet`/`.torrent` to a `DownloadRunner` (qBittorrent),
     which returns a `DownloadTask`; `torrent.New()` wraps it as a `Grabber`;
   - `http` в†’ fetches a direct URL to staging;
   - `author_today` в†’ `authortoday.NewGrabber(token)` downloads book files via API.
   Records a `QueueItem` + `History{grabbed}`.
5. **Monitor** вЂ” `GrabMonitor` (scheduler) polls active grabs via their grabber's
   `Status`; on `GrabCompleted` it fires `OutputFiles` into `ImportService`.
6. **Import** вЂ” on completion, `ImportService` enumerates output files, matches
   them to the target `Part`(s), applies `NamingService.Build`/`BuildEpisode` +
   `Library` (resolved via `Media.LibraryID`), moves the file, sets `Part.Path`,
   records `History{imported}`.

`ImportService` is **grabber-agnostic** (consumes only `OutputFiles` from
`GrabStatus`). `ReleaseParser` / `QualityProfile` are used only by the torrent
path (quality token in naming). `QualityProfile` is also used by series to
filter which episodes are "wanted".

## Abstractions (ports in `domain/`)

Three DB-backed config tables back the acquisition pipeline вЂ” each has its own
repository + a builder in `main.go` that constructs the runtime port:

```
sources          в†’  MetadataSource  в†’  []MediaProvider        (TMDB, Author.today)
indexers         в†’  IndexerSource    в†’  []ReleaseIndexer      (Prowlarr, Jackett)
download_clients в†’  ClientReloader    в†’  []Grabber             (torrent, http, author.today)
```

**Ports:**

- `MediaProvider` вЂ” metadata source (search, get media). Impl: `TMDB`
  (`infrastructure/tmdb/`), `Author.today` (`infrastructure/authortoday/` as a
  scraper). Planned: OpenLibrary, MusicBrainz. **Metadata only вЂ” never fetches
  files.**
- `ProviderMedia`, `ProviderGroup`, `ProviderPart` вЂ” transient types returned
  by `MediaProvider.GetMedia`, transferred into `Media` + `PartGroup` + `Part`.
- `SearchResult` вЂ” a single provider search result (title, cover, external ID,
  media type).
 - `MetadataSource` вЂ” `Active() []MediaProvider`; backed by the `sources` table.
   `SourceRepository` CRUDs `Source` rows (`{Id, Type, Name, Settings, Enabled,
   ProxyID}`, `UNIQUE(type)` вЂ” one per type). When `ProxyID` points to an enabled
   `Proxy`, the built provider routes outbound HTTP through it.
 - `ProxyRepository` вЂ” CRUDs `Proxy` rows (`{Id, Name, Type, Endpoint, Enabled}`)
   in the `proxies` table; `ListEnabled()` returns the proxies usable by sources.
 - `SourceReloader` вЂ” `Reload() error`; hot-reloads metadata sources after a
   `Source` row (or a referenced `Proxy`) is added/updated/removed. The provider
   builder resolves `Source.ProxyID` against `ProxyRepository` and constructs an
   `httpc.NewClient(timeout, proxy)` for that source.
- `ReleaseIndexer` вЂ” `Name() string; Search(ctx, query, cats) ([]Release, error)`.
  Impl: Torznab client. Backed by the `indexers` table.
- `IndexerSource` вЂ” `Active() []ReleaseIndexer`; backed by `IndexerRepository`.
- `IndexerReloader` вЂ” `Reload() error`; hot-reloads indexers.
- `Release` / `ParsedRelease` вЂ” indexer response + parsed quality/source/season/
  episode tokens.
- `Grabber` вЂ” file acquisition. Methods: `Name()`,
  `Supports(GrabTarget)`, `SupportsProvider(string)`,
  `Submit(GrabTarget) (GrabHandle, error)`,
  `Status(GrabHandle) (GrabStatus, error)`, `Cancel(GrabHandle) error`.
  Impl: `torrent.New()` (wraps a `DownloadRunner`), `direct.New()` (http),
  `authortoday.NewGrabber()` (author.today token). `GrabHandle` identifies a
  submitted job (`{GrabberName, JobID, Name}`); `GrabStatus` returns
  `{State, Progress, OutputFiles, Error, ResolvedID}`.
- `DownloadRunner` вЂ” torrent client interface. Methods: `Name()`,
  `Add(ctx, Release, category) (*DownloadTask, error)`,
  `Get(ctx, id) (*DownloadTask, error)`, `List(ctx) ([]DownloadTask, error)`,
  `Remove(ctx, id, deleteData) error`. `DownloadTask` =
  `{ID, Name, Status, Progress, OutputPath}`; `DownloadStatus` enum
  (`queued`/`downloading`/`completed`/`seeding`/`failed`, `.Active()`).
  Impl: qBittorrent.
- `ClientReloader` вЂ” `Reload() error`; rebuilds grabbers from the
  `download_clients` table and pushes them to `GrabberSink.SetGrabbers`.
- `GrabberSink` вЂ” `SetGrabbers([]Grabber)`; consumed by `GrabService` and
  `GrabMonitor` to swap grabber sets at runtime.
- `CoverStore` вЂ” `Store(ctx, sourceURL) (servedPath, error)`; caches covers
  locally and serves them at `/covers/*`.
- `FileService` вЂ” filesystem operations: `Move(src, dst)`, `Exists(path) bool`,
  `Size(path) (int64, error)`, `Remove(path) error`, `ListFiles(dir) ([]string, error)`.
- `Library` вЂ” `{Id, Name, Path, Type}`; one or more per media type, `Media.LibraryID`
  points to the one that stores the media's files. Replaced `RootFolder`.
  `Path` is an absolute filesystem path.
- `NamingService` (in `application/`) вЂ” `Build(rootPath, mediaName, quality, ext)`
  for movies/books; `BuildEpisode(rootPath, seriesName, season, episode,
  episodeName, quality, ext)` for series. Replaces unsafe filesystem chars.
- `Quality` вЂ” media-type-specific (`QualityKind`: video / book / audio). Video
  в†’ resolution/source (`480p`вЂ¦`2160p`). Book в†’ file format
  (`txt`/`pdf`/`djvu`/`fb2`/`epub`/`mobi`/`azw3`/`docx`). Audio в†’ codec
  (`mp3`/`aac`/`ogg`/`opus`/`alac`/`flac`/`wav`). `Rank()` + `AtLeast(cutoff)`
  drive cutoff logic. `QualityKindFor(MediaType)` maps media type to kind.
- `QualityProfile` вЂ” `{Id, Name, Type MediaType, Allowed []Quality, Cutoff Quality}`.
  Bound to one `MediaType`; its `Allowed` set and `Cutoff` are drawn from that
  type's quality catalog.
- `ReleaseParser.Parse(title, mediaType)` fills `ParsedRelease.Quality` using
  resolution regex (video) or extension/token detection (book/audio).
- `VersionedReader` вЂ” `FetchLastModified(ctx, externalID, mediaType) (string, error)`.
  Optional interface that `MediaProvider` may also implement; used by
  `refresh-metadata` task to detect stale metadata. Impl: Author.today API client.
- `TaskRepository` вЂ” CRUDs `Task` rows (seeded: `refresh-metadata` 24h,
  `grab-missing` 1h, `check-content` 6h).

**Repository interfaces** (all return domain types, `error` last):

| Interface | Entity | Key methods |
|---|---|---|
| `MediaRepository` | `Media` | `GetById`, `Create`, `Update`, `UpdateProviderMeta`, `Remove`, `GetPaged` |
| `PartRepository` | `Part` | `GetById`, `Add`, `GetByMediaId`, `GetWanted`, `Update`, `Remove` |
| `PartGroupRepository` | `PartGroup` | `Add`, `GetByMediaID`, `Remove` |
| `LibraryRepository` | `Library` | `GetById`, `Add`, `List`, `ListByType`, `Update`, `Remove` |
| `ProxyRepository` | `Proxy` | `Add`, `GetByID`, `List`, `ListEnabled`, `Update`, `Remove` |
| `SourceRepository` | `Source` | `Add`, `GetByID`, `List`, `Update`, `Remove` |
| `IndexerRepository` | `Indexer` | `Add`, `GetById`, `List`, `Update`, `Remove` |
| `DownloadClientRepository` | `DownloadClient` | `Add`, `GetById`, `List`, `Update`, `Remove` |
| `QualityProfileRepository` | `QualityProfile` | `Add`, `GetById`, `List`, `ListByType`, `Update`, `Remove` |
| `QueueRepository` | `QueueItem` | `Add`, `GetById`, `List`, `ListActive`, `Update`, `Remove` |
| `HistoryRepository` | `History` | `Add`, `GetByMediaId`, `List` |
| `TaskRepository` | `Task` | `List`, `Get`, `Create`, `Update`, `UpdateRun` |

**Metadata and files are strictly separated:** a `MediaProvider` only returns
metadata; a `Grabber` only returns files. Author.today is wired as **two separate
components** вЂ” `authortoday.Provider` (implements `MediaProvider`, scrapes
metadata, no token needed) and `authortoday.Grabber` (implements `Grabber`,
downloads book files via a bearer token from a `DownloadClient` row). Never one
object doing both.

## Conventions Specific to This Service

- Every media type stores its file on `Part.Path`.
- Transactions: `infrastructure/sqlite.TxManager` is a Unit-of-Work. Repositories
  accept a `DBTX` interface (`*sql.DB` satisfies it); `TxManager.Run(ctx,
  func(*Repos) error)` yields tx-scoped repos. Use it for Import / Grab /
  Refresh atomicity.
- Credentials for grabbers/indexers/clients are stored **plaintext** in the
  `sources`, `indexers`, and `download_clients` tables. Treat the SQLite DB as
  sensitive. User account passwords are bcrypt-hashed in `users.password_hash`.
- JWT signing secret is auto-generated on first start and stored in `settings`
  (`jwt_secret`); it is not configured via environment variables.
- Scheduler jobs (grab monitor, metadata refresh, RSS, cleanup-stuck) run in
  goroutines started from `main`. Grab-monitor interval is fixed at 30s.
- Author.today is wired as **two separate components**: `authortoday.Provider`
  (metadata, registered as a `Source` of type `author_today` in the `sources`
  table) and `authortoday.Grabber` (file downloads, registered as a
  `DownloadClient` of type `author_today` in the `download_clients` table).

## Capabilities

Supported end-to-end: movies and series (TMDB), books (OpenLibrary, Author.today),
music albums (MusicBrainz), Torznab indexers (Prowlarr/Jackett), qBittorrent and
direct/Author.today grabbers, RSS, media requests, and local + OIDC auth.

Not implemented: automatic cutoff upgrades, release blocklist, encryption of
integration credentials at rest.

## Architecture

Clean Architecture (Robert C. Martin):

```
media/
в”њв”Ђв”Ђ domain/           # Entities + Repository interfaces
в”њв”Ђв”Ђ application/      # Use cases / services
в”њв”Ђв”Ђ infrastructure/   # SQLite, download clients, etc.
в”њв”Ђв”Ђ delivery/         # HTTP handlers (Gin)
в”њв”Ђв”Ђ config/           # Configuration loading
в””в”Ђв”Ђ web/              # Frontend (React + Vite + Tailwind v4 + shadcn/ui)
```

**Dependency Rule:** dependencies point inward. `domain/` depends on nothing. `application/` depends on `domain/`. `delivery/` and `infrastructure/` depend on `application/` and `domain/`.

## Frontend (`web/`)

SPA consumed via the single Go binary (`go:embed`). Stack: **React 19 + Vite 7 +
TypeScript 5.7 + Tailwind v4** (shadcn/ui-style primitives), **TanStack Query 5**
(server state), **Zustand 5** (UI state), **React Router 7**, **Radix UI**
(dialog/select), **sonner** (toasts), **lucide-react** (icons),
**class-variance-authority** + **clsx** + **tailwind-merge**.

```
web/src/
в”њв”Ђв”Ђ main.tsx                 # Root: BrowserRouter, QueryClientProvider, ThemeProvider, Toaster, routes
в”њв”Ђв”Ђ index.css                # Tailwind v4 + design tokens (dark-first monochrome)
в”њв”Ђв”Ђ lib/
в”‚   в”њв”Ђв”Ђ cn.ts                # clsx + tailwind-merge helper
в”‚   в””в”Ђв”Ђ api/
в”‚       в”њв”Ђв”Ђ client.ts        # fetch wrapper: api.get/post/put/del; reads {error} from bodies
в”‚       в””в”Ђв”Ђ types.ts         # TS interfaces mirroring Go PascalCase JSON; MediaType + MEDIA_TYPE_LABEL
в”њв”Ђв”Ђ store/ui.ts              # Zustand global UI state (searchOpen, addResult)
в”њв”Ђв”Ђ components/
в”‚   в”њв”Ђв”Ђ app-shell.tsx        # Sidebar + <Outlet/>; renders SearchOverlay + AddMediaDialog
в”‚   в”њв”Ђв”Ђ search-overlay.tsx   # Global search overlay: local Library + Providers (all types)
в”‚   в”њв”Ђв”Ђ add-media-dialog.tsx # Modal: pick library (filtered by type) + import
в”‚   в”њв”Ђв”Ђ release-picker-dialog.tsx # Search releases + grab
в”‚   в”њв”Ђв”Ђ source-dialog.tsx    # Modal: add/edit metadata source; per-source proxy select
в”‚   в”њв”Ђв”Ђ proxy-dialog.tsx     # Modal: add/edit outbound proxy (HTTP, SOCKS5)
в”‚   в”њв”Ђв”Ђ indexer-dialog.tsx   # Modal: add/edit release indexer (Prowlarr, Jackett)
в”‚   в”њв”Ђв”Ђ download-client-dialog.tsx # Modal: add/edit download client
в”‚   в”њв”Ђв”Ђ quality-profile-dialog.tsx # Modal: create quality profile
в”‚   в”њв”Ђв”Ђ settings-layout.tsx  # Settings 2-col shell + SettingsSectionHeader
в”‚   в”њв”Ђв”Ђ settings/            # One file per settings section (nested routes)
в”‚   в”њв”Ђв”Ђ theme-provider.tsx
в”‚   в””в”Ђв”Ђ ui/                  # Primitives: button, badge, dialog, input, select, separator, spinner
в””в”Ђв”Ђ pages/                   # Route pages: movies, series, books, music, wanted, queue, history, вЂ¦
```

Conventions:

- **JSON responses are PascalCase** (Go structs without `json` tags). Some request
  bodies use snake_case field names where handlers bind explicitly вЂ” when adding
  APIs, match existing handlers and mirror types in `lib/api/types.ts`.
- **Server state via TanStack Query; UI state via Zustand.**
- **All API calls go through `lib/api/client.ts`**.
- **Search is global** via `SearchOverlay`.
- **Components are named exports** (no default exports).
- **Settings are URL-driven nested routes** under `/settings`.
- **Dev:** Vite (`:5173`) proxies `/api` and `/covers` to the Go server (`:42800`).
  **Prod:** `npm run build` emits `web/dist`, embedded by the Go binary.

## Code Style

- **No comments in code** вЂ” code should be self-documenting
- **No emoji** in any files
- Exported names use PascalCase (Go convention)
- Unexported names use camelCase
- Error variables use `Err` prefix: `var ErrMediaNotFound = ...`
- Interface names: `MediaRepository`, `PartRepository` (no `I` prefix)
- Constructor functions: `NewMediaService`, `NewSQLiteMediaRepository`

## Domain Layer Rules

- Structs are data-only, no methods that depend on external services
- Validation methods are allowed: `func (m *Media) Validate() error`
- Repository interfaces return domain types only
- Repository methods return `*T` for single results, `[]T` for slices
- Repository methods return `error` as last return value
- All list/get-paged methods must accept pagination parameters (`Page int, Limit int`)
- No dependencies on infrastructure or frameworks

## Testing

- Use `testing` stdlib + `github.com/stretchr/testify`
- Table-driven tests (subtests with `t.Run`)
- Mocks via `testify/mock`
- Test files in same package as the code under test
- Test file naming: `media_test.go`, `part_service_test.go`
- Test function naming: `TestMedia_Validate`, `TestMediaService_Create`

## Database

SQLite via `modernc.org/sqlite` (pure Go, no CGO). Use `database/sql` with `*sql.DB`.
No ORMs. Raw SQL in repository implementations.

Schema is applied by golang-migrate from
`infrastructure/sqlite/migrations/` (single baseline `000001_init`). Do not add
ad-hoc `CREATE TABLE` / column ensure paths outside migrations. Breaking schema
changes for pre-release installs may require deleting `media.db`.

## Commands

```bash
go run .                 # start server (run `npm run build` in web/ first for embed)
go test ./...            # run all tests
go build -o media .      # build binary (embeds web/dist)

# frontend (web/)
npm install
npm run dev              # Vite :5173, proxies /api to :42800
npm run build            # build web/dist (required before go build / docker)

# docker
docker compose up -d --build
```

## Configuration

`config.Load()` reads environment variables (`MEDIATOR_*`). Before reading
them it optionally loads a `.env` file from the working directory via `godotenv`
вЂ” `.env` is gitignored; copy `.env.example` to `.env` and fill in real values.

### Hardcoded paths (not env vars)

- `ConfigDir` = `/config` вЂ” SQLite DB, cover cache, persisted JWT secret
- `StagingDir` = `/downloads` вЂ” download staging
- `DBPath` = `/config/media.db`
- `CoverDir` = `/config/covers`
- OIDC callback = `/auth/oidc/callback` вЂ” absolute redirect URI is built from the
  request host/`X-Forwarded-*` headers (register that full URL in the IdP)

### Environment variables

- `MEDIATOR_PORT` вЂ” HTTP listen address (e.g. `:42800`). Defaults to `:42800`.
- `MEDIATOR_TMDB_LANGUAGE` вЂ” Language code for TMDB API requests (e.g. `en-US`, `ru-RU`).
  Defaults to `en-US`.
- `MEDIATOR_LOG_LEVEL` вЂ” `DEBUG` / `INFO` / `WARN` / `ERROR`. Defaults to `INFO`.
- `MEDIATOR_AUTH_ACCESS_TTL` / `MEDIATOR_AUTH_REFRESH_TTL` вЂ” token lifetimes.
- `MEDIATOR_AUTH_DISABLE_LOGIN` / `MEDIATOR_AUTH_DISABLE_REGISTRATION` вЂ” auth toggles.
- `MEDIATOR_AUTH_OIDC_*` вЂ” optional OIDC provider settings.

Metadata sources, indexers, download clients, and libraries are **not** env
configured вЂ” they live in the DB and are managed via the Settings UI.

Library `Path` is an **absolute** filesystem path. Mount media directories into the
container and set each library to that path (e.g. `/media/movies`).

## Development Environment (Docker Compose)

`compose.yml` runs a single `mediator` service. Indexers and download clients are
external вЂ” configure them in Settings after start.

```bash
cp .env.example .env
docker compose up -d --build
```

Volumes:

- `mediator-config` в†’ `/config` (DB, covers, JWT secret)
- `mediator-downloads` в†’ `/downloads` (staging)

Optional `compose.override.yml` (gitignored) for local mounts and debug logging.

## Git Conventions

- Conventional Commits (`feat`, `fix`, `docs`, `refactor`, `test`, `chore`)
- Pre-commit hook (`.githooks/pre-commit`): `go vet` / `go test` for Go changes,
  `npx tsc --noEmit` for TypeScript changes
- Feature work on dedicated branches / worktrees; merge to `main` via pull request

## Docker Image

Push to `main` builds a multi-arch image (`linux/amd64`, `linux/arm64`) via
GitHub Actions and publishes to `ghcr.io/<owner>/mediator`.

Manual build:

```bash
docker buildx build --platform linux/amd64,linux/arm64 \
  -t ghcr.io/siper/mediator:latest --push .
```
