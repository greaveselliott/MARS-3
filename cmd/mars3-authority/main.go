/*
FactoryDocSync:
docs:
- docs/features/F-002-work-authority.md
- docs/design-docs/ADR-001-git-beads-authority.md
- docs/code-documentation-map.md
*/

package main

import (
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/greaveselliott/MARS-3/internal/authority/bootstrap"
)

const usage = `usage:
  mars3-authority bootstrap-claim --repo <path> --beads-source <path> --beads-workspace <path> --beads-binary <path> [--execution-authorization <path> --apply]`

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "ERROR:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 || args[0] != "bootstrap-claim" {
		fmt.Fprintln(os.Stderr, usage)
		return errors.New("bootstrap-claim is required")
	}
	flags := flag.NewFlagSet("bootstrap-claim", flag.ContinueOnError)
	repo := flags.String("repo", ".", "MARS-3 repository root")
	beadsSource := flags.String("beads-source", "", "clean local Beads checkout at the signed revision")
	beadsWorkspace := flags.String("beads-workspace", "", "external canonical Beads workspace")
	beadsBinary := flags.String("beads-binary", "", "pinned unmodified Beads binary")
	executionAuthorization := flags.String("execution-authorization", "", "external signed post-review execution authorization")
	apply := flags.Bool("apply", false, "execute the one canonical claim after all checks")
	if err := flags.Parse(args[1:]); err != nil {
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
