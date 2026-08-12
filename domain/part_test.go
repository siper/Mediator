package domain

import (
	"testing"
)

func TestPart_Validate(t *testing.T) {
	partName := "Chapter 1"
	emptyName := ""
	groupID := ID(10)

	tests := []struct {
		name    string
		part    *Part
		wantErr error
	}{
		{
			name:    "valid part",
			part:    &Part{Name: &partName, MediaId: 1},
			wantErr: nil,
		},
		{
			name:    "missing media id",
			part:    &Part{Name: &partName, MediaId: 0},
			wantErr: ErrMediaNotFound,
		},
		{
			name:    "empty name pointer",
			part:    &Part{Name: &emptyName, MediaId: 1},
			wantErr: ErrEmptyName,
		},
		{
			name:    "nil name is valid",
			part:    &Part{Name: nil, MediaId: 1},
			wantErr: nil,
		},
		{
			name:    "with group",
			part:    &Part{Name: &partName, MediaId: 1, GroupId: &groupID},
			wantErr: nil,
		},
		{
			name:    "with path",
			part:    &Part{Name: &partName, MediaId: 1, Path: strPtr("/path/to/file")},
			wantErr: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.part.Validate()
			if tt.wantErr != nil {
				if err != tt.wantErr {
					t.Errorf("Part.Validate() error = %v, wantErr %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Errorf("Part.Validate() unexpected error = %v", err)
			}
		})
	}
}

func strPtr(s string) *string {
	return &s
}
