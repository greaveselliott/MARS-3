/*
FactoryDocSync:
docs:
- docs/features/F-001-doctrine-foundation.md
- docs/design-docs/ADR-001-git-beads-authority.md
- docs/code-documentation-map.md
*/

package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/greaveselliott/MARS-3/internal/doctrine"
)

const usage = `usage:
  mars3 doctrine check --repo <path>
  mars3 doctrine refresh --repo <path> --source <local-checkout> --ref <commit> [--apply]
  mars3 plan check --repo <path>
  mars3 docsync audit --repo <path>
  mars3 public-check --repo <path>`

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, "ERROR:", err)
		os.Exit(1)
	}
}

func run(args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		fmt.Fprintln(stderr, usage)
		return errors.New("command is required")
	}
	switch args[0] {
	case "doctrine":
		return runDoctrine(args[1:], stdout, stderr)
	case "plan":
		return runCheckCommand("plan", "check", args[1:], stdout, stderr, doctrine.CheckPlan)
	case "docsync":
		return runCheckCommand("docsync", "audit", args[1:], stdout, stderr, doctrine.AuditDocSync)
	case "public-check":
		return runCheckCommand("public-check", "", args[1:], stdout, stderr, doctrine.CheckPublic)
	case "help", "-h", "--help":
		fmt.Fprintln(stdout, usage)
		return nil
	default:
		fmt.Fprintln(stderr, usage)
		return fmt.Errorf("unknown command %q", args[0])
	}
}

type checkFunction func(string) ([]doctrine.Finding, error)

func runCheckCommand(name, operation string, args []string, stdout, stderr io.Writer, check checkFunction) error {
	if operation != "" {
		if len(args) == 0 || args[0] != operation {
			fmt.Fprintln(stderr, usage)
			return fmt.Errorf("%s requires %s", name, operation)
		}
		args = args[1:]
	}
	flags := flag.NewFlagSet(name, flag.ContinueOnError)
	flags.SetOutput(stderr)
	repo := flags.String("repo", ".", "repository root")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return errors.New("unexpected positional arguments")
	}
	findings, err := check(*repo)
	if err != nil {
		return err
	}
	if err := doctrine.WriteReport(stdout, name, findings); err != nil {
		return err
	}
	if len(findings) > 0 {
		return fmt.Errorf("%s failed", name)
	}
	return nil
}

func runDoctrine(args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		fmt.Fprintln(stderr, usage)
		return errors.New("doctrine requires check or refresh")
	}
	if args[0] == "check" {
		return runCheckCommand("doctrine", "check", args, stdout, stderr, doctrine.CheckDoctrine)
	}
	if args[0] != "refresh" {
		fmt.Fprintln(stderr, usage)
		return fmt.Errorf("unknown doctrine operation %q", args[0])
	}
	flags := flag.NewFlagSet("doctrine refresh", flag.ContinueOnError)
	flags.SetOutput(stderr)
	repo := flags.String("repo", ".", "repository root")
	source := flags.String("source", "", "local MARS checkout")
	ref := flags.String("ref", "", "exact MARS commit")
	apply := flags.Bool("apply", false, "atomically rewrite the generated provenance manifest")
	if err := flags.Parse(args[1:]); err != nil {
		return err
	}
	if flags.NArg() != 0 || *source == "" || *ref == "" {
		return errors.New("doctrine refresh requires --source and --ref")
	}
	result, err := doctrine.RefreshDoctrine(*repo, *source, *ref, *apply)
	if err != nil {
		return err
	}
	mode := "dry-run"
	if result.Applied {
		mode = "apply"
	}
	fmt.Fprintf(stdout, "PASS doctrine-refresh mode=%s target=%s source_files=%d\n", mode, result.Target, result.SourceFiles)
	return nil
}
