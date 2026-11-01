// Waterfall Trace Visualizer & Hierarchical Thread Inspector
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

  cleanOutput(raw) {
    if (!raw) return '';
    try {
      const p = JSON.parse(raw);
      if (typeof p === 'string') return p;
      if (p.title) return p.title + (p.summary ? ' — ' + p.summary : '');
      return p.message || JSON.stringify(p);
    } catch { return raw.replace(/\\n/g, '\n').trim(); }
  },

  renderTraceCard(trace) {
    const maxDur = Math.max(trace.durationMs, 1);
    const spans = trace.spans || [];
    const statusClass = trace.status === 'ERROR' ? 'badge-error' : 'badge-success';
    const displayName = trace.userPrompt || trace.name || 'Claude Code Execution';

    const tags = (trace.tags || []).map((t) => `<span class="badge" style="background:#21262d;color:#d97757">#${escapeHtml(t)}</span>`).join(' ');
    const scores = (trace.scores || []).map((s) => `<span class="badge" style="background:rgba(236,72,153,0.15);color:#ec4899">Score ${escapeHtml(s.name)}: ${s.value}</span>`).join(' ');
    const mult = trace.cacheMultiplier > 0 ? `<span class="badge" style="background:#1b4721;color:#3fb950">${trace.cacheMultiplier.toFixed(1)}x ROI</span>` : '';
    const speed = trace.tokensPerSec > 0 ? `<span class="badge" style="background:#21262d;color:#d97757">${trace.tokensPerSec.toFixed(0)} tok/s</span>` : '';
    const ttft = trace.avgTtftMs > 0 ? `<span class="badge" style="background:#21262d;color:#8b949e">${trace.avgTtftMs.toFixed(0)}ms TTFT</span>` : '';

    let ioHtml = '';
    if (trace.userPrompt || trace.userOutput) {
      const pRow = trace.userPrompt ? `<div class="io-row"><span class="io-badge user">USER</span><span class="io-text" title="${escapeHtml(trace.userPrompt)}">${escapeHtml(trace.userPrompt)}</span></div>` : '';
      const cleanOut = this.cleanOutput(trace.userOutput);
      const oRow = cleanOut ? `<div class="io-row"><span class="io-badge assistant">CLAUDE</span><span class="io-text output" title="${escapeHtml(cleanOut)}">${escapeHtml(cleanOut)}</span></div>` : '';
      ioHtml = `<div class="trace-io-wrap">${pRow}${oRow}</div>`;
    }
    const flatNodes = this.buildThreadNodes(spans);

    return `
      <div class="trace-card" data-trace-id="${trace.id}">
        <div class="trace-header">
          <div class="trace-title">
            <span class="badge ${statusClass}">${trace.status}</span>
            <span title="${escapeHtml(displayName)}">${escapeHtml(displayName.length > 70 ? displayName.slice(0, 67) + '...' : displayName)}</span>
            ${trace.project ? `<span class="badge" style="background:#21262d;color:#8b949e">${escapeHtml(trace.project)}</span>` : ''}
            ${tags} ${scores} ${mult} ${speed} ${ttft}
          </div>
          <div class="trace-meta">
            <span>${formatDuration(trace.durationMs)}</span><span class="cost-accent">${formatUSD(trace.totalCost)}</span>
            <span>${formatTokens(trace.totalTokens?.totalTokens || 0)} toks</span><span>${flatNodes.length} spans</span>
            <button class="btn btn-secondary btn-score" data-trace-id="${trace.id}" style="padding:2px 8px;font-size:11px">+ Score</button>
            <span>${timeAgo(trace.startTime)}</span>
          </div>
        </div>
        ${ioHtml}
        <div class="trace-sub-bar">
          <div style="display:flex;align-items:center;gap:14px">${this.renderTokenBar(trace.totalTokens)}${this.renderContextGauge(trace.contextSaturationPct || 0)}</div>
          <div style="font-family:var(--font-mono);font-size:11px;color:var(--text-muted)">Session: ${escapeHtml((trace.sessionId || trace.id).slice(0, 12))}...</div>
        </div>
        <div class="trace-body">${flatNodes.map((n) => this.renderTreeNode(n, trace, maxDur)).join('')}</div>
      </div>
    `;
  },

  buildThreadNodes(spans) {
    if (!spans || spans.length === 0) return [];
    // Deslopify: filter out 0ms internal hook and plumbing events
    const cleanSpans = spans.filter((s) => {
      const n = s.name || '';
      if (n.startsWith('Event: hook_') || n.startsWith('Event: managed_settings_') || n.startsWith('Event: plugin_loaded')) return false;
      if (n === 'Event: user_prompt') return false;
      return true;
    });
    const map = new Map(), roots = [], nodes = [];
    cleanSpans.forEach((s) => map.set(s.id, { ...s, children: [] }));
    cleanSpans.forEach((s) => {
      const node = map.get(s.id);
      if (s.parentId && map.has(s.parentId)) map.get(s.parentId).children.push(node);
      else roots.push(node);
    });
    const walk = (item, depth, isLast, openGuides) => {
      nodes.push({ item, depth, isLast, openGuides: [...openGuides] });
      const len = item.children ? item.children.length : 0;
      (item.children || []).forEach((c, idx) => walk(c, depth + 1, idx === len - 1, [...openGuides, !isLast]));
    };
    roots.forEach((r, idx) => walk(r, 0, idx === roots.length - 1, []));
    return nodes;
  },

  renderTreeNode({ item, depth, isLast, openGuides }, trace, maxDur) {
    const isTool = item.kind === 'TOOL' || item.tool;
    const isGen = item.kind === 'GENERATION' || (item.model && !isTool);
    const isErr = item.status === 'ERROR';
    const barClass = isErr ? 'error' : isGen ? 'gen' : isTool ? 'tool' : 'event';
    const widthPct = Math.max((item.durationMs / maxDur) * 100, 2);

    let iconHtml = '<span class="node-pill-icon">#</span>';
    if (isErr) iconHtml = '<span class="node-dot-err"></span>';
    else if (isGen) iconHtml = '<span class="node-dot-claude"></span>';
    else if (isTool) iconHtml = '<span class="node-dot-tool"></span>';
    else if (item.name.includes('Assistant')) iconHtml = '<span class="node-dot-event"></span>';

    let badges = '';
    if (isTool && item.tool?.name) badges = `<span class="badge" style="background:#38bdf820;color:#38bdf8">tool:${escapeHtml(item.tool.name)}</span>`;
    else if (item.model) badges = `<span class="badge" style="background:#d9775720;color:#d97757">${escapeHtml(item.model)}</span>`;
    if (isErr) badges += ' <span class="badge badge-error">ERROR</span>';

    const u = item.tokens;
    const tokStr = u && u.totalTokens > 0 ? `<span class="meta-flow">${formatTokens(u.inputTokens || 0)} → ${formatTokens(u.outputTokens || 0)} (Σ ${formatTokens(u.totalTokens)})</span>` : '';
    const costStr = item.costUsd > 0 ? `<span class="meta-cost">${formatUSD(item.costUsd)}</span>` : '';
    const cmdStr = item.tool?.input ? `<span class="meta-dim" title="${escapeHtml(item.tool.input)}">${escapeHtml(item.tool.input.slice(0, 35))}...</span>` : '';

    let guidesHtml = '';
    for (let i = 0; i < depth; i++) {
      const leftPx = i * 16 + 8;
      if (i === depth - 1) {
        guidesHtml += `<div class="tree-guide-line ${isLast ? 'last-child' : ''}" style="left:${leftPx}px"></div><div class="tree-guide-elbow" style="left:${leftPx}px"></div>`;
      } else if (openGuides[i]) {
        guidesHtml += `<div class="tree-guide-line" style="left:${leftPx}px"></div>`;
      }
    }

    return `
      <div class="tree-row" data-span-id="${item.id}" data-trace-id="${trace.id}">
        <div class="tree-branch-container" style="width:${depth * 16}px">${guidesHtml}</div>
        <div class="node-icon-wrap">${iconHtml}</div>
        <div class="tree-content">
          <div class="tree-title-line"><span class="tree-name">${escapeHtml(item.name.replace('claude_code.', ''))}</span>${badges}</div>
          <div class="tree-meta-sub"><span class="meta-duration">${formatDuration(item.durationMs)}</span>${tokStr ? `<span>·</span> ${tokStr}` : ''}${costStr ? `<span>·</span> ${costStr}` : ''}${cmdStr ? `<span>·</span> ${cmdStr}` : ''}</div>
        </div>
        <div class="tree-bar-box" title="${formatDuration(item.durationMs)}"><div class="tree-bar ${barClass}" style="width:${widthPct}%"></div></div>
      </div>
    `;
  },

  renderTokenBar(u) {
    if (!u || !u.totalTokens) return '';
    const tot = Math.max(u.totalTokens, 1);
    return `
      <div style="display:flex;align-items:center;gap:6px" title="Read: ${u.cacheReadInputTokens || 0}, Create: ${u.cacheCreationInputTokens || 0}, In: ${u.inputTokens || 0}, Out: ${u.outputTokens || 0}">
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
    return `<div class="context-gauge" title="Context Saturation against 200k limit"><span>Context: ${pct.toFixed(1)}%</span><div class="gauge-bar"><div class="gauge-fill ${cls}" style="width:${Math.min(pct, 100)}%"></div></div></div>`;
  },

  attachEventListeners() {
    document.querySelectorAll('.tree-row').forEach((row) => {
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
    const toolHtml = span.tool ? `<div style="margin-bottom:12px"><h4>Tool Execution (${escapeHtml(span.tool.name)})</h4><pre class="code-block"><code>${escapeHtml(span.tool.input || '(empty)')}\n-> ${escapeHtml(span.tool.output || '(no output)')}</code></pre></div>` : '';
    const perfInfo = span.ttftMs ? `<div><strong>TTFT / Speed:</strong> ${span.ttftMs.toFixed(0)}ms (${(span.tokensPerSec || 0).toFixed(0)} tok/s)</div>` : '';
    document.getElementById('modal-body').innerHTML = `
      <div style="display:grid;grid-template-columns:1fr 1fr;gap:8px;margin-bottom:12px">
        <div><strong>Kind:</strong> ${span.kind}</div><div><strong>Status:</strong> <span class="badge ${span.status === 'ERROR' ? 'badge-error' : 'badge-success'}">${span.status}</span></div>
        <div><strong>Duration:</strong> ${formatDuration(span.durationMs)}</div><div><strong>Cost:</strong> ${formatUSD(span.costUsd)}</div>
        <div><strong>Model:</strong> ${span.model || 'N/A'}</div><div><strong>Tokens:</strong> In: ${span.tokens?.inputTokens || 0}, Out: ${span.tokens?.outputTokens || 0}, Cache: ${span.tokens?.cacheReadInputTokens || 0}</div>${perfInfo}
      </div>${toolHtml}<h4>Raw Span Payload</h4><pre class="code-block"><code>${escapeHtml(JSON.stringify(span, null, 2))}</code></pre>`;
    document.getElementById('span-modal').classList.remove('hidden');
  },
};
