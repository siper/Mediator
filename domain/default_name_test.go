package domain

import "testing"

func TestIndexer_ApplyDefaultName(t *testing.T) {
	ix := &Indexer{Type: IndexerProwlarr, Settings: map[string]string{"endpoint": "http://x"}}
	if err := ix.Validate(); err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if ix.Name != "Prowlarr" {
		t.Fatalf("Name = %q, want Prowlarr", ix.Name)
	}

	ix2 := &Indexer{Type: IndexerJackett, Name: "  Custom  ", Settings: map[string]string{"endpoint": "http://x"}}
	if err := ix2.Validate(); err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if ix2.Name != "Custom" {
		t.Fatalf("Name = %q, want Custom", ix2.Name)
	}
}

func TestDownloadClient_ApplyDefaultName(t *testing.T) {
	cl := &DownloadClient{Type: DownloadClientQBittorrent, Settings: map[string]string{"host": "http://x"}}
	if err := cl.Validate(); err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if cl.Name != "qBittorrent" {
		t.Fatalf("Name = %q, want qBittorrent", cl.Name)
	}
}

func TestLibrary_ApplyDefaultName(t *testing.T) {
	l := &Library{Type: MediaTypeMovie, Path: "movies"}
	if err := l.Validate(); err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if l.Name != "Movies" {
		t.Fatalf("Name = %q, want Movies", l.Name)
	}
}

func TestSource_ApplyDefaultName(t *testing.T) {
	s := &Source{Type: string(SourceTMDB), Settings: map[string]string{"api_key": "k"}}
	if err := s.Validate(); err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if s.Name != "TMDB" {
		t.Fatalf("Name = %q, want TMDB", s.Name)
	}
}
