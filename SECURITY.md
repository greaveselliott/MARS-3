# Security policy

MARS-3 is building a security boundary; the current foundation is not yet a
production-ready one. Only the latest commit on `main` receives security
fixes during this pre-release phase.

## Report privately

Do not open a public issue for a suspected vulnerability. Use GitHub's
[private vulnerability reporting](https://github.com/greaveselliott/MARS-3/security/advisories/new)
for confidential coordination. Include the affected commit, a minimal
reproduction using synthetic data, impact, and any suggested mitigation.
Never include a real credential, customer record, private source file, raw
prompt, provider session, or production trace in the report.

We will acknowledge a report, classify its scope and ownership, coordinate a
fix, and agree on disclosure timing. A reporter should not test against systems
or data they do not own or have explicit authorization to assess.

## If a secret reaches Git

Deleting the value in a later commit is insufficient. Maintainers must:

1. rotate or revoke the credential immediately;
2. open a private security incident and preserve only redacted evidence;
3. remediate Git history and every replica or published artifact;
4. rerun secret scanning and verify that the old credential cannot authenticate;
5. disclose the incident according to the coordinated response decision.

Repository evidence may contain hashes, relative paths, command names,
versions, outcomes, and opaque trace references. It must never contain the raw
payload behind those references.
