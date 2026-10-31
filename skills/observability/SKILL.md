---
name: observability
description: Inspect and monitor real-time Claude Code performance, token consumption, prompt cache ROI, context saturation, and tool execution stats.
argument-hint: [status|open|export]
allowed-tools: [Bash]
---

# Observability Skill for Claude Code

Provides instant visibility into session costs, prompt caching savings, and trace waterfalls.

## Actions

### 1. Show Status
When the user asks for status, costs, tokens, or cache hit rate:
Run `curl -s http://localhost:3456/api/v1/metrics` or run `./bin/server.exe status`.
Display:
- Total Cost ($) and Prompt Cache Savings ($)
- Cache ROI Multiplier ($Saved / $Paid)
- Time to First Token (TTFT) and Streaming Velocity (tokens/sec)
- Context Saturation percentage of the 200,000 limit

### 2. Open Live Dashboard
When the user asks to open or view the dashboard:
Run `./bin/server.exe open` (or navigate to `http://localhost:3456`).

### 3. Export Traces
Fetch `http://localhost:3456/api/v1/traces/export` and save to `claude-traces-export.json`.

