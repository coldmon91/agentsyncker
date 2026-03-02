package backup

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"agentsyncker/internal/config"
)

func TestBackupFileAndRestore(t *testing.T) {
	tmp := t.TempDir()
	backupRoot := filepath.Join(tmp, "backups")
	manager, err := NewManager(backupRoot)
	if err != nil {
		t.Fatalf("new manager failed: %v", err)
	}

	home := filepath.Join(tmp, "home", ".gemini")
	if err := os.MkdirAll(home, 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	mainPath := filepath.Join(home, "GEMINI.md")
	if err := os.WriteFile(mainPath, []byte("original"), 0o644); err != nil {
		t.Fatalf("write file failed: %v", err)
	}

	backupName, err := manager.BackupFile("gemini", mainPath)
	if err != nil {
		t.Fatalf("backup file failed: %v", err)
	}

	if err := os.WriteFile(mainPath, []byte("changed"), 0o644); err != nil {
		t.Fatalf("write modified file failed: %v", err)
	}

	tool := config.Tool{
		Name:       "gemini",
		HomeDir:    home,
		MainFile:   "GEMINI.md",
		CommandDir: "commands",
		SkillDir:   "skills",
	}

	restoredPath, err := manager.Restore(tool, backupName)
	if err != nil {
		t.Fatalf("restore failed: %v", err)
	}
	if restoredPath != mainPath {
		t.Fatalf("unexpected restored path: %s", restoredPath)
	}

	content, err := os.ReadFile(mainPath)
	if err != nil {
		t.Fatalf("read restored file failed: %v", err)
	}
	if string(content) != "original" {
		t.Fatalf("unexpected restored content: %s", string(content))
	}

	entries, err := manager.List("gemini")
	if err != nil {
		t.Fatalf("list backups failed: %v", err)
	}
	preRestoreFound := false
	for _, entry := range entries {
		if entry.PreRestore {
			preRestoreFound = true
			break
		}
	}
	if !preRestoreFound {
		t.Fatal("expected pre-restore backup to be created")
	}
}

func TestBackupDirectoryAndRestore(t *testing.T) {
	tmp := t.TempDir()
	backupRoot := filepath.Join(tmp, "backups")
	manager, err := NewManager(backupRoot)
	if err != nil {
		t.Fatalf("new manager failed: %v", err)
	}

	home := filepath.Join(tmp, "home", ".gemini")
	commandsDir := filepath.Join(home, "commands")
	if err := os.MkdirAll(commandsDir, 0o755); err != nil {
		t.Fatalf("mkdir commands failed: %v", err)
	}
	originalFile := filepath.Join(commandsDir, "test.toml")
	if err := os.WriteFile(originalFile, []byte("description = \"A\"\n"), 0o644); err != nil {
		t.Fatalf("write original file failed: %v", err)
	}

	backupName, err := manager.BackupDirectory("gemini", commandsDir)
	if err != nil {
		t.Fatalf("backup directory failed: %v", err)
	}

	if err := os.RemoveAll(commandsDir); err != nil {
		t.Fatalf("remove commands dir failed: %v", err)
	}
	if err := os.MkdirAll(commandsDir, 0o755); err != nil {
		t.Fatalf("recreate commands dir failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(commandsDir, "new.toml"), []byte("new"), 0o644); err != nil {
		t.Fatalf("write replacement file failed: %v", err)
	}

	tool := config.Tool{
		Name:       "gemini",
		HomeDir:    home,
		MainFile:   "GEMINI.md",
		CommandDir: "commands",
		SkillDir:   "skills",
	}

	if _, err := manager.Restore(tool, backupName); err != nil {
		t.Fatalf("restore directory failed: %v", err)
	}

	restoredContent, err := os.ReadFile(originalFile)
	if err != nil {
		t.Fatalf("read restored original failed: %v", err)
	}
	if string(restoredContent) != "description = \"A\"\n" {
		t.Fatalf("unexpected restored content: %q", string(restoredContent))
	}

	if _, err := os.Stat(filepath.Join(commandsDir, "new.toml")); !os.IsNotExist(err) {
		t.Fatal("expected replaced file to be gone after restore")
	}
}

func TestBackupPruneKeepTen(t *testing.T) {
	tmp := t.TempDir()
	backupRoot := filepath.Join(tmp, "backups")
	manager, err := NewManager(backupRoot)
	if err != nil {
		t.Fatalf("new manager failed: %v", err)
	}

	baseTime := time.Date(2026, 3, 2, 12, 0, 0, 0, time.UTC)
	counter := 0
	manager.Now = func() time.Time {
		t := baseTime.Add(time.Duration(counter) * time.Second)
		counter++
		return t
	}

	file := filepath.Join(tmp, "GEMINI.md")
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatalf("write source file failed: %v", err)
	}

	for i := 0; i < 12; i++ {
		if err := os.WriteFile(file, []byte(strings.Repeat("x", i+1)), 0o644); err != nil {
			t.Fatalf("update source file failed: %v", err)
		}
		if _, err := manager.BackupFile("gemini", file); err != nil {
			t.Fatalf("backup iteration %d failed: %v", i, err)
		}
	}

	entries, err := os.ReadDir(backupRoot)
	if err != nil {
		t.Fatalf("read backup dir failed: %v", err)
	}

	count := 0
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), "gemini_GEMINI.md_") && strings.HasSuffix(entry.Name(), ".bak") {
			count++
		}
	}
	if count != 10 {
		t.Fatalf("expected 10 backups after prune, got %d", count)
	}
}

