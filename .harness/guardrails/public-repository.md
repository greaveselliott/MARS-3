# Public repository guardrail

Every committed byte is presumed public immediately. Permitted content is
original Apache-licensed source and documentation, public attribution,
synthetic fixtures, public architecture and threat models, and redacted
evidence made of hashes, relative paths, commands, versions, bounded outcomes,
and opaque trace references.

Prohibited content includes secrets; real identities or tenant data; private
source or traces; raw prompts, completions, reasoning, tool payloads, or
provider state; databases; local configuration; developer-home paths; machine
metadata; private hostnames; unsafe active fixtures; and unlicensed copied
text. H-001 additionally rejects every binary or unscannable artifact; a later
milestone must introduce explicit asset provenance and inspection before that
restriction can change.

If a secret reaches Git, rotate it immediately, open a private security
incident, remediate history, and verify the result. A later deletion does not
undo disclosure.
