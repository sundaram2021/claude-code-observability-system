# LLM Observability System

A self-hosted observability platform for LLM-powered agents. It traces every Anthropic API call and tool execution using OpenTelemetry, stores everything in memory, and renders a live dashboard in the browser — no external services needed.

## Demo
<img width="1918" height="906" alt="image" src="https://github.com/user-attachments/assets/58110b3b-2b95-445e-96dc-92b63c922b0d" />
<img width="1901" height="870" alt="image" src="https://github.com/user-attachments/assets/37d08355-b5d0-43c4-8761-210b3b605bd7" />

## Architecture

```mermaid
graph TD
    subgraph Browser
        HTML[index.html]
        JS[app.js]
        CSS[style.css]
        HTML --> JS
        HTML --> CSS
    end

    subgraph Express Server
        INDEX[index.ts]
        INST[instrumentation.ts]
        AGENT[agent.ts]
        LOGGER[logger.ts]
    end

    subgraph OpenTelemetry
        SDK[NodeSDK]
        EXPORTER[InMemorySpanExporter]
        PROCESSOR[SimpleSpanProcessor]
        ANTHRO_INST[AnthropicInstrumentation]
    end

    subgraph External
        ANTHROPIC[Anthropic API]
    end

    JS -- "GET /api/telemetry" --> INDEX
    JS -- "POST /api/chat" --> INDEX
    INDEX -- serves static files --> HTML

    INDEX -- imports --> INST
    INDEX -- imports --> AGENT
    AGENT -- imports --> LOGGER
    LOGGER -- pushes logs --> INST

    INST -- creates --> SDK
    SDK -- uses --> PROCESSOR
    PROCESSOR -- writes to --> EXPORTER
    SDK -- registers --> ANTHRO_INST

    AGENT -- "messages.create()" --> ANTHROPIC
    ANTHRO_INST -- auto-traces --> ANTHROPIC

    INDEX -- "reads spans from" --> EXPORTER
    INDEX -- "reads logs from" --> INST
```


### Backend (`src/`)

| File | What it does |
|------|-------------|
| `src/index.ts` | Express server entry point. Serves the static frontend, exposes `POST /api/chat` to send messages to Claude, and `GET /api/telemetry` to return collected spans and logs as JSON. |
| `src/instrumentation.ts` | Initializes the OpenTelemetry SDK with an in-memory span exporter and the Anthropic auto-instrumentation. Every Anthropic API call gets traced automatically. Also holds the shared `memoryLogStore` array. |
| `src/services/agent.ts` | The chat agent. Sends user messages to Claude with two tools (`get_date`, `calculator`), handles the tool-use loop, and wraps each tool execution in its own OTel span. |
| `src/utils/logger.ts` | Simple logger that pushes structured log entries (with trace correlation) into the shared `memoryLogStore` so they show up on the dashboard. |


## Setup

```bash
# 1. Clone and install
git clone <repo-url>
cd llm-observability-system
pnpm install

# 2. Add your Anthropic API key
echo ANTHROPIC_API_KEY=sk-ant-... > .env.local

# 3. Start the server
pnpm dev

# 4. Open the dashboard
# http://localhost:3000

# 5. Send a chat message (from another terminal)
curl -X POST http://localhost:3000/api/chat \
  -H "Content-Type: application/json" \
  -d '{"message": "What is 42 * 17?"}'

# 6. Watch the dashboard update with traces and spans
```

## Concepts
<img width="1577" height="806" alt="image" src="https://github.com/user-attachments/assets/433293a1-7808-42e0-9499-433229211c3b" />
