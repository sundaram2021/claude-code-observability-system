// Metrics & Statistics Rendering
const MetricsView = {
  render(metrics) {
    if (!metrics) return;
    State.metrics = metrics;

    document.getElementById('m-traces').textContent = metrics.totalTraces.toLocaleString();
    document.getElementById('m-spans').textContent = `${metrics.totalSpans.toLocaleString()} spans`;

    document.getElementById('m-cost').textContent = formatUSD(metrics.totalCostUsd);
    document.getElementById('m-savings').textContent = `Saved ${formatUSD(metrics.totalCacheSavingsUsd)} cache`;

    const roiMult = metrics.cacheMultiplier > 0 ? `${metrics.cacheMultiplier.toFixed(1)}x ROI` : '1.0x ROI';
    document.getElementById('m-roi').textContent = roiMult;
    document.getElementById('m-cache-hit').textContent = `${(metrics.cacheHitRate || 0).toFixed(1)}% Cache Hit`;

    const speed = metrics.avgTokensPerSec > 0 ? `${metrics.avgTokensPerSec.toFixed(1)} tok/s` : `${formatDuration(metrics.avgLatencyMs)}`;
    const ttftText = metrics.avgTtftMs > 0 ? `${metrics.avgTtftMs.toFixed(0)}ms TTFT` : `P95: ${formatDuration(metrics.p95LatencyMs)}`;
    document.getElementById('m-speed').textContent = speed;
    document.getElementById('m-ttft').textContent = ttftText;

    const ctxSat = (metrics.avgContextSaturation || 0).toFixed(1);
    document.getElementById('m-context').textContent = `${ctxSat}%`;
    const ctxToks = metrics.tokens?.contextTokens || (metrics.tokens?.inputTokens || 0) + (metrics.tokens?.cacheReadInputTokens || 0);
    document.getElementById('m-context-toks').textContent = `${formatTokens(ctxToks)} / 200k tokens`;

    const healRate = metrics.errorCount > 0 ? `${(metrics.selfCorrectionRate || 0).toFixed(0)}% healed` : '100% resilient';
    document.getElementById('m-resilience').textContent = healRate;
    document.getElementById('m-error-rate').textContent = `${(metrics.errorRate || 0).toFixed(1)}% failure rate (${metrics.errorCount || 0} errs)`;

    this.renderToolStats(metrics.tools);
    this.renderModelStats(metrics.models);
  },

  renderToolStats(tools) {
    const tbody = document.getElementById('tool-stats-tbody');
    if (!tools || tools.length === 0) {
      tbody.innerHTML = '<tr><td colspan="4" class="text-center">No tool data recorded</td></tr>';
      return;
    }

    tbody.innerHTML = tools
      .map(
        (t) => `
      <tr>
        <td><span class="badge badge-tool">${escapeHtml(t.name)}</span></td>
        <td>${t.callCount.toLocaleString()}</td>
        <td>${formatDuration(t.avgDurationMs)}</td>
        <td class="${t.errorCount > 0 ? 'error-accent' : ''}">${t.errorCount}</td>
      </tr>
    `
      )
      .join('');
  },

  renderModelStats(models) {
    const tbody = document.getElementById('model-stats-tbody');
    if (!models || models.length === 0) {
      tbody.innerHTML = '<tr><td colspan="4" class="text-center">No model data recorded</td></tr>';
      return;
    }

    tbody.innerHTML = models
      .map(
        (m) => `
      <tr>
        <td><span class="badge badge-model">${escapeHtml(m.model)}</span></td>
        <td>${m.calls.toLocaleString()}</td>
        <td>${formatTokens(m.totalTokens)}</td>
        <td class="cost-accent">${formatUSD(m.costUsd)}</td>
      </tr>
    `
      )
      .join('');
  },
};