func TestBackupFileSkipsWhenContentHashIsSame(t *testing.T) {
	tmp := t.TempDir()
	backupRoot := filepath.Join(tmp, "backups")
	manager, err := NewManager(backupRoot)
	if err != nil {
		t.Fatalf("new manager failed: %v", err)
	}

	baseTime := time.Date(2026, 3, 2, 12, 0, 0, 0, time.UTC)
	counter := 0
	manager.Now = func() time.Time {
		tm := baseTime.Add(time.Duration(counter) * time.Second)
		counter++
		return tm
	}

	file := filepath.Join(tmp, "GEMINI.md")
	if err := os.WriteFile(file, []byte("same-content"), 0o644); err != nil {
		t.Fatalf("write source file failed: %v", err)
	}

	first, err := manager.BackupFile("gemini", file)
	if err != nil {
		t.Fatalf("first backup failed: %v", err)
	}
	if first == "" {
		t.Fatal("expected first backup to be created")
	}

	second, err := manager.BackupFile("gemini", file)
	if err != nil {
		t.Fatalf("second backup failed: %v", err)
	}
	if second != "" {
		t.Fatalf("expected duplicate backup to be skipped, got %s", second)
	}

	entries, err := manager.List("gemini")
	if err != nil {
		t.Fatalf("list backups failed: %v", err)
	}
	backupCount := 0
	for _, entry := range entries {
		if entry.Asset == "GEMINI.md" && entry.Extension == ".bak" {
			backupCount++
		}
	}
	if backupCount != 1 {
		t.Fatalf("expected one file backup after duplicate skip, got %d", backupCount)
	}
}

func TestBackupDirectorySkipsWhenContentHashIsSame(t *testing.T) {
	tmp := t.TempDir()
	backupRoot := filepath.Join(tmp, "backups")
	manager, err := NewManager(backupRoot)
	if err != nil {
		t.Fatalf("new manager failed: %v", err)
	}

	baseTime := time.Date(2026, 3, 2, 12, 0, 0, 0, time.UTC)
	counter := 0
	manager.Now = func() time.Time {
		tm := baseTime.Add(time.Duration(counter) * time.Second)
		counter++
		return tm
	}

	commandsDir := filepath.Join(tmp, "commands")
	if err := os.MkdirAll(commandsDir, 0o755); err != nil {
		t.Fatalf("mkdir commands failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(commandsDir, "test.toml"), []byte("description = \"A\"\n"), 0o644); err != nil {
		t.Fatalf("write command file failed: %v", err)
	}

	first, err := manager.BackupDirectory("gemini", commandsDir)
	if err != nil {
		t.Fatalf("first dir backup failed: %v", err)
	}
	if first == "" {
		t.Fatal("expected first dir backup to be created")
	}

	second, err := manager.BackupDirectory("gemini", commandsDir)
	if err != nil {
		t.Fatalf("second dir backup failed: %v", err)
	}
	if second != "" {
		t.Fatalf("expected duplicate dir backup to be skipped, got %s", second)
	}

	entries, err := manager.List("gemini")
	if err != nil {
		t.Fatalf("list backups failed: %v", err)
	}
	backupCount := 0
	for _, entry := range entries {
		if entry.Asset == "commands" && entry.Extension == ".tar.gz" {
			backupCount++
		}
	}
	if backupCount != 1 {
		t.Fatalf("expected one dir backup after duplicate skip, got %d", backupCount)
	}
}

