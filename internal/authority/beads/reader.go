/*
FactoryDocSync:
docs:
- docs/features/F-002-work-authority.md
- docs/design-docs/ADR-001-git-beads-authority.md
- docs/code-documentation-map.md
*/

package beads

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

const maximumBeadsOutput = 2 << 20

type CLIReader struct {
	binarySHA256 string
	binary       string
	workspace    string
	projectID    string
}

func NewCLIReader(binary, binarySHA256, workspace, projectID string) (*CLIReader, error) {
	reader := &CLIReader{binarySHA256: binarySHA256, binary: binary, workspace: workspace, projectID: projectID}
	if err := reader.verifyBoundary(); err != nil {
		return nil, err
	}
	return reader, nil
}

func (reader *CLIReader) ReadIssue(ctx context.Context, beadID string) ([]byte, error) {
	if beadID == "" || strings.ContainsAny(beadID, " /\\") {
		return nil, ErrProjectionInvalid
	}
	return reader.run(ctx, "show", beadID, "--json")
}

func (reader *CLIReader) ListIssueIDs(ctx context.Context) ([]string, error) {
	output, err := reader.run(ctx, "list", "--limit", "100", "--json")
	if err != nil {
		return nil, err
	}
	var rows []map[string]json.RawMessage
	if json.Unmarshal(output, &rows) != nil || len(rows) > 100 {
		return nil, ErrProjectionInvalid
	}
	ids := make([]string, 0, len(rows))
	for _, row := range rows {
		var id string
		if decodeField(row, "id", &id) != nil || id == "" {
			return nil, ErrProjectionInvalid
		}
		ids = append(ids, id)
	}
	if hasDuplicateStrings(ids) {
		return nil, ErrProjectionInvalid
	}
	sort.Strings(ids)
	return ids, nil
}

func (reader *CLIReader) run(ctx context.Context, args ...string) ([]byte, error) {
	if err := reader.verifyBoundary(); err != nil {
		return nil, err
	}
	scratchHome, err := os.MkdirTemp("", "mars3-beads-reader-")
	if err != nil {
		return nil, ErrProjectionInvalid
	}
	commandArgs := append([]string{"-C", reader.workspace, "--readonly"}, args...)
	command := exec.CommandContext(ctx, reader.binary, commandArgs...)
	command.Env = reader.environment(scratchHome)
	var output boundedBuffer
	command.Stdout = &output
	command.Stderr = io.Discard
	commandErr := command.Run()
	cleanupErr := os.RemoveAll(scratchHome)
	if commandErr != nil || cleanupErr != nil {
		return nil, ErrProjectionInvalid
	}
	return output.Bytes(), nil
}

func (reader *CLIReader) environment(scratchHome string) []string {
	return []string{
		"BEADS_DIR=" + filepath.Join(reader.workspace, ".beads"),
		"BEADS_DOLT_SERVER_DATABASE=M3", "BEADS_DOLT_SHARED_SERVER=0",
		"BEADS_DOLT_SERVER_MODE=0", "BD_DISABLE_METRICS=1", "BD_NO_HOOKS=1",
		"HOME=" + scratchHome, "XDG_CONFIG_HOME=" + filepath.Join(scratchHome, ".config"),
		"NO_COLOR=1", "LANG=C", "TZ=UTC",
	}
}

func (reader *CLIReader) verifyBoundary() error {
	binary, err := filepath.Abs(reader.binary)
	if err != nil || binary != filepath.Clean(reader.binary) || !digestFile(binary, reader.binarySHA256) {
		return ErrProjectionInvalid
	}
	binaryInfo, err := os.Lstat(binary)
	if err != nil || !binaryInfo.Mode().IsRegular() || binaryInfo.Mode()&os.ModeSymlink != 0 {
		return ErrProjectionInvalid
	}
	workspace, err := filepath.Abs(reader.workspace)
	if err != nil || workspace != filepath.Clean(reader.workspace) {
		return ErrProjectionInvalid
	}
	workspaceInfo, err := os.Lstat(workspace)
	if err != nil || !workspaceInfo.IsDir() || workspaceInfo.Mode()&os.ModeSymlink != 0 {
		return ErrProjectionInvalid
	}
	beadsDir := filepath.Join(workspace, ".beads")
	beadsInfo, err := os.Lstat(beadsDir)
	if err != nil || !beadsInfo.IsDir() || beadsInfo.Mode()&os.ModeSymlink != 0 {
		return ErrProjectionInvalid
	}
	for _, forbidden := range []string{"redirect", ".env", "config.local.yaml"} {
		if _, err := os.Lstat(filepath.Join(beadsDir, forbidden)); !errors.Is(err, os.ErrNotExist) {
			return ErrProjectionInvalid
		}
	}
	if config, err := os.ReadFile(filepath.Join(beadsDir, "config.yaml")); err == nil {
		for _, line := range strings.Split(string(config), "\n") {
			trimmed := strings.TrimSpace(line)
			if trimmed != "" && !strings.HasPrefix(trimmed, "#") {
				return ErrProjectionInvalid
			}
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return ErrProjectionInvalid
	}
	metadata, err := os.ReadFile(filepath.Join(beadsDir, "metadata.json"))
	if err != nil {
		return ErrProjectionInvalid
	}
	var identity struct {
		Backend      string `json:"backend"`
		Database     string `json:"database"`
		DoltMode     string `json:"dolt_mode"`
		DoltDatabase string `json:"dolt_database"`
		ProjectID    string `json:"project_id"`
	}
	decoder := json.NewDecoder(bytes.NewReader(metadata))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&identity) != nil || identity.Backend != "dolt" || identity.Database != "dolt" || identity.DoltMode != "embedded" || identity.DoltDatabase != "M3" || identity.ProjectID != reader.projectID {
		return ErrProjectionInvalid
	}
	return nil
}

type boundedBuffer struct {
	data bytes.Buffer
}

func (buffer *boundedBuffer) Write(data []byte) (int, error) {
	if buffer.data.Len()+len(data) > maximumBeadsOutput {
		return 0, ErrProjectionInvalid
	}
	return buffer.data.Write(data)
}

func (buffer *boundedBuffer) Bytes() []byte { return append([]byte(nil), buffer.data.Bytes()...) }

func digestFile(path, expected string) bool {
	if len(expected) != 64 {
		return false
	}
	file, err := os.Open(path)
	if err != nil {
		return false
	}
	defer file.Close()
	digest := sha256.New()
	if _, err := io.Copy(digest, file); err != nil {
		return false
	}
	return hex.EncodeToString(digest.Sum(nil)) == expected
}
