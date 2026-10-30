# Claude Code Observability Command (/observe)

You are the observability manager for Claude Code. This command controls the local, ultra-low-memory Langfuse-like observability server for Claude Code.

## Usage
The user ran `/observe` (with optional arguments: `$ARGUMENTS`).

## Instructions

1. **If argument is empty or "start"**:
   - Check if the observability server is running on `http://localhost:3000`.
   - If not running, execute `./bin/server.exe start` (or `go run ./cmd/server start`) using the bash tool.
   - Configure the current environment:
     ```bash
     export OTEL_EXPORTER_OTLP_ENDPOINT="http://localhost:3000/v1/traces"
     export OTEL_EXPORTER_OTLP_PROTOCOL="http/json"
     ```
   - Inform the user:
     > 🚀 **Claude Code Observability is active at http://localhost:3000**
     > - All tool executions (`bash`, `file_edit`, `glob`, `view`), LLM generations, and prompt cache savings are streaming live.
     > - Open `http://localhost:3000` to inspect waterfall traces and session analytics.

2. **If argument is "status"**:
   - Run `./bin/server.exe status` (or query `http://localhost:3000/api/v1/metrics`).
   - Display a concise summary of:
     - Total Traces & Turns
     - Total Cost & Prompt Cache Savings ($)
     - Cache Hit %
     - Tool Call counts and Error Rates

3. **If argument is "open"**:
   - Run `./bin/server.exe open` to open `http://localhost:3000` in the user's default browser.

4. **If argument is "export"**:
   - Fetch `http://localhost:3000/api/v1/traces/export` and save to `claude-traces-export.json`.
   - Confirm export path to the user.

