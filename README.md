> [!WARNING]
> **Test version.** Mediator is an early preview and is **not production-ready**.
> The database schema, APIs, and behavior may change without notice.
> Do not run this on data you cannot afford to lose.

# Mediator

Self-hosted media manager in the style of the *arr stack: catalog movies, series,
books, and music; find releases; download them; import files into your library.

Mediator stores metadata and file paths. It does not stream media.

## Quick start

```bash
cp .env.example .env
docker compose up -d --build
```

Set `PUID` and `PGID` in `.env` to the uid/gid of the user that owns mounted
data (default `1000`). The app runs as that user and does not change file
ownership on the volumes.

Open http://localhost:42800, register the first user (becomes admin), then in
Settings add:

- a metadata source (e.g. TMDB)
- an indexer (Prowlarr or Jackett) if you use torrents
- a download client (qBittorrent, Author.today, …)
- one or more libraries with **absolute** paths (mount host folders into the
  container, e.g. `/media/movies`)

Container image (built on push to `main`):

```text
ghcr.io/siper/mediator:latest
```

## Development

```bash
cd web && npm install && npm run build && cd ..
go run .
```

Frontend hot reload: `npm run dev` in `web/` (proxies `/api` to `:42800`).

Tests: `go test ./...`

See [AGENTS.md](AGENTS.md) for architecture and contributor conventions.

## Notes

- SQLite lives at `/config/media.db`. Wipe that file if you need a clean database
  after a breaking schema reset.
- Integration credentials (API keys, qBittorrent password, etc.) are stored
  plaintext in the DB; user passwords are bcrypt-hashed.
- JWT signing secret is generated automatically and stored in the DB.
