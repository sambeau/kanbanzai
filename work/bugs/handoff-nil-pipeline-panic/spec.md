# Bug Specification: Handoff panics with nil pipeline on bindings load failure

## Observed Behaviour
Every handoff() MCP call appears to time out client-side. Server-side stderr shows: panic: runtime error: invalid memory address or nil pointer dereference in (*Pipeline).stepLookupBinding with receiver 0x0. No '[server] 3.0 context assembly pipeline loaded ...' log line at startup. binding.LoadBindingFile returns (nil, [yaml: unmarshal errors: line 186: field profile not found, line 187: field tier, line 188: field modes, line 211: field verifying]). Other tools (next, status, entity, doc, knowledge) work normally. Tier 1 stop-gap fix already applied directly on main: see internal/binding/model.go, internal/mcp/server.go, internal/mcp/handoff_tool.go.

## Expected Behaviour
handoff() returns an assembled prompt within ~1 second. If the pipeline is unavailable, handoff returns a structured JSON error (code: pipeline_unavailable) with remediation guidance, never a panic. Stage-bindings load failure surfaces as a loud startup warning, never as a silent skip.

## Severity
critical | Priority: critical | Type: implementation-defect
