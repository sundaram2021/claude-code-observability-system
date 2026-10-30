// API Client for Claude Code Observability
const API = {
  async getTraces(params = {}) {
    const qs = new URLSearchParams(params).toString();
    const res = await fetch(`/api/v1/traces?${qs}`);
    if (!res.ok) throw new Error(`HTTP ${res.status}`);
    return res.json();
  },

  async getTrace(id) {
    const res = await fetch(`/api/v1/traces/${encodeURIComponent(id)}`);
    if (!res.ok) throw new Error(`HTTP ${res.status}`);
    return res.json();
  },

  async getSessions(params = {}) {
    const qs = new URLSearchParams(params).toString();
    const res = await fetch(`/api/v1/sessions?${qs}`);
    if (!res.ok) throw new Error(`HTTP ${res.status}`);
    return res.json();
  },

  async getMetrics() {
    const res = await fetch('/api/v1/metrics');
    if (!res.ok) throw new Error(`HTTP ${res.status}`);
    return res.json();
  },

  async runAgent(payload) {
    const res = await fetch('/api/v1/agent/run', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload),
    });
    if (!res.ok) throw new Error(`HTTP ${res.status}`);
    return res.json();
  },

  async clearTraces() {
    const res = await fetch('/api/v1/traces', { method: 'DELETE' });
    if (!res.ok) throw new Error(`HTTP ${res.status}`);
    return res.json();
  },

  connectSSE(onEvent, onStatusChange) {
    let evtSource = null;

    function connect() {
      if (onStatusChange) onStatusChange(false);
      evtSource = new EventSource('/api/v1/live');

      evtSource.addEventListener('connected', () => {
        if (onStatusChange) onStatusChange(true);
      });

      evtSource.addEventListener('TRACE_CREATED', (e) => {
        try {
          const data = JSON.parse(e.data);
          if (onEvent) onEvent('TRACE_CREATED', data);
        } catch (err) {
          console.error('SSE parse error:', err);
        }
      });

      evtSource.onerror = () => {
        if (onStatusChange) onStatusChange(false);
        evtSource.close();
        setTimeout(connect, 4000);
      };
    }

    connect();
  },
};

