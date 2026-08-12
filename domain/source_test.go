package domain

import "testing"

func TestSource_ValidateOpenLibrary(t *testing.T) {
	s := &Source{Type: string(SourceOpenLibrary), Name: "OpenLibrary", Settings: map[string]string{}}
	if err := s.Validate(); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestSource_ValidateRejectsInvalidType(t *testing.T) {
	s := &Source{Type: "bogus", Name: "x", Settings: map[string]string{}}
	if err := s.Validate(); err == nil {
		t.Fatal("expected error for invalid type, got nil")
	}
}
