/*
FactoryDocSync:
docs:
- docs/features/F-001-doctrine-foundation.md
- docs/design-docs/mars-provenance.md
- docs/code-documentation-map.md
*/

package doctrine

import (
	"crypto/sha1" // Git's object format at the pinned source uses SHA-1 object IDs.
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// RefreshResult describes a bounded provenance refresh. Target is always a
// repository-relative generated path.
type RefreshResult struct {
	Target      string
	SourceFiles int
	Applied     bool
}

// RefreshDoctrine validates a local checkout against the pinned commit and
// blob IDs. In apply mode its sole possible write is the generated manifest.
func RefreshDoctrine(repo, source, ref string, apply bool) (RefreshResult, error) {
	return refreshDoctrine(repo, source, ref, apply, requiredMARSSourceBlobs)
}

func refreshDoctrine(repo, source, ref string, apply bool, required map[string]string) (RefreshResult, error) {
	result := RefreshResult{Target: ".harness/generated/mars/source-manifest.json", Applied: apply}
	root, err := repositoryRoot(repo)
	if err != nil {
		return result, err
	}
	sourceRoot, err := repositoryRoot(source)
	if err != nil {
		return result, fmt.Errorf("source checkout: %w", err)
	}
	if ref != marsCommit {
		return result, fmt.Errorf("ref must equal pinned MARS commit %s", marsCommit)
	}
	head, err := gitOutput(sourceRoot, "rev-parse", "HEAD")
	if err != nil {
		return result, errors.New("source checkout HEAD cannot be resolved")
	}
	if head != marsCommit {
		return result, errors.New("source checkout HEAD does not equal the pinned MARS commit")
	}

	manifestPath := filepath.Join(root, filepath.FromSlash(result.Target))
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return result, errors.New("generated MARS source manifest is missing")
	}
	var manifest marsSourceManifest
	if err := decodeStrictJSON(data, &manifest); err != nil {
		return result, fmt.Errorf("invalid generated MARS source manifest: %w", err)
	}
	if manifest.SchemaVersion != 1 || manifest.Upstream.Repository != marsRepository || manifest.Upstream.Commit != marsCommit || manifest.Upstream.License != "Apache-2.0" {
		return result, errors.New("generated MARS source manifest does not match the approved provenance pin")
	}
	if len(manifest.GeneratedScope) != 1 || manifest.GeneratedScope[0] != result.Target {
		return result, errors.New("generated scope must permit only the MARS source manifest")
	}
	observed := make(map[string]string)
	for _, sourceFile := range manifest.SourceFiles {
		if !safeRelativePath(sourceFile.Path) || !sha1Pattern.MatchString(sourceFile.GitBlob) {
			return result, errors.New("source manifest contains an invalid path or blob ID")
		}
		content, err := os.ReadFile(filepath.Join(sourceRoot, filepath.FromSlash(sourceFile.Path)))
		if err != nil {
			return result, fmt.Errorf("declared MARS source %q is missing", sourceFile.Path)
		}
		if gitBlobID(content) != sourceFile.GitBlob {
			return result, fmt.Errorf("declared MARS source %q does not match its pinned blob ID", sourceFile.Path)
		}
		observed[sourceFile.Path] = sourceFile.GitBlob
	}
	for requiredPath, requiredBlob := range required {
		if observed[requiredPath] != requiredBlob {
			return result, fmt.Errorf("required MARS source %q is absent or has the wrong blob ID", requiredPath)
		}
	}
	result.SourceFiles = len(manifest.SourceFiles)
	if !apply {
		return result, nil
	}

	canonical, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return result, err
	}
	canonical = append(canonical, '\n')
	if bytesEqual(data, canonical) {
		return result, nil
	}
	if err := atomicWriteFile(manifestPath, canonical, 0o644); err != nil {
		return result, fmt.Errorf("write generated source manifest: %w", err)
	}
	return result, nil
}

func gitOutput(directory string, arguments ...string) (string, error) {
	command := exec.Command("git", append([]string{"-C", directory}, arguments...)...)
	command.Env = append(os.Environ(), "GIT_CONFIG_NOSYSTEM=1")
	output, err := command.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

func gitBlobID(content []byte) string {
	hasher := sha1.New()
	_, _ = fmt.Fprintf(hasher, "blob %d%c", len(content), byte(0))
	_, _ = hasher.Write(content)
	return hex.EncodeToString(hasher.Sum(nil))
}

func bytesEqual(left, right []byte) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func atomicWriteFile(path string, data []byte, mode os.FileMode) error {
	directory := filepath.Dir(path)
	temporary, err := os.CreateTemp(directory, ".mars3-provenance-*")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	defer func() { _ = os.Remove(temporaryPath) }()
	if err := temporary.Chmod(mode); err != nil {
		_ = temporary.Close()
		return err
	}
	if _, err := temporary.Write(data); err != nil {
		_ = temporary.Close()
		return err
	}
	if err := temporary.Sync(); err != nil {
		_ = temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	return os.Rename(temporaryPath, path)
}
