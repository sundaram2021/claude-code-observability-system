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
  },

  setupModals() {
    const modal = document.getElementById('span-modal');
    document.getElementById('modal-close').addEventListener('click', () => {
      modal.classList.add('hidden');
    });
    modal.addEventListener('click', (e) => {
      if (e.target === modal) modal.classList.add('hidden');
    });
    window.addEventListener('keydown', (e) => {
      if (e.key === 'Escape') {
        modal.classList.add('hidden');
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

