package backup

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"proman/internal/config"
)

type Entry struct {
	Name       string
	Path       string
	Tool       string
	Asset      string
	Timestamp  time.Time
	PreRestore bool
	Extension  string
}

type Manager struct {
	RootDir string
	Keep    int
	Now     func() time.Time
}

var backupNamePattern = regexp.MustCompile(`^([a-z0-9]+)_(.+?)_(\d{8}_\d{6})(?:_(pre_restore))?(\.bak|\.tar\.gz)$`)

const hashMetadataExtension = ".sha256"
const snapshotAsset = "snapshot"

func DefaultRoot(home string) string {
	return filepath.Join(home, ".proman", "backups")
}

func NewManager(rootDir string) (*Manager, error) {
	if err := os.MkdirAll(rootDir, 0o755); err != nil {
		return nil, fmt.Errorf("create backup root: %w", err)
	}
	return &Manager{RootDir: rootDir, Keep: 10, Now: time.Now}, nil
}

func (m *Manager) BackupFile(tool string, sourcePath string) (string, error) {
	return m.backupFile(tool, sourcePath, false)
}

func (m *Manager) BackupDirectory(tool string, sourceDir string) (string, error) {
	return m.backupDirectory(tool, sourceDir, false)
}

func (m *Manager) BackupFilePreRestore(tool string, sourcePath string) (string, error) {
	return m.backupFile(tool, sourcePath, true)
}

func (m *Manager) BackupDirectoryPreRestore(tool string, sourceDir string) (string, error) {
	return m.backupDirectory(tool, sourceDir, true)
}

func (m *Manager) BackupToolSnapshot(tool config.Tool) (string, error) {
	return m.backupToolSnapshot(tool, false)
}

func (m *Manager) BackupToolSnapshotPreRestore(tool config.Tool) (string, error) {
	return m.backupToolSnapshot(tool, true)
}

func (m *Manager) List(tool string) ([]Entry, error) {
	entries, err := os.ReadDir(m.RootDir)
	if err != nil {
		return nil, fmt.Errorf("read backup root: %w", err)
	}

	result := make([]Entry, 0, len(entries))
	for _, dirEntry := range entries {
		if dirEntry.IsDir() {
			continue
		}
		entry, ok := parseBackupName(dirEntry.Name())
		if !ok {
			continue
		}
		if tool != "" && entry.Tool != tool {
			continue
		}
		entry.Path = filepath.Join(m.RootDir, dirEntry.Name())
		result = append(result, entry)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Timestamp.After(result[j].Timestamp)
	})
	return result, nil
}

func (m *Manager) Delete(backupName string) error {
	if _, ok := parseBackupName(backupName); !ok {
		return fmt.Errorf("invalid backup filename: %s", backupName)
	}

	backupPath := filepath.Join(m.RootDir, backupName)
	if err := os.Remove(backupPath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("backup not found: %s", backupName)
		}
		return fmt.Errorf("delete backup file: %w", err)
	}

	metadataPath := backupPath + hashMetadataExtension
	if err := os.Remove(metadataPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("delete backup hash metadata: %w", err)
	}

	return nil
}

func (m *Manager) Restore(tool config.Tool, backupName string) (string, error) {
	entry, ok := parseBackupName(backupName)
	if !ok {
		return "", fmt.Errorf("invalid backup filename: %s", backupName)
	}
	if entry.Tool != tool.Name {
		return "", fmt.Errorf("backup tool mismatch: expected %s, got %s", tool.Name, entry.Tool)
	}

	backupPath := filepath.Join(m.RootDir, backupName)
	if _, err := os.Stat(backupPath); err != nil {
		return "", fmt.Errorf("backup not found: %w", err)
	}

	if entry.Asset == snapshotAsset && entry.Extension == ".tar.gz" {
		if _, err := m.BackupToolSnapshotPreRestore(tool); err != nil {
			return "", fmt.Errorf("pre-restore backup failed: %w", err)
		}
		if err := restoreToolSnapshot(backupPath, tool, m.Now); err != nil {
			return "", err
		}
		return tool.HomeDir, nil
	}

	targetPath, err := tool.ResolveAssetPath(entry.Asset, entry.Extension)
	if err != nil {
		return "", err
	}

	if entry.Extension == ".bak" {
		if _, err := os.Stat(targetPath); err == nil {
			if _, err := m.BackupFilePreRestore(tool.Name, targetPath); err != nil {
				return "", fmt.Errorf("pre-restore backup failed: %w", err)
			}
		}
		if err := restoreFile(backupPath, targetPath); err != nil {
			return "", err
		}
		return targetPath, nil
	}

	if entry.Extension == ".tar.gz" {
		if info, err := os.Stat(targetPath); err == nil && info.IsDir() {
			if _, err := m.BackupDirectoryPreRestore(tool.Name, targetPath); err != nil {
				return "", fmt.Errorf("pre-restore backup failed: %w", err)
			}
		}
		if err := restoreDirectory(backupPath, targetPath, entry.Asset, m.Now); err != nil {
			return "", err
		}
		return targetPath, nil
	}

	return "", fmt.Errorf("unsupported backup extension: %s", entry.Extension)
}

