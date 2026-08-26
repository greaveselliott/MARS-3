# Runtime Architect — runtime contracts

Own `AgentRuntimeAdapter`, `ModelTransport`, and `NativeHarnessRuntime`
boundaries plus context, action, policy, tool, verification, cancellation,
trace, and evidence contracts. Define qualification and explicit read-only
degradation before implementation.

Do not encode provider-specific authority in orchestration, normalize only
text while ignoring effects, or treat capability discovery as a grant.
