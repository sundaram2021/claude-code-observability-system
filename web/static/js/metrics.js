// Metrics & Statistics Rendering
const MetricsView = {
  render(metrics) {
    if (!metrics) return;
    State.metrics = metrics;

    document.getElementById('m-traces').textContent = metrics.totalTraces.toLocaleString();
    document.getElementById('m-spans').textContent = `${metrics.totalSpans.toLocaleString()} spans`;

    document.getElementById('m-cost').textContent = formatUSD(metrics.totalCostUsd);
    document.getElementById('m-savings').textContent = `Saved ${formatUSD(metrics.totalCacheSavingsUsd)} cache`;

    const inp = metrics.tokens?.inputTokens || 0;
    const out = metrics.tokens?.outputTokens || 0;
    document.getElementById('m-tokens').textContent = `${formatTokens(inp)} / ${formatTokens(out)}`;
    document.getElementById('m-cache-hit').textContent = `${(metrics.cacheHitRate || 0).toFixed(1)}% Cache Hit`;

    const avgLat = formatDuration(metrics.avgLatencyMs);
    const p95Lat = formatDuration(metrics.p95LatencyMs);
    document.getElementById('m-latency').textContent = `${avgLat} / ${p95Lat}`;

    const totalToolCalls = (metrics.tools || []).reduce((acc, t) => acc + t.callCount, 0);
    document.getElementById('m-tools').textContent = totalToolCalls.toLocaleString();
    document.getElementById('m-tool-types').textContent = `${(metrics.tools || []).length} distinct tools`;

    document.getElementById('m-errors').textContent = (metrics.errorCount || 0).toLocaleString();
    document.getElementById('m-error-rate').textContent = `${(metrics.errorRate || 0).toFixed(1)}% failure rate`;

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