func parseBackupName(name string) (Entry, bool) {
	matches := backupNamePattern.FindStringSubmatch(name)
	if len(matches) != 6 {
		return Entry{}, false
	}

	ts, err := time.Parse("20060102_150405", matches[3])
	if err != nil {
		return Entry{}, false
	}

	return Entry{
		Name:       name,
		Tool:       matches[1],
		Asset:      matches[2],
		Timestamp:  ts,
		PreRestore: matches[4] == "pre_restore",
		Extension:  matches[5],
	}, true
}

func (m *Manager) backupFile(tool string, sourcePath string, preRestore bool) (string, error) {
	info, err := os.Stat(sourcePath)
	if err != nil {
		return "", fmt.Errorf("stat source file: %w", err)
	}
	if info.IsDir() {
		return "", fmt.Errorf("source path is a directory: %s", sourcePath)
	}

	asset := filepath.Base(sourcePath)
	sourceHash, err := hashFile(sourcePath)
	if err != nil {
		return "", fmt.Errorf("hash source file: %w", err)
	}
	skip, err := m.shouldSkipBackup(tool, asset, ".bak", sourceHash)
	if err != nil {
		return "", err
	}
	if skip {
		return "", nil
	}

	name := makeBackupName(tool, asset, ".bak", m.Now(), preRestore)
	target := filepath.Join(m.RootDir, name)
	if err := copyFile(sourcePath, target); err != nil {
		return "", err
	}
	if err := m.writeHashMetadata(name, sourceHash); err != nil {
		return "", err
	}
	if err := m.prune(tool, asset, ".bak"); err != nil {
		return "", err
	}
	return name, nil
}

