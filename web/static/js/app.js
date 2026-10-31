// Main Application Entrypoint & Tab Orchestrator
const App = {
  async init() {
    this.setupTabs();
    this.setupFilters();
    this.setupModals();
    Simulator.init();

    document.getElementById('btn-refresh').addEventListener('click', () => this.refreshAll());
    document.getElementById('btn-clear').addEventListener('click', async () => {
      if (confirm('Clear all recorded traces?')) {
        await API.clearTraces();
        await this.refreshAll();
      }
    });

    const btnExport = document.getElementById('btn-export');
    if (btnExport) {
      btnExport.addEventListener('click', () => {
        window.open('/api/v1/traces/export', '_blank');
      });
    }

    API.connectSSE(
      () => this.refreshAll(),
      (connected) => {
        const pill = document.getElementById('live-indicator');
        const status = document.getElementById('live-status');
        if (connected) {
          pill.classList.add('connected');
          status.textContent = 'Live SSE';
        } else {
          pill.classList.remove('connected');
          status.textContent = 'Reconnecting';
        }
      }
    );

    await this.refreshAll();
  },

  setupTabs() {
    document.querySelectorAll('.nav-tab').forEach((tab) => {
      tab.addEventListener('click', () => {
        const targetId = tab.getAttribute('data-tab');
        this.switchTab(targetId);
      });
    });
  },

  switchTab(targetId) {
    document.querySelectorAll('.nav-tab').forEach((t) => {
      t.classList.toggle('active', t.getAttribute('data-tab') === targetId);
    });
    document.querySelectorAll('.tab-pane').forEach((p) => {
      p.classList.toggle('active', p.id === targetId);
    });
  },

  setupFilters() {
    const searchInput = document.getElementById('trace-search');
    const statusSelect = document.getElementById('status-filter');

    let debounceTimer;
    const triggerSearch = () => {
      clearTimeout(debounceTimer);
      debounceTimer = setTimeout(() => {
        this.loadTraces({
          search: searchInput.value.trim(),
          status: statusSelect.value,
        });
      }, 250);
    };

    searchInput.addEventListener('input', triggerSearch);
    statusSelect.addEventListener('change', triggerSearch);

    document.querySelectorAll('.filter-pills .pill').forEach((pill) => {
      pill.addEventListener('click', () => {
        document.querySelectorAll('.filter-pills .pill').forEach((p) => p.classList.remove('active'));
        pill.classList.add('active');
        this.filterTracesByPill(pill.getAttribute('data-filter'));
      });
    });
  },

  filterTracesByPill(filter) {
    let list = State.traces || [];
    if (filter === 'tool') {
      list = list.filter((t) => (t.spans || []).some((s) => s.kind === 'TOOL' || s.tool));
    } else if (filter === 'cache') {
      list = list.filter((t) => {
        const u = t.totalTokens;
        return u && (u.cacheReadInputTokens / Math.max(u.inputTokens + u.cacheReadInputTokens, 1)) >= 0.8;
      });
    } else if (filter === 'slow') {
      list = list.filter((t) => t.durationMs >= 3000);
    }
    WaterfallView.render(list);
  },

  setupModals() {
    const spanModal = document.getElementById('span-modal');
    document.getElementById('modal-close').addEventListener('click', () => {
      spanModal.classList.add('hidden');
    });

    const scoreModal = document.getElementById('score-modal');
    const scoreClose = document.getElementById('score-close');
    if (scoreClose) {
      scoreClose.addEventListener('click', () => scoreModal.classList.add('hidden'));
    }

    const scoreForm = document.getElementById('score-form');
    if (scoreForm) {
      scoreForm.addEventListener('submit', async (e) => {
        e.preventDefault();
        const traceId = document.getElementById('score-trace-id').value;
        const name = document.getElementById('score-name').value.trim();
        const value = parseFloat(document.getElementById('score-value').value);
        const comment = document.getElementById('score-comment').value.trim();

        try {
          await API.createScore({ traceId, name, value, comment });
          scoreModal.classList.add('hidden');
          await this.loadTraces();
        } catch (err) {
          alert('Failed to save score: ' + err.message);
        }
      });
    }

    window.addEventListener('keydown', (e) => {
      if (e.key === 'Escape') {
        spanModal.classList.add('hidden');
        if (scoreModal) scoreModal.classList.add('hidden');
        document.getElementById('sim-modal').classList.add('hidden');
      }
    });
  },

  async refreshAll() {
    await Promise.all([this.loadTraces(), this.loadSessions(), this.loadMetrics()]);
  },

  async loadTraces(extraParams = {}) {
    try {
      const res = await API.getTraces(extraParams);
      State.traces = res.data || [];
      WaterfallView.render(State.traces);
    } catch (err) {
      console.error('Failed to load traces:', err);
    }
  },

  async loadSessions() {
    try {
      const res = await API.getSessions();
      SessionsView.render(res.data || []);
    } catch (err) {
      console.error('Failed to load sessions:', err);
    }
  },

  async loadMetrics() {
    try {
      const res = await API.getMetrics();
      MetricsView.render(res.data);
    } catch (err) {
      console.error('Failed to load metrics:', err);
    }
  },
};

document.addEventListener('DOMContentLoaded', () => App.init());
