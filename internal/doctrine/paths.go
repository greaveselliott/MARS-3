/*
FactoryDocSync:
docs:
- docs/features/F-001-doctrine-foundation.md
- docs/design-docs/mars-provenance.md
- docs/code-documentation-map.md
*/

package doctrine

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const maxAuditedFileSize = 4 << 20

func repositoryRoot(repo string) (string, error) {
	if strings.TrimSpace(repo) == "" {
		repo = "."
	}
	root, err := filepath.Abs(repo)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(root)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", errors.New("repository root is not a directory")
	}
	return root, nil
}

func cleanPublicPath(path string) string {
	path = filepath.ToSlash(filepath.Clean(path))
	if path == "." || path == "" {
		return "."
	}
	for strings.HasPrefix(path, "../") {
		path = strings.TrimPrefix(path, "../")
	}
	return strings.TrimPrefix(path, "./")
}

func readRepoFile(root, relative string) ([]byte, error) {
	return os.ReadFile(filepath.Join(root, filepath.FromSlash(relative)))
}

func repoFileExists(root, relative string) bool {
	info, err := os.Stat(filepath.Join(root, filepath.FromSlash(relative)))
	return err == nil && info.Mode().IsRegular()
}

func walkRepository(root string) ([]string, error) {
	return walkRepositoryWithIgnoredDirectories(root, ignoredDirectory)
}

func walkPublicRepository(root string) ([]string, error) {
	return walkRepositoryWithIgnoredDirectories(root, func(path string) bool {
		return filepath.Base(path) == ".git"
	})
}

func walkRepositoryWithIgnoredDirectories(root string, ignored func(string) bool) ([]string, error) {
	var paths []string
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		relative = cleanPublicPath(relative)
		if entry.IsDir() {
			if relative == "." {
				return nil
			}
			if ignored(relative) {
				return filepath.SkipDir
			}
			return nil
		}
		if entry.Type().IsRegular() {
			paths = append(paths, relative)
		}
		return nil
	})
	sort.Strings(paths)
	return paths, err
}

func ignoredDirectory(path string) bool {
	base := filepath.Base(path)
	switch base {
	case ".git", "node_modules", "vendor", "dist", "build", ".cache", ".idea", ".vscode":
		return true
	default:
		return false
	}
}

func readAuditedFile(root, relative string) ([]byte, error) {
	path := filepath.Join(root, filepath.FromSlash(relative))
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if info.Size() > maxAuditedFileSize {
		return nil, errors.New("file exceeds audit size limit")
	}
	return os.ReadFile(path)
}