func (m *Manager) backupDirectory(tool string, sourceDir string, preRestore bool) (string, error) {
	info, err := os.Stat(sourceDir)
	if err != nil {
		return "", fmt.Errorf("stat source dir: %w", err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("source path is not a directory: %s", sourceDir)
	}

	asset := filepath.Base(sourceDir)
	sourceHash, err := hashDirectory(sourceDir)
	if err != nil {
		return "", fmt.Errorf("hash source dir: %w", err)
	}
	skip, err := m.shouldSkipBackup(tool, asset, ".tar.gz", sourceHash)
	if err != nil {
		return "", err
	}
	if skip {
		return "", nil
	}

	name := makeBackupName(tool, asset, ".tar.gz", m.Now(), preRestore)
	target := filepath.Join(m.RootDir, name)
	if err := createTarGz(sourceDir, target); err != nil {
		return "", err
	}
	if err := m.writeHashMetadata(name, sourceHash); err != nil {
		return "", err
	}
	if err := m.prune(tool, asset, ".tar.gz"); err != nil {
		return "", err
	}
	return name, nil
}

type snapshotSource struct {
	archiveRoot string
	sourcePath  string
	isDir       bool
}

func (m *Manager) backupToolSnapshot(tool config.Tool, preRestore bool) (string, error) {
	sources, err := collectSnapshotSources(tool)
	if err != nil {
		return "", err
	}
	if len(sources) == 0 {
		return "", nil
	}

	sourceHash, err := hashSnapshotSources(sources)
	if err != nil {
		return "", fmt.Errorf("hash snapshot sources: %w", err)
	}

	skip, err := m.shouldSkipBackup(tool.Name, snapshotAsset, ".tar.gz", sourceHash)
	if err != nil {
		return "", err
	}
	if skip {
		return "", nil
	}

	name := makeBackupName(tool.Name, snapshotAsset, ".tar.gz", m.Now(), preRestore)
	target := filepath.Join(m.RootDir, name)
	if err := createSnapshotTarGz(sources, target); err != nil {
		return "", err
	}
	if err := m.writeHashMetadata(name, sourceHash); err != nil {
		return "", err
	}
	if err := m.prune(tool.Name, snapshotAsset, ".tar.gz"); err != nil {
		return "", err
	}
	return name, nil
}

func collectSnapshotSources(tool config.Tool) ([]snapshotSource, error) {
	sources := make([]snapshotSource, 0, 4)

	if info, err := os.Stat(tool.MainFilePath()); err == nil && !info.IsDir() {
		sources = append(sources, snapshotSource{
			archiveRoot: filepath.ToSlash(filepath.Join("main", tool.MainFile)),
			sourcePath:  tool.MainFilePath(),
			isDir:       false,
		})
	} else if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("stat main file: %w", err)
	}

	dirs := []struct {
		archiveRoot string
		path        string
	}{
		{archiveRoot: "commands", path: tool.CommandDirPath()},
		{archiveRoot: "skills", path: tool.SkillDirPath()},
	}
	if tool.AgentDirPath() != "" {
		dirs = append(dirs, struct {
			archiveRoot string
			path        string
		}{archiveRoot: "agents", path: tool.AgentDirPath()})
	}

	for _, dir := range dirs {
		if info, err := os.Stat(dir.path); err == nil && info.IsDir() {
			sources = append(sources, snapshotSource{
				archiveRoot: dir.archiveRoot,
				sourcePath:  dir.path,
				isDir:       true,
			})
		} else if err != nil && !os.IsNotExist(err) {
			return nil, fmt.Errorf("stat source dir %s: %w", dir.path, err)
		}
	}

	sort.Slice(sources, func(i, j int) bool {
		return sources[i].archiveRoot < sources[j].archiveRoot
	})
	return sources, nil
}

func (m *Manager) prune(tool string, asset string, extension string) error {
	entries, err := os.ReadDir(m.RootDir)
	if err != nil {
		return fmt.Errorf("read backup root for prune: %w", err)
	}

	prefix := tool + "_" + asset + "_"
	matching := make([]string, 0)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasPrefix(name, prefix) || !strings.HasSuffix(name, extension) {
			continue
		}
		matching = append(matching, filepath.Join(m.RootDir, name))
	}

	if len(matching) <= m.Keep {
		return nil
	}

	sort.Slice(matching, func(i, j int) bool {
		ii, iErr := os.Stat(matching[i])
		jj, jErr := os.Stat(matching[j])
		if iErr != nil || jErr != nil {
			return matching[i] > matching[j]
		}
		return ii.ModTime().After(jj.ModTime())
	})

	for _, path := range matching[m.Keep:] {
		if err := os.Remove(path); err != nil {
			return fmt.Errorf("prune old backup %s: %w", path, err)
		}
		metadataPath := path + hashMetadataExtension
		if err := os.Remove(metadataPath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("prune backup hash metadata %s: %w", metadataPath, err)
		}
	}
	return nil
}

func (m *Manager) shouldSkipBackup(tool string, asset string, extension string, sourceHash string) (bool, error) {
	latestEntry, ok, err := m.latestBackupEntry(tool, asset, extension)
	if err != nil || !ok {
		return false, err
	}

	hashValue, found, err := m.readHashMetadata(latestEntry.Name)
	if err != nil {
		return false, err
	}
	if !found && extension == ".bak" {
		hashValue, err = hashFile(latestEntry.Path)
		if err != nil {
			return false, fmt.Errorf("hash latest backup file: %w", err)
		}
	}
	if hashValue == "" {
		return false, nil
	}
	return hashValue == sourceHash, nil
}

func (m *Manager) latestBackupEntry(tool string, asset string, extension string) (Entry, bool, error) {
	entries, err := m.List(tool)
	if err != nil {
		return Entry{}, false, err
	}

	for _, entry := range entries {
		if entry.Asset == asset && entry.Extension == extension {
			return entry, true, nil
		}
	}
	return Entry{}, false, nil
}