func TestBackupToolSnapshotCreatesSingleArchive(t *testing.T) {
	tmp := t.TempDir()
	backupRoot := filepath.Join(tmp, "backups")
	manager, err := NewManager(backupRoot)
	if err != nil {
		t.Fatalf("new manager failed: %v", err)
	}

	tool := config.Tool{
		Name:       "gemini",
		HomeDir:    filepath.Join(tmp, "home", ".gemini"),
		MainFile:   "GEMINI.md",
		CommandDir: "commands",
		SkillDir:   "skills",
	}

	if err := os.MkdirAll(tool.CommandDirPath(), 0o755); err != nil {
		t.Fatalf("create command dir failed: %v", err)
	}
	if err := os.MkdirAll(tool.SkillDirPath(), 0o755); err != nil {
		t.Fatalf("create skill dir failed: %v", err)
	}
	if err := os.WriteFile(tool.MainFilePath(), []byte("main"), 0o644); err != nil {
		t.Fatalf("write main file failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tool.CommandDirPath(), "test.toml"), []byte("cmd"), 0o644); err != nil {
		t.Fatalf("write command file failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tool.SkillDirPath(), "SKILL.md"), []byte("skill"), 0o644); err != nil {
		t.Fatalf("write skill file failed: %v", err)
	}

	name, err := manager.BackupToolSnapshot(tool)
	if err != nil {
		t.Fatalf("backup snapshot failed: %v", err)
	}
	if name == "" {
		t.Fatal("expected snapshot backup file")
	}
	if !strings.Contains(name, "gemini_snapshot_") || !strings.HasSuffix(name, ".tar.gz") {
		t.Fatalf("unexpected snapshot name: %s", name)
	}

	entries, err := os.ReadDir(backupRoot)
	if err != nil {
		t.Fatalf("read backup root failed: %v", err)
	}

	hasArchive := false
	artifactCount := 0
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".tar.gz") {
			hasArchive = true
		}
		if strings.HasPrefix(entry.Name(), "gemini_snapshot_") {
			artifactCount++
		}
	}
	if !hasArchive {
		t.Fatal("expected snapshot archive")
	}
	if artifactCount == 0 {
		t.Fatal("expected snapshot artifacts in backup directory")
	}
}

