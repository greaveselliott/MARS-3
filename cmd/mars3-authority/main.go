/*
FactoryDocSync:
docs:
- docs/features/F-002-work-authority.md
- docs/design-docs/ADR-001-git-beads-authority.md
- docs/code-documentation-map.md
*/

package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/greaveselliott/MARS-3/internal/authority/bootstrap"
	"github.com/greaveselliott/MARS-3/internal/authority/closeout"
)

const usage = `usage:
  mars3-authority bootstrap-claim --repo <path> --beads-source <path> --beads-workspace <path> --beads-binary <path> [--execution-authorization <path> --apply]
  mars3-authority terminal-reconcile --repo <path> --tenant <id> --project <id> --beads-workspace <path> --beads-binary <path> --beads-binary-sha256 <digest> --postgres-url-file <path> --fence-generation <id> --execution-authorization <path> [--apply]`

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "ERROR:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, usage)
		return errors.New("authority command is required")
	}
	switch args[0] {
	case "bootstrap-claim":
		return runBootstrapClaim(args[1:])
	case "terminal-reconcile":
		return runTerminalReconcile(args[1:])
	default:
		fmt.Fprintln(os.Stderr, usage)
		return errors.New("unknown authority command")
	}
}

func runBootstrapClaim(args []string) error {
	flags := flag.NewFlagSet("bootstrap-claim", flag.ContinueOnError)
	repo := flags.String("repo", ".", "MARS-3 repository root")
	beadsSource := flags.String("beads-source", "", "clean local Beads checkout at the signed revision")
	beadsWorkspace := flags.String("beads-workspace", "", "external canonical Beads workspace")
	beadsBinary := flags.String("beads-binary", "", "pinned unmodified Beads binary")
	executionAuthorization := flags.String("execution-authorization", "", "external signed post-review execution authorization")
	apply := flags.Bool("apply", false, "execute the one canonical claim after all checks")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return errors.New("unexpected positional arguments")
	}
	return bootstrap.Run(bootstrap.Options{
		Repo: *repo, BeadsSource: *beadsSource, BeadsWorkspace: *beadsWorkspace,
		BeadsBinary: *beadsBinary, ExecutionAuthorization: *executionAuthorization,
		Apply: *apply, Stdout: os.Stdout, Stderr: os.Stderr,
	})
}

func runTerminalReconcile(args []string) error {
	flags := flag.NewFlagSet("terminal-reconcile", flag.ContinueOnError)
	repo := flags.String("repo", ".", "MARS-3 repository root")
	tenant := flags.String("tenant", "", "canonical authority tenant identifier")
	project := flags.String("project", "", "canonical authority project identifier")
	beadsWorkspace := flags.String("beads-workspace", "", "external canonical Beads workspace")
	beadsBinary := flags.String("beads-binary", "", "pinned gateway-enabled Beads binary")
	beadsBinarySHA256 := flags.String("beads-binary-sha256", "", "signed SHA-256 of the gateway-enabled Beads binary")
	postgresURLFile := flags.String("postgres-url-file", "", "regular file containing the canonical PostgreSQL URL")
	fenceGeneration := flags.String("fence-generation", "", "externally anchored non-reusable fence generation")
	executionAuthorization := flags.String("execution-authorization", "", "external signed one-hour terminal execution authorization")
	apply := flags.Bool("apply", false, "execute the canonical W-001 terminal reconciliation")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return errors.New("unexpected positional arguments")
	}
	return closeout.Run(context.Background(), closeout.Options{
		Repo: *repo, TenantID: *tenant, ProjectID: *project, BeadsWorkspace: *beadsWorkspace,
		BeadsBinary: *beadsBinary, BeadsBinarySHA256: *beadsBinarySHA256, PostgreSQLURLFile: *postgresURLFile,
		FenceGeneration: *fenceGeneration, ExecutionAuthorization: *executionAuthorization,
		Apply: *apply, Stdout: os.Stdout, Stderr: os.Stderr,
	})
}
