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

  async createScore(payload) {
    const res = await fetch('/api/v1/scores', {
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
    let retryDelay = 1000;

    function connect() {
      if (evtSource) {
        evtSource.close();
      }
      evtSource = new EventSource('/api/v1/live');

      evtSource.onopen = () => {
        retryDelay = 1000;
        if (onStatusChange) onStatusChange(true);
      };

      evtSource.addEventListener('connected', () => {
        retryDelay = 1000;
        if (onStatusChange) onStatusChange(true);
      });

      const handleEvent = (e) => {
        try {
          const data = e.data ? JSON.parse(e.data) : {};
          if (onEvent) onEvent(e.type, data);
        } catch (_) {
          if (onEvent) onEvent(e.type, {});
        }
      };

      evtSource.addEventListener('TRACE_CREATED', handleEvent);
      evtSource.addEventListener('TRACE_UPDATED', handleEvent);
      evtSource.addEventListener('SPAN_CREATED', handleEvent);
      evtSource.onmessage = handleEvent;

      evtSource.onerror = () => {
        if (onStatusChange) onStatusChange(false);
        evtSource.close();
        setTimeout(connect, retryDelay);
        retryDelay = Math.min(retryDelay * 1.5, 5000);
      };
    }

    connect();
  },
};
