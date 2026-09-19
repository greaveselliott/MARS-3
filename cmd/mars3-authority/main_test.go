/*
FactoryDocSync:
docs:
- docs/features/F-002-work-authority.md
- docs/design-docs/ADR-001-git-beads-authority.md
- docs/code-documentation-map.md
*/

package main

import (
	"strings"
	"testing"
)

func TestRunRejectsUnknownAuthorityCommand(t *testing.T) {
	if err := run([]string{"unknown"}); err == nil || !strings.Contains(err.Error(), "unknown") {
		t.Fatalf("unknown command error = %v", err)
	}
}

func TestOperatorRequiresProtectedSignedInputs(t *testing.T) {
	for _, args := range [][]string{{"operator"}, {"operator", "unexpected"}, {"operator", "--connection", "forbidden"}} {
		if err := run(args); err == nil || strings.Contains(err.Error(), "unknown authority command") {
			t.Fatalf("operator did not fail closed at its own admission boundary: %v", err)
		}
	}
}

func TestTerminalReconcileRequiresCompleteBoundedInputs(t *testing.T) {
	if err := run([]string{"terminal-reconcile"}); err == nil {
		t.Fatal("terminal reconciliation accepted missing canonical handles")
	}
	if err := run([]string{"terminal-reconcile", "unexpected"}); err == nil || !strings.Contains(err.Error(), "positional") {
		t.Fatalf("unexpected positional error = %v", err)
	}
}

func TestBootstrapClaimStillRejectsUnexpectedPositionals(t *testing.T) {
	if err := run([]string{"bootstrap-claim", "unexpected"}); err == nil || !strings.Contains(err.Error(), "positional") {
		t.Fatalf("unexpected positional error = %v", err)
	}
}
