package detector

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetect(t *testing.T) {
	tests := []struct {
		name         string
		files        []string
		wantManagers []PackageManager
		wantErr      bool
	}{
		{
			name:         "Go project",
			files:        []string{"go.mod", "go.sum"},
			wantManagers: []PackageManager{Go},
		},
		{
			name:         "npm project",
			files:        []string{"package.json", "package-lock.json"},
			wantManagers: []PackageManager{Npm},
		},
		{
			name:         "yarn project",
			files:        []string{"package.json", "yarn.lock"},
			wantManagers: []PackageManager{Yarn},
		},
		{
			name:         "pnpm project",
			files:        []string{"package.json", "pnpm-lock.yaml"},
			wantManagers: []PackageManager{Pnpm},
		},
		{
			name:         "poetry project",
			files:        []string{"pyproject.toml", "poetry.lock"},
			wantManagers: []PackageManager{Poetry},
		},
		{
			name:         "uv project",
			files:        []string{"pyproject.toml", "uv.lock"},
			wantManagers: []PackageManager{Uv},
		},
		{
			name:         "pip project",
			files:        []string{"requirements.txt"},
			wantManagers: []PackageManager{Pip},
		},
		{
			name:         "multiple managers (Go + npm)",
			files:        []string{"go.mod", "go.sum", "package.json", "package-lock.json"},
			wantManagers: []PackageManager{Go, Npm},
		},
		{
			name:    "no package manager",
			files:   []string{"README.md"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp directory for test
			tmpDir := t.TempDir()

			// Create test files
			for _, file := range tt.files {
				path := filepath.Join(tmpDir, file)
				if err := os.WriteFile(path, []byte("test"), 0644); err != nil {
					t.Fatalf("failed to create test file: %v", err)
				}
			}

			results, err := Detect(tmpDir)
			if (err != nil) != tt.wantErr {
				t.Errorf("Detect() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}

			if len(results) != len(tt.wantManagers) {
				t.Errorf("Detect() got %d managers, want %d", len(results), len(tt.wantManagers))
				return
			}

			for i, want := range tt.wantManagers {
				if results[i].Manager != want {
					t.Errorf("Detect()[%d] = %v, want %v", i, results[i].Manager, want)
				}
			}
		})
	}
}

func TestDetectSingle(t *testing.T) {
	tests := []struct {
		name        string
		files       []string
		wantManager PackageManager
	}{
		{
			name:        "Go has priority over npm",
			files:       []string{"go.mod", "package.json", "package-lock.json"},
			wantManager: Go,
		},
		{
			name:        "pnpm has priority over yarn",
			files:       []string{"package.json", "yarn.lock", "pnpm-lock.yaml"},
			wantManager: Pnpm,
		},
		{
			name:        "yarn has priority over npm",
			files:       []string{"package.json", "yarn.lock", "package-lock.json"},
			wantManager: Yarn,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			for _, file := range tt.files {
				path := filepath.Join(tmpDir, file)
				if err := os.WriteFile(path, []byte("test"), 0644); err != nil {
					t.Fatalf("failed to create test file: %v", err)
				}
			}

			result, err := DetectSingle(tmpDir)
			if err != nil {
				t.Fatalf("DetectSingle() error = %v", err)
			}
			if result.Manager != tt.wantManager {
				t.Errorf("DetectSingle() = %v, want %v", result.Manager, tt.wantManager)
			}
		})
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		manager string
		want    PackageManager
		wantErr bool
	}{
		{"valid go", "go", Go, false},
		{"valid npm", "npm", Npm, false},
		{"valid yarn", "yarn", Yarn, false},
		{"valid pnpm", "pnpm", Pnpm, false},
		{"valid pip", "pip", Pip, false},
		{"valid poetry", "poetry", Poetry, false},
		{"valid uv", "uv", Uv, false},
		{"invalid manager", "invalid", "", true},
		{"empty string", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Validate(tt.manager)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Validate() = %v, want %v", got, tt.want)
			}
		})
	}
}
