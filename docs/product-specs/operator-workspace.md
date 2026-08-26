# Operator workspace specification

**Status:** Planned
**Goal:** G-001
**Decision link:** PD-003
**Planned Bead:** UI-001

The hosted web client and installable Electron client will share a React and
TypeScript application. The left sidebar exposes organizations, projects,
active-plan state, Beads, jobs, modes, approvals, and blockers. The main
timeline exposes human steering, public-safe activity summaries, artifact
commits, policy decisions, tool receipts, verification, handoffs, review
verdicts, run dispositions, and release verdicts. It never exposes private
chain-of-thought.

Each task header shows the durable lineage `Goal -> Product decision ->
Feature/scenario -> Active plan -> Bead -> Job -> Run attempt`. Sidebar work
state comes only from Beads plus factory projections; provider threads,
projects, todos, and subagents are attempt-local details and never become work
authority. The runtime selector distinguishes hosted agent-loop adapters from
the native open-model harness, shows the pinned capability snapshot, and keeps
an unqualified adapter disabled with its reason. There is no silent fallback.

An ancillary advisor chat is non-mutating until a human promotes a message to a
structured steering command. Browser, terminal, file, diff, tool, BDD,
decision, Bead, evidence, policy, cost, and diagnostic panes attach only to an
isolated agent sandbox.

Steering becomes durable before it is delivered to an adapter and reports
`applied`, `queued`, or `rejected`. Human takeover cancels or pauses the
adapter, asks the gateway to revoke the current epoch and issue a new one only
after full tuple validation, and invalidates earlier
verification. Handback requires a fresh diff, DocSync audit, policy decision,
and verification. Browser content is streamed from an ephemeral compartment;
the renderer never receives browser control protocols, host files, authority
credentials, provider keys, or raw model reasoning.

The Electron shell is thin and can connect to local or hosted endpoints. The
web and Electron clients replay the same ordered event history.
