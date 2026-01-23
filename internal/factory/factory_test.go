package factory

import (
	"testing"

	"github.com/pragmaticivan/faro/internal/detector"
)

func TestCreateScanner(t *testing.T) {
	tests := []struct {
		name    string
		pm      detector.PackageManager
		wantErr bool
	}{
		{"go", detector.Go, false},
		{"npm", detector.Npm, false},
		{"yarn", detector.Yarn, false},
		{"pnpm", detector.Pnpm, false},
		{"pip", detector.Pip, false},
		{"poetry", detector.Poetry, false},
		{"uv", detector.Uv, false},
		{"invalid", "invalid", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scanner, err := CreateScanner(tt.pm, "/tmp")
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateScanner() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && scanner == nil {
				t.Errorf("CreateScanner() returned nil scanner")
			}
		})
	}
}

func TestCreateUpdater(t *testing.T) {
	tests := []struct {
		name    string
		pm      detector.PackageManager
		wantErr bool
	}{
		{"go", detector.Go, false},
		{"npm", detector.Npm, false},
		{"yarn", detector.Yarn, false},
		{"pnpm", detector.Pnpm, false},
		{"pip", detector.Pip, false},
		{"poetry", detector.Poetry, false},
		{"uv", detector.Uv, false},
		{"invalid", "invalid", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			updater, err := CreateUpdater(tt.pm, "/tmp")
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateUpdater() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && updater == nil {
				t.Errorf("CreateUpdater() returned nil updater")
			}
		})
	}
}
