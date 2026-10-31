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
    const displayName = trace.userPrompt || trace.name || 'Claude Code Execution';

    const tagsHtml = (trace.tags || []).map((t) => `<span class="badge" style="background:#21262d;color:#a371f7">#${escapeHtml(t)}</span>`).join(' ');
    const scoresHtml = (trace.scores || []).map((s) => `<span class="badge" style="background:rgba(210,153,34,0.15);color:#d29922">★ ${escapeHtml(s.name)}: ${s.value}</span>`).join(' ');
    const multHtml = trace.cacheMultiplier > 0 ? `<span class="badge" style="background:#1b4721;color:#3fb950">${trace.cacheMultiplier.toFixed(1)}x ROI</span>` : '';
    const speedHtml = trace.tokensPerSec > 0 ? `<span class="badge" style="background:#21262d;color:#58a6ff">⚡ ${trace.tokensPerSec.toFixed(0)} tok/s</span>` : '';
    const ttftHtml = trace.avgTtftMs > 0 ? `<span class="badge" style="background:#21262d;color:#8b949e">⏱️ ${trace.avgTtftMs.toFixed(0)}ms TTFT</span>` : '';
    const healedHtml = trace.selfCorrected ? `<span class="badge" style="background:rgba(56,189,248,0.15);color:#38bdf8">✨ Healed</span>` : '';

    const promptBar = trace.userPrompt ? `
      <div class="trace-prompt-bar" title="${escapeHtml(trace.userPrompt)}">
        <span class="prompt-tag">💬 USER PROMPT:</span>
        <span class="prompt-text">${escapeHtml(trace.userPrompt)}</span>
      </div>` : '';

    return `
      <div class="trace-card" data-trace-id="${trace.id}">
        <div class="trace-header">
          <div class="trace-title">
            <span class="badge ${statusClass}">${trace.status}</span>
            <span title="${escapeHtml(displayName)}">${escapeHtml(displayName.length > 70 ? displayName.slice(0, 67) + '...' : displayName)}</span>
            ${trace.project ? `<span class="badge" style="background:#21262d;color:#8b949e">${escapeHtml(trace.project)}</span>` : ''}
            ${tagsHtml} ${scoresHtml} ${multHtml} ${speedHtml} ${ttftHtml} ${healedHtml}
          </div>
          <div class="trace-meta">
            <span>${formatDuration(trace.durationMs)}</span>
            <span class="cost-accent">${formatUSD(trace.totalCost)}</span>
            <span>${formatTokens(trace.totalTokens?.totalTokens || 0)} toks</span>
            <span>${spans.length} spans</span>
            <button class="btn btn-secondary btn-score" data-trace-id="${trace.id}" style="padding:2px 8px;font-size:11px">+ Score</button>
            <span>${timeAgo(trace.startTime)}</span>
          </div>
        </div>
        ${promptBar}
        <div class="trace-sub-bar">
          <div style="display:flex;align-items:center;gap:14px">
            ${this.renderTokenBar(trace.totalTokens)}
            ${this.renderContextGauge(trace.contextSaturationPct || 0)}
          </div>
          <div class="step-flow">${this.renderStepFlow(spans)}</div>
        </div>
        <div class="trace-body">
          ${spans.map((s) => this.renderSpanRow(s, trace, maxDur)).join('')}
        </div>
      </div>
    `;
  },

  renderTokenBar(u) {
    if (!u || !u.totalTokens) return '';
    const tot = Math.max(u.totalTokens, 1);
    return `
      <div style="display:flex;align-items:center;gap:6px" title="Cache Read: ${u.cacheReadInputTokens || 0}, Create: ${u.cacheCreationInputTokens || 0}, In: ${u.inputTokens || 0}, Out: ${u.outputTokens || 0}">
        <span style="color:var(--text-muted)">Tokens:</span>
        <div class="token-bar-container">
          <div class="seg-read" style="width:${((u.cacheReadInputTokens || 0) / tot) * 100}%"></div>
          <div class="seg-create" style="width:${((u.cacheCreationInputTokens || 0) / tot) * 100}%"></div>
          <div class="seg-input" style="width:${((u.inputTokens || 0) / tot) * 100}%"></div>
          <div class="seg-output" style="width:${((u.outputTokens || 0) / tot) * 100}%"></div>
        </div>
      </div>
    `;
  },

  renderContextGauge(pct) {
    const cls = pct > 75 ? 'danger' : pct > 50 ? 'warning' : '';
    return `
      <div class="context-gauge" title="Context Saturation against 200k limit">
        <span>Context: ${pct.toFixed(1)}%</span>
        <div class="gauge-bar"><div class="gauge-fill ${cls}" style="width:${Math.min(pct, 100)}%"></div></div>
      </div>
    `;
  },

  renderStepFlow(spans) {
    if (!spans || spans.length === 0) return '';
    const nodes = spans.slice(0, 6).map((s) => {
      const isTool = s.kind === 'TOOL' || s.tool;
      const cls = s.status === 'ERROR' ? 'error' : isTool ? 'tool' : '';
      const name = isTool && s.tool?.name ? s.tool.name : s.name.replace('Event: ', '');
      return `<span class="step-node ${cls}">${escapeHtml(name.slice(0, 15))}</span>`;
    });
    return nodes.join('<span class="step-arrow">➔</span>');
  },

  renderSpanRow(s, trace, maxDur) {
    const indent = s.parentId ? 24 : 0;
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
        <div class="span-info"><span class="span-name">${escapeHtml(s.name)}</span></div>
        <div class="span-meta">${metaBadge} ${isError ? `<span class="badge badge-error">ERR</span>` : ''}</div>
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
        this.openSpanModal(row.getAttribute('data-trace-id'), row.getAttribute('data-span-id'));
      });
    });

    document.querySelectorAll('.btn-score').forEach((btn) => {
      btn.addEventListener('click', (e) => {
        e.stopPropagation();
        document.getElementById('score-trace-id').value = btn.getAttribute('data-trace-id');
        document.getElementById('score-modal').classList.remove('hidden');
      });
    });
  },

  openSpanModal(traceId, spanId) {
    const trace = State.traces.find((t) => t.id === traceId);
    const span = trace ? (trace.spans || []).find((s) => s.id === spanId) : null;
    if (!span) return;

    document.getElementById('modal-title').textContent = span.name;
    const toolHtml = span.tool ? `
      <div style="margin-bottom:12px">
        <h4>Tool Execution (${escapeHtml(span.tool.name)})</h4>
        <pre class="code-block"><code>${escapeHtml(span.tool.input || '(empty)')}\n➔ ${escapeHtml(span.tool.output || '(no output)')}</code></pre>
      </div>` : '';

    const perfInfo = span.ttftMs ? `<div><strong>TTFT / Speed:</strong> ${span.ttftMs.toFixed(0)}ms (${(span.tokensPerSec || 0).toFixed(0)} tok/s)</div>` : '';
    document.getElementById('modal-body').innerHTML = `
      <div style="display:grid;grid-template-columns:1fr 1fr;gap:8px;margin-bottom:12px">
        <div><strong>Kind:</strong> ${span.kind}</div>
        <div><strong>Status:</strong> <span class="badge ${span.status === 'ERROR' ? 'badge-error' : 'badge-success'}">${span.status}</span></div>
        <div><strong>Duration:</strong> ${formatDuration(span.durationMs)}</div>
        <div><strong>Cost:</strong> ${formatUSD(span.costUsd)}</div>
        <div><strong>Model:</strong> ${span.model || 'N/A'}</div>
        <div><strong>Tokens:</strong> In: ${span.tokens?.inputTokens || 0}, Out: ${span.tokens?.outputTokens || 0}, Cache: ${span.tokens?.cacheReadInputTokens || 0}</div>
        ${perfInfo}
      </div>
      ${toolHtml}
      <h4>Raw Span Payload</h4>
      <pre class="code-block"><code>${escapeHtml(JSON.stringify(span, null, 2))}</code></pre>
    `;
    document.getElementById('span-modal').classList.remove('hidden');
  },
};
