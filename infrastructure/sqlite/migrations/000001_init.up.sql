CREATE TABLE IF NOT EXISTS libraries (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	name TEXT NOT NULL DEFAULT '',
	path TEXT NOT NULL,
	type INTEGER NOT NULL,
	settings TEXT NOT NULL DEFAULT '{}'
);

CREATE TABLE IF NOT EXISTS quality_profiles (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	name TEXT NOT NULL,
	type INTEGER NOT NULL,
	allowed TEXT NOT NULL DEFAULT '[]',
	cutoff TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS proxies (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	name TEXT NOT NULL,
	type TEXT NOT NULL DEFAULT 'http',
	endpoint TEXT NOT NULL,
	enabled INTEGER NOT NULL DEFAULT 1
);

CREATE TABLE IF NOT EXISTS media (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	name TEXT NOT NULL,
	original_name TEXT NOT NULL DEFAULT '',
	folder TEXT,
	cover TEXT,
	type INTEGER NOT NULL,
	library_id INTEGER REFERENCES libraries(id) ON DELETE SET NULL,
	provider_id TEXT NOT NULL DEFAULT '',
	external_id TEXT NOT NULL DEFAULT '',
	status TEXT NOT NULL DEFAULT 'continuing',
	last_modified TEXT NOT NULL DEFAULT '',
	quality_profile_id INTEGER REFERENCES quality_profiles(id) ON DELETE SET NULL
);

CREATE TABLE IF NOT EXISTS part_groups (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	name TEXT NOT NULL,
	order_num INTEGER NOT NULL DEFAULT 0,
	media_id INTEGER NOT NULL DEFAULT 0 REFERENCES media(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS parts (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	name TEXT,
	group_order INTEGER,
	group_id INTEGER REFERENCES part_groups(id) ON DELETE SET NULL,
	media_id INTEGER NOT NULL REFERENCES media(id) ON DELETE CASCADE,
	path TEXT,
	monitored INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS indexers (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	name TEXT NOT NULL,
	type TEXT NOT NULL DEFAULT 'prowlarr',
	settings TEXT NOT NULL DEFAULT '{}',
	enabled INTEGER NOT NULL DEFAULT 1
);

CREATE TABLE IF NOT EXISTS download_clients (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	name TEXT NOT NULL,
	type TEXT NOT NULL DEFAULT 'qbittorrent',
	settings TEXT NOT NULL DEFAULT '{}',
	enabled INTEGER NOT NULL DEFAULT 1
);

CREATE TABLE IF NOT EXISTS queue (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	media_id INTEGER NOT NULL REFERENCES media(id) ON DELETE CASCADE,
	part_ids TEXT NOT NULL DEFAULT '',
	grabber_name TEXT NOT NULL,
	job_id TEXT NOT NULL,
	download_id TEXT,
	release_title TEXT NOT NULL DEFAULT '',
	state TEXT NOT NULL DEFAULT 'queued',
	progress REAL NOT NULL DEFAULT 0,
	added_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS history (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	media_id INTEGER NOT NULL REFERENCES media(id) ON DELETE CASCADE,
	part_id INTEGER REFERENCES parts(id) ON DELETE SET NULL,
	event_type TEXT NOT NULL,
	release_title TEXT NOT NULL DEFAULT '',
	data TEXT NOT NULL DEFAULT '',
	created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS tasks (
	name TEXT PRIMARY KEY,
	interval TEXT NOT NULL DEFAULT '24h',
	enabled INTEGER NOT NULL DEFAULT 1,
	last_run TEXT,
	next_run TEXT,
	last_status TEXT NOT NULL DEFAULT 'idle',
	last_error TEXT NOT NULL DEFAULT '',
	settings TEXT NOT NULL DEFAULT '{}',
	data TEXT NOT NULL DEFAULT '{}'
);

CREATE TABLE IF NOT EXISTS sources (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	type TEXT NOT NULL,
	name TEXT NOT NULL DEFAULT '',
	settings TEXT NOT NULL DEFAULT '{}',
	enabled INTEGER NOT NULL DEFAULT 1,
	proxy_id INTEGER REFERENCES proxies(id) ON DELETE SET NULL,
	UNIQUE(type)
);

CREATE TABLE IF NOT EXISTS users (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	name TEXT NOT NULL,
	email TEXT NOT NULL UNIQUE,
	password_hash TEXT,
	role TEXT NOT NULL DEFAULT 'user',
	created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS user_sessions (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	refresh_token_hash TEXT NOT NULL,
	expires_at TEXT NOT NULL,
	created_at TEXT NOT NULL,
	revoked INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS oidc_identities (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	issuer TEXT NOT NULL,
	subject TEXT NOT NULL,
	UNIQUE(issuer, subject)
);

CREATE TABLE IF NOT EXISTS settings (
	key TEXT PRIMARY KEY,
	value TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS media_requests (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	provider TEXT NOT NULL,
	external_id TEXT NOT NULL,
	type INTEGER NOT NULL,
	library_id INTEGER NOT NULL,
	quality_profile_id INTEGER REFERENCES quality_profiles(id) ON DELETE SET NULL,
	folder TEXT,
	status TEXT NOT NULL DEFAULT 'pending',
	created_at TEXT NOT NULL,
	approved_at TEXT,
	approved_by INTEGER REFERENCES users(id) ON DELETE SET NULL,
	rejected_at TEXT,
	rejected_by INTEGER REFERENCES users(id) ON DELETE SET NULL,
	canceled_at TEXT,
	notes TEXT,
	media_id INTEGER REFERENCES media(id) ON DELETE SET NULL,
	title TEXT,
	cover TEXT
);

CREATE INDEX IF NOT EXISTS idx_media_library_id ON media(library_id);
CREATE INDEX IF NOT EXISTS idx_media_quality_profile_id ON media(quality_profile_id);
CREATE INDEX IF NOT EXISTS idx_part_groups_media_id ON part_groups(media_id);
CREATE INDEX IF NOT EXISTS idx_parts_media_id ON parts(media_id);
CREATE INDEX IF NOT EXISTS idx_parts_group_id ON parts(group_id);
CREATE INDEX IF NOT EXISTS idx_queue_media_id ON queue(media_id);
CREATE INDEX IF NOT EXISTS idx_history_media_id ON history(media_id);
CREATE INDEX IF NOT EXISTS idx_history_part_id ON history(part_id);
CREATE INDEX IF NOT EXISTS idx_sources_proxy_id ON sources(proxy_id);
CREATE INDEX IF NOT EXISTS idx_sessions_user_id ON user_sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_sessions_refresh_hash ON user_sessions(refresh_token_hash);
CREATE INDEX IF NOT EXISTS idx_oidc_identities_user_id ON oidc_identities(user_id);
CREATE INDEX IF NOT EXISTS idx_requests_user_id ON media_requests(user_id);
CREATE INDEX IF NOT EXISTS idx_requests_status ON media_requests(status);
CREATE INDEX IF NOT EXISTS idx_requests_provider_external ON media_requests(provider, external_id);
CREATE INDEX IF NOT EXISTS idx_requests_media_id ON media_requests(media_id);

INSERT OR IGNORE INTO tasks (name, interval, enabled) VALUES ('refresh-metadata', '24h', 1);
INSERT OR IGNORE INTO tasks (name, interval, enabled) VALUES ('grab-missing', '1h', 1);
INSERT OR IGNORE INTO tasks (name, interval, enabled) VALUES ('check-content', '6h', 1);
INSERT OR IGNORE INTO tasks (name, interval, enabled) VALUES ('rss', '15m', 1);
INSERT OR IGNORE INTO tasks (name, interval, enabled, settings, data)
VALUES ('cleanup-stuck', '10m', 0, '{"penalty":"3"}', '{}');

INSERT OR IGNORE INTO settings (key, value) VALUES ('auto_approve_requests', '0');
