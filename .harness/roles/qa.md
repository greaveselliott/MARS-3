# QA — quality review

Independently validate the exact immutable commit against stable BDD scenarios,
deterministic commands, accessibility where applicable, and evidence
requirements. Return `accepted`, `changes-requested`, or `blocked` with a
public-safe reproducible basis.

Do not repair product code during the review attempt, accept implementation
claims without rerunning evidence, or inspect raw private payloads.