func (m *Manager) writeHashMetadata(backupName string, hashValue string) error {
	metadataPath := filepath.Join(m.RootDir, backupName+hashMetadataExtension)
	if err := os.WriteFile(metadataPath, []byte(hashValue+"\n"), 0o644); err != nil {
		return fmt.Errorf("write backup hash metadata: %w", err)
	}
	return nil
}

func (m *Manager) readHashMetadata(backupName string) (string, bool, error) {
	metadataPath := filepath.Join(m.RootDir, backupName+hashMetadataExtension)
	content, err := os.ReadFile(metadataPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", false, nil
		}
		return "", false, fmt.Errorf("read backup hash metadata: %w", err)
	}
	value := strings.TrimSpace(string(content))
	if value == "" {
		return "", false, nil
	}
	return value, true, nil
}

func hashFile(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hasher.Sum(nil)), nil
}

func hashDirectory(root string) (string, error) {
	hasher := sha256.New()
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		relPath, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if relPath == "." {
			return nil
		}
		relPath = filepath.ToSlash(relPath)

		if d.IsDir() {
			if _, err := hasher.Write([]byte("D:" + relPath + "\n")); err != nil {
				return err
			}
			return nil
		}

		info, err := d.Info()
		if err != nil {
			return err
		}
		if _, err := hasher.Write([]byte(fmt.Sprintf("F:%s:%o\n", relPath, info.Mode().Perm()))); err != nil {
			return err
		}

		file, err := os.Open(path)
		if err != nil {
			return err
		}
		if _, err := io.Copy(hasher, file); err != nil {
			file.Close()
			return err
		}
		if err := file.Close(); err != nil {
			return err
		}
		if _, err := hasher.Write([]byte{'\n'}); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(hasher.Sum(nil)), nil
}

func hashSnapshotSources(sources []snapshotSource) (string, error) {
	hasher := sha256.New()
	for _, source := range sources {
		marker := "file"
		if source.isDir {
			marker = "dir"
		}
		if _, err := hasher.Write([]byte(marker + ":" + source.archiveRoot + "\n")); err != nil {
			return "", err
		}

		pathHash, err := hashPathForSnapshot(source.sourcePath)
		if err != nil {
			return "", err
		}
		if _, err := hasher.Write([]byte(pathHash + "\n")); err != nil {
			return "", err
		}
	}
	return hex.EncodeToString(hasher.Sum(nil)), nil
}

func hashPathForSnapshot(path string) (string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	if info.IsDir() {
		return hashDirectory(path)
	}
	return hashFile(path)
}

func makeBackupName(tool string, asset string, extension string, now time.Time, preRestore bool) string {
	timestamp := now.Format("20060102_150405")
	suffix := ""
	if preRestore {
		suffix = "_pre_restore"
	}
	return fmt.Sprintf("%s_%s_%s%s%s", tool, asset, timestamp, suffix, extension)
}

func copyFile(sourcePath string, targetPath string) error {
	source, err := os.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("open source file: %w", err)
	}
	defer source.Close()

	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		return fmt.Errorf("create target dir: %w", err)
	}

	target, err := os.Create(targetPath)
	if err != nil {
		return fmt.Errorf("create target file: %w", err)
	}
	defer target.Close()

	if _, err := io.Copy(target, source); err != nil {
		return fmt.Errorf("copy file contents: %w", err)
	}

	return nil
}

func createTarGz(sourceDir string, targetPath string) error {
	file, err := os.Create(targetPath)
	if err != nil {
		return fmt.Errorf("create tar.gz file: %w", err)
	}
	defer file.Close()

	gzipWriter := gzip.NewWriter(file)
	defer gzipWriter.Close()

	tarWriter := tar.NewWriter(gzipWriter)
	defer tarWriter.Close()

	baseDir := filepath.Base(sourceDir)
	return filepath.Walk(sourceDir, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		relPath, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return err
		}

		tarPath := baseDir
		if relPath != "." {
			tarPath = filepath.ToSlash(filepath.Join(baseDir, relPath))
		}
		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		header.Name = tarPath

		if err := tarWriter.WriteHeader(header); err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		fileToWrite, err := os.Open(path)
		if err != nil {
			return err
		}
		defer fileToWrite.Close()

		if _, err := io.Copy(tarWriter, fileToWrite); err != nil {
			return err
		}
		return nil
	})
}

