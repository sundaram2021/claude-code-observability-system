// Waterfall Trace Visualizer & Span Inspector
const WaterfallView = {
  render(traces) {
    const container = document.getElementById('traces-list');
    if (!traces || traces.length === 0) {
      container.innerHTML = '<div class="empty-state text-center" style="padding:40px;color:var(--text-muted)">No traces recorded yet. Run a Claude Code task or click "+ Simulate Task".</div>';
      return;
    }

    container.innerHTML = traces.map((t) => this.renderTraceCard(t)).join('');
    this.attachEventListeners();
  },

  renderTraceCard(trace) {
    const maxDur = Math.max(trace.durationMs, 1);
    const spans = trace.spans || [];
    const statusClass = trace.status === 'ERROR' ? 'badge-error' : 'badge-success';

    return `
      <div class="trace-card" data-trace-id="${trace.id}">
        <div class="trace-header">
          <div class="trace-title">
            <span class="badge ${statusClass}">${trace.status}</span>
            <span>${escapeHtml(trace.name || 'Claude Code Execution')}</span>
            ${trace.project ? `<span class="badge" style="background:#21262d;color:#8b949e">${escapeHtml(trace.project)}</span>` : ''}
          </div>
          <div class="trace-meta">
            <span>${formatDuration(trace.durationMs)}</span>
            <span class="cost-accent">${formatUSD(trace.totalCost)}</span>
            <span>${formatTokens(trace.totalTokens?.totalTokens || 0)} toks</span>
            <span>${spans.length} spans</span>
            <span>${timeAgo(trace.startTime)}</span>
          </div>
        </div>
        <div class="trace-body">
          ${spans.map((s) => this.renderSpanRow(s, trace, maxDur)).join('')}
        </div>
      </div>
    `;
  },

  renderSpanRow(s, trace, maxDur) {
    const isRoot = !s.parentId;
    const indent = isRoot ? 0 : 24;
    const widthPct = Math.max((s.durationMs / maxDur) * 100, 2);
    const isTool = s.kind === 'TOOL' || !!s.tool;
    const isError = s.status === 'ERROR';
    const barClass = isError ? 'error' : isTool ? 'tool' : '';

    let metaBadge = '';
    if (isTool && s.tool?.name) {
      metaBadge = `<span class="badge badge-tool">tool:${escapeHtml(s.tool.name)}</span>`;
    } else if (s.model) {
      metaBadge = `<span class="badge badge-model">${escapeHtml(s.model)}</span>`;
    }

    return `
      <div class="span-row" data-span-id="${s.id}" data-trace-id="${trace.id}" style="padding-left:${indent + 10}px">
        <div class="span-info">
          <span class="span-name">${escapeHtml(s.name)}</span>
        </div>
        <div class="span-meta">
          ${metaBadge}
          ${isError ? `<span class="badge badge-error">ERR</span>` : ''}
        </div>
        <div class="span-metrics">
          ${s.tokens?.totalTokens ? `<span>${formatTokens(s.tokens.totalTokens)} t</span>` : ''}
          ${s.costUsd > 0 ? `<span class="cost-accent">${formatUSD(s.costUsd)}</span>` : ''}
        </div>
        <div class="span-bar-container">
          <div class="span-bar ${barClass}" style="width:${widthPct}%"></div>
          <span class="span-duration-label">${formatDuration(s.durationMs)}</span>
        </div>
      </div>
    `;
  },

  attachEventListeners() {
    document.querySelectorAll('.span-row').forEach((row) => {
      row.addEventListener('click', (e) => {
        e.stopPropagation();
        const spanId = row.getAttribute('data-span-id');
        const traceId = row.getAttribute('data-trace-id');
        this.openSpanModal(traceId, spanId);
      });
    });
  },

  openSpanModal(traceId, spanId) {
    const trace = State.traces.find((t) => t.id === traceId);
    if (!trace) return;
    const span = (trace.spans || []).find((s) => s.id === spanId);
    if (!span) return;

    document.getElementById('modal-title').textContent = span.name;
    const body = document.getElementById('modal-body');

    let toolHtml = '';
    if (span.tool) {
      toolHtml = `
        <div style="margin-bottom:14px">
          <h4>Tool Execution (${escapeHtml(span.tool.name)})</h4>
          <p style="margin:4px 0 2px 0;color:var(--text-muted)">Input:</p>
          <pre class="code-block"><code>${escapeHtml(span.tool.input || '(empty)')}</code></pre>
          <p style="margin:4px 0 2px 0;color:var(--text-muted)">Output:</p>
          <pre class="code-block"><code>${escapeHtml(span.tool.output || '(no output)')}</code></pre>
        </div>
      `;
    }

    body.innerHTML = `
      <div style="display:grid;grid-template-columns:1fr 1fr;gap:10px;margin-bottom:14px">
        <div><strong>Kind:</strong> ${span.kind}</div>
        <div><strong>Status:</strong> <span class="badge ${span.status === 'ERROR' ? 'badge-error' : 'badge-success'}">${span.status}</span></div>
        <div><strong>Duration:</strong> ${formatDuration(span.durationMs)}</div>
        <div><strong>Cost:</strong> ${formatUSD(span.costUsd)}</div>
        <div><strong>Model:</strong> ${span.model || 'N/A'}</div>
        <div><strong>Tokens:</strong> In: ${span.tokens?.inputTokens || 0}, Out: ${span.tokens?.outputTokens || 0}, Cache: ${span.tokens?.cacheReadInputTokens || 0}</div>
      </div>
      ${toolHtml}
      <h4>Raw Span Payload</h4>
      <pre class="code-block"><code>${escapeHtml(JSON.stringify(span, null, 2))}</code></pre>
    `;

    document.getElementById('span-modal').classList.remove('hidden');
  },
};

