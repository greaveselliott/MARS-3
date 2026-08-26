/*
FactoryDocSync:
docs:
- docs/features/F-001-doctrine-foundation.md
- docs/design-docs/mars-provenance.md
- docs/code-documentation-map.md
*/

package doctrine

import (
	"fmt"
	"io"
	"sort"
	"strings"
)

// Finding is one deterministic validation failure. Path is always repository
// relative so a public evidence log cannot disclose a developer workstation.
type Finding struct {
	Path    string
	Code    string
	Message string
}

func (f Finding) String() string {
	location := f.Path
	if location == "" {
		location = "."
	}
	return fmt.Sprintf("%s: %s: %s", location, f.Code, f.Message)
}

func sortFindings(findings []Finding) {
	sort.Slice(findings, func(i, j int) bool {
		if findings[i].Path != findings[j].Path {
			return findings[i].Path < findings[j].Path
		}
		if findings[i].Code != findings[j].Code {
			return findings[i].Code < findings[j].Code
		}
		return findings[i].Message < findings[j].Message
	})
}

// WriteReport emits a bounded, stable report suitable for CI evidence.
func WriteReport(w io.Writer, checkName string, findings []Finding) error {
	sortFindings(findings)
	if len(findings) == 0 {
		_, err := fmt.Fprintf(w, "PASS %s\n", checkName)
		return err
	}
	if _, err := fmt.Fprintf(w, "FAIL %s (%d findings)\n", checkName, len(findings)); err != nil {
		return err
	}
	for _, finding := range findings {
		if _, err := fmt.Fprintln(w, finding.String()); err != nil {
			return err
		}
	}
	return nil
}

func addFinding(findings *[]Finding, path, code, format string, args ...any) {
	*findings = append(*findings, Finding{
		Path:    cleanPublicPath(path),
		Code:    code,
		Message: strings.TrimSpace(fmt.Sprintf(format, args...)),
	})
}