func TestBackupToolSnapshotSkipsWhenUnchanged(t *testing.T) {
	tmp := t.TempDir()
	backupRoot := filepath.Join(tmp, "backups")
	manager, err := NewManager(backupRoot)
	if err != nil {
		t.Fatalf("new manager failed: %v", err)
	}

	tool := config.Tool{
		Name:       "gemini",
		HomeDir:    filepath.Join(tmp, "home", ".gemini"),
		MainFile:   "GEMINI.md",
		CommandDir: "commands",
		SkillDir:   "skills",
	}
	if err := os.MkdirAll(tool.CommandDirPath(), 0o755); err != nil {
		t.Fatalf("create command dir failed: %v", err)
	}
	if err := os.MkdirAll(tool.SkillDirPath(), 0o755); err != nil {
		t.Fatalf("create skill dir failed: %v", err)
	}
	if err := os.WriteFile(tool.MainFilePath(), []byte("main"), 0o644); err != nil {
		t.Fatalf("write main file failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tool.CommandDirPath(), "test.toml"), []byte("cmd"), 0o644); err != nil {
		t.Fatalf("write command file failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tool.SkillDirPath(), "SKILL.md"), []byte("skill"), 0o644); err != nil {
		t.Fatalf("write skill file failed: %v", err)
	}

	first, err := manager.BackupToolSnapshot(tool)
	if err != nil {
		t.Fatalf("first snapshot failed: %v", err)
	}
	if first == "" {
		t.Fatal("expected first snapshot")
	}

	second, err := manager.BackupToolSnapshot(tool)
	if err != nil {
		t.Fatalf("second snapshot failed: %v", err)
	}
	if second != "" {
		t.Fatalf("expected unchanged snapshot skip, got %s", second)
	}
}

func TestRestoreFromToolSnapshot(t *testing.T) {
	tmp := t.TempDir()
	backupRoot := filepath.Join(tmp, "backups")
	manager, err := NewManager(backupRoot)
	if err != nil {
		t.Fatalf("new manager failed: %v", err)
	}

	tool := config.Tool{
		Name:       "gemini",
		HomeDir:    filepath.Join(tmp, "home", ".gemini"),
		MainFile:   "GEMINI.md",
		CommandDir: "commands",
		SkillDir:   "skills",
	}
	if err := os.MkdirAll(tool.CommandDirPath(), 0o755); err != nil {
		t.Fatalf("create command dir failed: %v", err)
	}
	if err := os.MkdirAll(tool.SkillDirPath(), 0o755); err != nil {
		t.Fatalf("create skill dir failed: %v", err)
	}
	if err := os.WriteFile(tool.MainFilePath(), []byte("main-v1"), 0o644); err != nil {
		t.Fatalf("write main file failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tool.CommandDirPath(), "test.toml"), []byte("cmd-v1"), 0o644); err != nil {
		t.Fatalf("write command file failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tool.SkillDirPath(), "SKILL.md"), []byte("skill-v1"), 0o644); err != nil {
		t.Fatalf("write skill file failed: %v", err)
	}

	snapshotName, err := manager.BackupToolSnapshot(tool)
	if err != nil {
		t.Fatalf("backup snapshot failed: %v", err)
	}
	if snapshotName == "" {
		t.Fatal("expected snapshot backup")
	}

	if err := os.WriteFile(tool.MainFilePath(), []byte("main-v2"), 0o644); err != nil {
		t.Fatalf("update main file failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tool.CommandDirPath(), "test.toml"), []byte("cmd-v2"), 0o644); err != nil {
		t.Fatalf("update command file failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tool.SkillDirPath(), "SKILL.md"), []byte("skill-v2"), 0o644); err != nil {
		t.Fatalf("update skill file failed: %v", err)
	}

	if _, err := manager.Restore(tool, snapshotName); err != nil {
		t.Fatalf("restore snapshot failed: %v", err)
	}

	mainContent, err := os.ReadFile(tool.MainFilePath())
	if err != nil {
		t.Fatalf("read restored main failed: %v", err)
	}
	if string(mainContent) != "main-v1" {
		t.Fatalf("unexpected restored main: %s", string(mainContent))
	}
	commandContent, err := os.ReadFile(filepath.Join(tool.CommandDirPath(), "test.toml"))
	if err != nil {
		t.Fatalf("read restored command failed: %v", err)
	}
	if string(commandContent) != "cmd-v1" {
		t.Fatalf("unexpected restored command: %s", string(commandContent))
	}
	skillContent, err := os.ReadFile(filepath.Join(tool.SkillDirPath(), "SKILL.md"))
	if err != nil {
		t.Fatalf("read restored skill failed: %v", err)
	}
	if string(skillContent) != "skill-v1" {
		t.Fatalf("unexpected restored skill: %s", string(skillContent))
	}
}

func TestDeleteBackupRemovesFileAndMetadata(t *testing.T) {
	tmp := t.TempDir()
	backupRoot := filepath.Join(tmp, "backups")
	manager, err := NewManager(backupRoot)
	if err != nil {
		t.Fatalf("new manager failed: %v", err)
	}

	sourceFile := filepath.Join(tmp, "GEMINI.md")
	if err := os.WriteFile(sourceFile, []byte("content"), 0o644); err != nil {
		t.Fatalf("write source file failed: %v", err)
	}

	backupName, err := manager.BackupFile("gemini", sourceFile)
	if err != nil {
		t.Fatalf("backup file failed: %v", err)
	}
	if backupName == "" {
		t.Fatal("expected backup to be created")
	}

	backupPath := filepath.Join(backupRoot, backupName)
	metadataPath := backupPath + hashMetadataExtension
	if _, err := os.Stat(metadataPath); err != nil {
		t.Fatalf("expected hash metadata to exist: %v", err)
	}

	if err := manager.Delete(backupName); err != nil {
		t.Fatalf("delete backup failed: %v", err)
	}

	if _, err := os.Stat(backupPath); !os.IsNotExist(err) {
		t.Fatalf("expected backup file removed, stat err=%v", err)
	}
	if _, err := os.Stat(metadataPath); !os.IsNotExist(err) {
		t.Fatalf("expected backup metadata removed, stat err=%v", err)
	}
}

func TestExtractTarGzRejectsPathTraversal(t *testing.T) {
	tmp := t.TempDir()
	archivePath := filepath.Join(tmp, "malicious.tar.gz")
	if err := writeTarGz(archivePath, "../outside.txt", []byte("x")); err != nil {
		t.Fatalf("create malicious tar failed: %v", err)
	}

	dest := filepath.Join(tmp, "dest")
	if err := os.MkdirAll(dest, 0o755); err != nil {
		t.Fatalf("create dest dir failed: %v", err)
	}

	err := extractTarGz(archivePath, dest)
	if err == nil {
		t.Fatal("expected extractTarGz to reject path traversal")
	}
	if !strings.Contains(err.Error(), "invalid tar path") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func writeTarGz(path string, name string, content []byte) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	gz := gzip.NewWriter(file)
	defer gz.Close()

	tw := tar.NewWriter(gz)
	defer tw.Close()

	header := &tar.Header{
		Name: name,
		Mode: 0o644,
		Size: int64(len(content)),
	}
	if err := tw.WriteHeader(header); err != nil {
		return err
	}
	if _, err := io.Copy(tw, bytes.NewReader(content)); err != nil {
		return err
	}
	return nil
}