func createSnapshotTarGz(sources []snapshotSource, targetPath string) error {
	file, err := os.Create(targetPath)
	if err != nil {
		return fmt.Errorf("create snapshot tar.gz file: %w", err)
	}
	defer file.Close()

	gzipWriter := gzip.NewWriter(file)
	defer gzipWriter.Close()

	tarWriter := tar.NewWriter(gzipWriter)
	defer tarWriter.Close()

	for _, source := range sources {
		if source.isDir {
			if err := addDirectoryToTar(tarWriter, source.sourcePath, source.archiveRoot); err != nil {
				return err
			}
			continue
		}

		if err := addFileToTar(tarWriter, source.sourcePath, source.archiveRoot); err != nil {
			return err
		}
	}
	return nil
}

func addDirectoryToTar(tarWriter *tar.Writer, sourceDir string, archiveRoot string) error {
	return filepath.Walk(sourceDir, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		relPath, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return err
		}

		tarPath := archiveRoot
		if relPath != "." {
			tarPath = filepath.ToSlash(filepath.Join(archiveRoot, relPath))
		}

		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		header.Name = tarPath

		if err := tarWriter.WriteHeader(header); err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		fileToWrite, err := os.Open(path)
		if err != nil {
			return err
		}
		if _, err := io.Copy(tarWriter, fileToWrite); err != nil {
			fileToWrite.Close()
			return err
		}
		if err := fileToWrite.Close(); err != nil {
			return err
		}
		return nil
	})
}

func addFileToTar(tarWriter *tar.Writer, sourceFile string, archivePath string) error {
	info, err := os.Stat(sourceFile)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return fmt.Errorf("source path is directory: %s", sourceFile)
	}

	header, err := tar.FileInfoHeader(info, "")
	if err != nil {
		return err
	}
	header.Name = filepath.ToSlash(archivePath)
	if err := tarWriter.WriteHeader(header); err != nil {
		return err
	}

	fileToWrite, err := os.Open(sourceFile)
	if err != nil {
		return err
	}
	if _, err := io.Copy(tarWriter, fileToWrite); err != nil {
		fileToWrite.Close()
		return err
	}
	if err := fileToWrite.Close(); err != nil {
		return err
	}
	return nil
}

func restoreFile(backupPath string, targetPath string) error {
	content, err := os.ReadFile(backupPath)
	if err != nil {
		return fmt.Errorf("read backup file: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		return fmt.Errorf("create target dir: %w", err)
	}

	tmpPath := targetPath + ".proman-restore-tmp"
	if err := os.WriteFile(tmpPath, content, 0o644); err != nil {
		return fmt.Errorf("write restore temp file: %w", err)
	}

	if err := os.Rename(tmpPath, targetPath); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("replace target file: %w", err)
	}

	return nil
}

