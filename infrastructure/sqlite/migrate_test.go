package sqlite

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunMigrations_FreshDB_CreatesAllTables(t *testing.T) {
	db := newTestDB(t)

	expected := []string{
		"libraries", "quality_profiles", "proxies",
		"media", "part_groups", "parts",
		"indexers", "download_clients", "queue",
		"history", "tasks", "sources",
		"users", "user_sessions", "oidc_identities",
		"settings", "media_requests",
		"schema_migrations",
	}
	rows, err := db.Query(`SELECT name FROM sqlite_master WHERE type='table'`)
	require.NoError(t, err)
	defer rows.Close()
	got := map[string]bool{}
	for rows.Next() {
		var name string
		require.NoError(t, rows.Scan(&name))
		got[name] = true
	}
	require.NoError(t, rows.Err())
	for _, name := range expected {
		assert.True(t, got[name], "table %s should exist", name)
	}
}

func TestRunMigrations_SecondRunIsNoChange(t *testing.T) {
	db := newTestDB(t)
	require.NoError(t, RunMigrations(db))
}

func TestRunMigrations_FreshDB_VersionIsOne(t *testing.T) {
	db := newTestDB(t)
	var version int
	var dirty bool
	require.NoError(t, db.QueryRow(`SELECT version, dirty FROM schema_migrations`).Scan(&version, &dirty))
	assert.Equal(t, 1, version)
	assert.False(t, dirty)
}

func TestForeignKeys_Enforced_PartWithoutMedia(t *testing.T) {
	db := newTestDB(t)
	_, err := db.Exec(`INSERT INTO parts (name, media_id, monitored) VALUES ('orphan', 99999, 1)`)
	assert.Error(t, err, "inserting part with non-existent media_id must violate FK")
}

func TestNewDB_EnforcesForeignKeys(t *testing.T) {
	db, err := NewDB(t.TempDir() + "/fk.db")
	require.NoError(t, err)
	defer db.Close()
	require.NoError(t, RunMigrations(db))
	var fk int
	require.NoError(t, db.QueryRow(`PRAGMA foreign_keys`).Scan(&fk))
	assert.Equal(t, 1, fk)
}

func TestNewDB_EnablesWALAndBusyTimeout(t *testing.T) {
	db, err := NewDB(t.TempDir() + "/wal.db")
	require.NoError(t, err)
	defer db.Close()

	var journal string
	require.NoError(t, db.QueryRow(`PRAGMA journal_mode`).Scan(&journal))
	assert.Equal(t, "wal", journal)

	var timeout int
	require.NoError(t, db.QueryRow(`PRAGMA busy_timeout`).Scan(&timeout))
	assert.Equal(t, 5000, timeout)
}

func TestNewDB_RejectsReadOnlyFile(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix file modes")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "ro.db")
	db, err := NewDB(path)
	require.NoError(t, err)
	require.NoError(t, db.Close())

	require.NoError(t, os.Chmod(path, 0o444))
	require.NoError(t, os.Chmod(dir, 0o555))
	t.Cleanup(func() {
		_ = os.Chmod(dir, 0o755)
		_ = os.Chmod(path, 0o644)
	})

	_, err = NewDB(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not writable")
}
