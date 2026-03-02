package detector

import (
	"os"
	"path/filepath"
	"testing"

	"proman/internal/config"
)

func TestDetect(t *testing.T) {
	tmp := t.TempDir()
	installedHome := filepath.Join(tmp, ".claude")
	notInstalledHome := filepath.Join(tmp, ".gemini")
	if err := os.MkdirAll(installedHome, 0o755); err != nil {
		t.Fatalf("failed to create installed dir: %v", err)
	}

	tools := []config.Tool{
		{Name: "claude", HomeDir: installedHome},
		{Name: "gemini", HomeDir: notInstalledHome},
	}

	statuses, err := Detect(tools)
	if err != nil {
		t.Fatalf("detect returned error: %v", err)
	}
	if len(statuses) != 2 {
		t.Fatalf("expected 2 statuses, got %d", len(statuses))
	}

	if !statuses[0].Installed {
		t.Fatal("expected first tool to be installed")
	}
	if statuses[1].Installed {
		t.Fatal("expected second tool to be not installed")
	}
}

func TestInstalledTools(t *testing.T) {
	tmp := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmp, "a"), 0o755); err != nil {
		t.Fatalf("failed to create dir: %v", err)
	}

	tools := []config.Tool{
		{Name: "a", HomeDir: filepath.Join(tmp, "a")},
		{Name: "b", HomeDir: filepath.Join(tmp, "b")},
	}

	installed, err := InstalledTools(tools)
	if err != nil {
		t.Fatalf("installed tools returned error: %v", err)
	}
	if len(installed) != 1 {
		t.Fatalf("expected one installed tool, got %d", len(installed))
	}
	if installed[0].Name != "a" {
		t.Fatalf("unexpected tool: %s", installed[0].Name)
	}
}
