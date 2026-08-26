# Effect admission guardrail

An effect is admissible only when all of the following are true:

- the current Bead, claim, and exclusive paths authorize it; after W-001, the
  current lease epoch must also authorize it;
- the action is a typed proposal accepted by policy;
- its data labels satisfy the Rule-of-Two constraint;
- an effect intent is durably recorded before execution;
- the tool broker can return a bounded, secret-free receipt;
- verification and a terminal run disposition can be correlated to the same
  trace; and
- required human approval is present for production, destructive, trust, or
  publication boundaries.

Policy rejection must name the current state, required transition, and exact
allowed corrective action.

H-001 has no gateway, policy service, broker, trace service, or lease-fencing
claim. The external, human-authorized bootstrap harness is the temporary
authority for H-001 repository writes, Git publication, and Beads operations.
It applies the declared checks manually, records explicit intents and receipts,
and cannot perform production or autonomous effects. W-001, T-001, and S-001
replace those temporary controls with typed enforcement; this exception cannot
be reused by tenant work.