func restoreDirectory(backupPath string, targetDir string, expectedRoot string, nowFn func() time.Time) error {
	parentDir := filepath.Dir(targetDir)
	if err := os.MkdirAll(parentDir, 0o755); err != nil {
		return fmt.Errorf("create parent dir: %w", err)
	}

	tmpDir, err := os.MkdirTemp(parentDir, "proman-restore-*")
	if err != nil {
		return fmt.Errorf("create temp restore dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	if err := extractTarGz(backupPath, tmpDir); err != nil {
		return err
	}

	extractedRoot := filepath.Join(tmpDir, expectedRoot)
	if _, err := os.Stat(extractedRoot); err != nil {
		return fmt.Errorf("restore archive missing root %q: %w", expectedRoot, err)
	}

	oldPath := ""
	if info, err := os.Stat(targetDir); err == nil && info.IsDir() {
		oldPath = fmt.Sprintf("%s.proman-old-%s", targetDir, nowFn().Format("20060102150405"))
		if err := os.Rename(targetDir, oldPath); err != nil {
			return fmt.Errorf("move current dir aside: %w", err)
		}
	}

	if err := os.Rename(extractedRoot, targetDir); err != nil {
		if oldPath != "" {
			_ = os.Rename(oldPath, targetDir)
		}
		return fmt.Errorf("move restored dir into place: %w", err)
	}

	if oldPath != "" {
		_ = os.RemoveAll(oldPath)
	}
	return nil
}

func restoreToolSnapshot(backupPath string, tool config.Tool, nowFn func() time.Time) error {
	parentDir := filepath.Dir(tool.HomeDir)
	if err := os.MkdirAll(parentDir, 0o755); err != nil {
		return fmt.Errorf("create temp snapshot parent dir: %w", err)
	}

	tmpDir, err := os.MkdirTemp(parentDir, "proman-restore-snapshot-*")
	if err != nil {
		return fmt.Errorf("create temp snapshot restore dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	if err := extractTarGz(backupPath, tmpDir); err != nil {
		return err
	}

	snapshotMainPath := filepath.Join(tmpDir, "main", tool.MainFile)
	if info, err := os.Stat(snapshotMainPath); err == nil && !info.IsDir() {
		if err := restoreFile(snapshotMainPath, tool.MainFilePath()); err != nil {
			return err
		}
	}

	dirMappings := []struct {
		snapshotDir string
		targetDir   string
	}{
		{snapshotDir: filepath.Join(tmpDir, "commands"), targetDir: tool.CommandDirPath()},
		{snapshotDir: filepath.Join(tmpDir, "skills"), targetDir: tool.SkillDirPath()},
	}
	if tool.AgentDirPath() != "" {
		dirMappings = append(dirMappings, struct {
			snapshotDir string
			targetDir   string
		}{snapshotDir: filepath.Join(tmpDir, "agents"), targetDir: tool.AgentDirPath()})
	}

	for _, mapping := range dirMappings {
		info, err := os.Stat(mapping.snapshotDir)
		if err != nil || !info.IsDir() {
			continue
		}
		if err := restoreDirectoryFromPreparedSource(mapping.snapshotDir, mapping.targetDir, nowFn); err != nil {
			return err
		}
	}

	return nil
}

func restoreDirectoryFromPreparedSource(preparedDir string, targetDir string, nowFn func() time.Time) error {
	parentDir := filepath.Dir(targetDir)
	if err := os.MkdirAll(parentDir, 0o755); err != nil {
		return fmt.Errorf("create parent dir: %w", err)
	}

	oldPath := ""
	if info, err := os.Stat(targetDir); err == nil && info.IsDir() {
		oldPath = fmt.Sprintf("%s.proman-old-%s", targetDir, nowFn().Format("20060102150405"))
		if err := os.Rename(targetDir, oldPath); err != nil {
			return fmt.Errorf("move current dir aside: %w", err)
		}
	}

	if err := os.Rename(preparedDir, targetDir); err != nil {
		if oldPath != "" {
			_ = os.Rename(oldPath, targetDir)
		}
		return fmt.Errorf("move restored dir into place: %w", err)
	}

	if oldPath != "" {
		_ = os.RemoveAll(oldPath)
	}
	return nil
}

func extractTarGz(archivePath string, destination string) error {
	file, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("open archive: %w", err)
	}
	defer file.Close()

	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		return fmt.Errorf("open gzip reader: %w", err)
	}
	defer gzipReader.Close()

	tarReader := tar.NewReader(gzipReader)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("read tar entry: %w", err)
		}

		cleanName := filepath.Clean(header.Name)
		targetPath := filepath.Join(destination, cleanName)
		relTarget, err := filepath.Rel(destination, targetPath)
		if err != nil {
			return fmt.Errorf("resolve tar target path: %w", err)
		}
		if relTarget == ".." || strings.HasPrefix(relTarget, ".."+string(os.PathSeparator)) {
			return fmt.Errorf("invalid tar path: %s", header.Name)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(targetPath, os.FileMode(header.Mode)); err != nil {
				return fmt.Errorf("create restored dir: %w", err)
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
				return fmt.Errorf("create parent dir: %w", err)
			}
			out, err := os.Create(targetPath)
			if err != nil {
				return fmt.Errorf("create restored file: %w", err)
			}
			if _, err := io.Copy(out, tarReader); err != nil {
				out.Close()
				return fmt.Errorf("write restored file: %w", err)
			}
			if err := out.Close(); err != nil {
				return fmt.Errorf("close restored file: %w", err)
			}
		default:
			continue
		}
	}

	return nil
}
