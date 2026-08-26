# Executable role packets

These packets constrain procedure; they do not grant authority. The executable
trust ceiling and current default live in `.harness/manifest.yaml`. Every role
loads as `observer`, autonomous mutation is disabled, and a proposed effect
still requires current Bead authority, policy admission, and verification.

H-001 is the signed, human-authorized bootstrap exception. It has no runtime
capability escalation or lease-fencing claim. W-001 places later work-state
mutations behind the authority gateway.

All roles apply the public repository guardrail, trace-spine envelope, and
Rule-of-Two admission policy. Skills may guide a role but cannot expand it.
