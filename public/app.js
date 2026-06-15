const PRICING = {
  'claude-haiku-4-5-20251001': { input: 0.80, output: 4.00 },
  'claude-3-5-haiku-latest': { input: 0.80, output: 4.00 },
  'claude-3-5-sonnet-latest': { input: 3.00, output: 15.00 },
  'claude-sonnet-4-20250514': { input: 3.00, output: 15.00 },
  'claude-3-5-haiku-20241022': { input: 0.80, output: 4.00 },
  default: { input: 3.00, output: 15.00 },
};

function getCost(model, inputTokens, outputTokens) {
  const rates = PRICING[model] || PRICING.default;
  return (inputTokens * rates.input + outputTokens * rates.output) / 1_000_000;
}

document.addEventListener('DOMContentLoaded', () => {
  document.getElementById('refresh-btn').addEventListener('click', loadData);
  loadData();
  setInterval(loadData, 5000);
});

async function loadData() {
  try {
    const res = await fetch('/api/telemetry');
    const data = await res.json();
    renderDashboard(data.spans, data.logs);
  } catch (error) {
    console.error("Failed to fetch telemetry:", error);
  }
}

function renderDashboard(spans, logs) {
  let totalLatency = 0;
  let rootSpans = 0;
  let totalInputTokens = 0;
  let totalOutputTokens = 0;
  let totalCost = 0;
  let errorCount = logs.filter(l => l.level === 'ERROR').length;

  spans.forEach(span => {
    if (!span.parentId) {
      rootSpans++;
      totalLatency += span.durationMs;
    }
    if (span.attributes) {
      const input = Number(span.attributes['llm.token_count.prompt'] || span.attributes['gen_ai.usage.input_tokens'] || 0);
      const output = Number(span.attributes['llm.token_count.completion'] || span.attributes['gen_ai.usage.output_tokens'] || 0);
      if (input || output) {
        totalInputTokens += input;
        totalOutputTokens += output;
        const model = span.attributes['llm.model_name'] || span.attributes['gen_ai.request.model'] || '';
        totalCost += getCost(model, input, output);
      }
    }
  });

  document.getElementById('metric-traces').innerText = rootSpans;
  document.getElementById('metric-latency').innerText = rootSpans > 0 ? `${(totalLatency / rootSpans).toFixed(0)}ms` : '0ms';
  document.getElementById('metric-input-tokens').innerText = totalInputTokens.toLocaleString();
  document.getElementById('metric-output-tokens').innerText = totalOutputTokens.toLocaleString();
  document.getElementById('metric-cost').innerText = totalCost < 0.01 ? `$${totalCost.toFixed(4)}` : `$${totalCost.toFixed(2)}`;
  document.getElementById('metric-errors').innerText = errorCount;

  const tracesContainer = document.getElementById('traces-list');
  tracesContainer.innerHTML = '';

  const traceGroups = {};
  spans.forEach(span => {
    if (!traceGroups[span.traceId]) traceGroups[span.traceId] = [];
    traceGroups[span.traceId].push(span);
  });

  if (Object.keys(traceGroups).length === 0) {
    tracesContainer.innerHTML = '<p class="text-muted">No traces found yet. Make an API request!</p>';
  }

  for (const [traceId, groupSpans] of Object.entries(traceGroups)) {
    const maxDuration = Math.max(...groupSpans.map(s => s.durationMs));

    let traceInputTokens = 0;
    let traceOutputTokens = 0;
    let traceCost = 0;
    groupSpans.forEach(s => {
      if (s.attributes) {
        const inp = Number(s.attributes['llm.token_count.prompt'] || s.attributes['gen_ai.usage.input_tokens'] || 0);
        const out = Number(s.attributes['llm.token_count.completion'] || s.attributes['gen_ai.usage.output_tokens'] || 0);
        if (inp || out) {
          traceInputTokens += inp;
          traceOutputTokens += out;
          const m = s.attributes['llm.model_name'] || s.attributes['gen_ai.request.model'] || '';
          traceCost += getCost(m, inp, out);
        }
      }
    });

    groupSpans.sort((a, b) => {
      const aTime = a.startTime[0] * 1000 + a.startTime[1] / 1e6;
      const bTime = b.startTime[0] * 1000 + b.startTime[1] / 1e6;
      return aTime - bTime;
    });

    const groupDiv = document.createElement('div');
    groupDiv.className = 'trace-group';

    let statsHtml = '';
    if (traceInputTokens || traceOutputTokens) {
      statsHtml = `
        <div class="trace-stats">
          <div class="stat"><span class="stat-label">In:</span><span class="stat-value tokens">${traceInputTokens.toLocaleString()}</span></div>
          <div class="stat"><span class="stat-label">Out:</span><span class="stat-value tokens">${traceOutputTokens.toLocaleString()}</span></div>
          <div class="stat"><span class="stat-label">Cost:</span><span class="stat-value cost">${traceCost < 0.01 ? '$' + traceCost.toFixed(4) : '$' + traceCost.toFixed(2)}</span></div>
        </div>
      `;
    }

    groupDiv.innerHTML = `
      <div class="trace-header">
        <span>Trace ID: ${traceId}</span>
        <span>${groupSpans.length} Spans</span>
      </div>
      ${statsHtml}
    `;

    groupSpans.forEach(span => {
      const widthPct = maxDuration > 0 ? (span.durationMs / maxDuration) * 100 : 0;
      const indent = span.parentId ? 'margin-left: 20px; border-left: 2px solid var(--border);' : '';

      const toolName = span.attributes['gen_ai.tool.name'] || '';
      const model = span.attributes['llm.model_name'] || span.attributes['gen_ai.request.model'] || '';
      const displayMeta = toolName ? `Tool: ${toolName}` : model ? `Model: ${model}` : '';

      const inp = Number(span.attributes['llm.token_count.prompt'] || span.attributes['gen_ai.usage.input_tokens'] || 0);
      const out = Number(span.attributes['llm.token_count.completion'] || span.attributes['gen_ai.usage.output_tokens'] || 0);
      let tokenHtml = '';
      if (inp || out) {
        const spanCost = getCost(model, inp, out);
        tokenHtml = `<div class="span-tokens"><span>↑${inp}</span><span>↓${out}</span><span style="color:var(--accent-green)">$${spanCost.toFixed(4)}</span></div>`;
      }

      const spanHtml = `
        <div class="span-row" style="${indent}">
          <div class="span-info">
            <span class="span-name">${span.name}</span>
            <span class="span-duration">${span.durationMs.toFixed(2)}ms</span>
          </div>
          ${displayMeta ? `<div class="span-meta">${displayMeta}</div>` : ''}
          ${tokenHtml}
          <div class="duration-bar" style="width: ${Math.max(widthPct, 2)}%;"></div>
        </div>
      `;
      groupDiv.innerHTML += spanHtml;
    });

    tracesContainer.appendChild(groupDiv);
  }

  const logsContainer = document.getElementById('logs-list');
  logsContainer.innerHTML = '';

  if (logs.length === 0) {
    logsContainer.innerHTML = '<p class="text-muted">Waiting for system logs...</p>';
  }

  logs.slice().reverse().forEach(log => {
    const time = new Date(log.timestamp).toLocaleTimeString();
    const traceBadge = log.traceId ? `<span class="log-trace">[${log.traceId.substring(0,8)}]</span>` : '';

    logsContainer.innerHTML += `
      <div class="log-entry">
        <span class="log-time">${time}</span>
        <span class="log-level ${log.level.toLowerCase()}">${log.level}</span>
        ${traceBadge}
        <span class="log-msg">${escapeHtml(log.message)}</span>
      </div>
    `;
  });
}

function escapeHtml(unsafe) {
  return unsafe
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;")
    .replace(/'/g, "&#039;");
}