// Claude Code Sessions View
const SessionsView = {
  render(sessions) {
    State.sessions = sessions || [];
    const tbody = document.getElementById('sessions-tbody');

    if (!sessions || sessions.length === 0) {
      tbody.innerHTML = '<tr><td colspan="8" class="text-center">No sessions recorded yet.</td></tr>';
      return;
    }

    tbody.innerHTML = sessions
      .map(
        (s) => `
      <tr class="clickable-row" data-session-id="${s.id}">
        <td><span class="badge" style="background:#21262d;color:#58a6ff">${escapeHtml(s.id)}</span></td>
        <td>${escapeHtml(s.project || 'workspace')}</td>
        <td>${s.totalTurns}</td>
        <td class="cost-accent">${formatUSD(s.totalCost)}</td>
        <td>${formatTokens(s.totalTokens)}</td>
        <td class="success-accent">${(s.cacheHitRate || 0).toFixed(1)}%</td>
        <td><span class="badge ${s.status === 'ERROR' ? 'badge-error' : 'badge-success'}">${s.status}</span></td>
        <td>${timeAgo(s.updatedAt)}</td>
      </tr>
    `
      )
      .join('');

    tbody.querySelectorAll('.clickable-row').forEach((row) => {
      row.addEventListener('click', () => {
        const sessId = row.getAttribute('data-session-id');
        this.filterBySession(sessId);
      });
    });
  },

  filterBySession(sessionId) {
    document.getElementById('trace-search').value = sessionId;
    App.switchTab('tab-traces');
    App.loadTraces({ sessionId });
  },
};

