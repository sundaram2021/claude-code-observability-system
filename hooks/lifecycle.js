const http = require('http');
const fs = require('fs');
const path = require('path');
const os = require('os');
const { spawn } = require('child_process');

const PORT = 3456;
const PLUGIN_ROOT = process.env.CLAUDE_PLUGIN_ROOT || path.resolve(__dirname, '..');

const TELEMETRY_VARS = {
  CLAUDE_CODE_ENABLE_TELEMETRY: '1',
  CLAUDE_CODE_ENHANCED_TELEMETRY_BETA: '1',
  OTEL_TRACES_EXPORTER: 'otlp',
  OTEL_METRICS_EXPORTER: 'otlp',
  OTEL_LOGS_EXPORTER: 'otlp',
  OTEL_EXPORTER_OTLP_PROTOCOL: 'http/json',
  OTEL_EXPORTER_OTLP_ENDPOINT: `http://localhost:${PORT}`,
  OTEL_EXPORTER_OTLP_TRACES_ENDPOINT: `http://localhost:${PORT}/v1/traces`,
  OTEL_LOG_USER_PROMPTS: '1',
  OTEL_LOG_TOOL_DETAILS: '1',
  OTEL_LOG_TOOL_CONTENT: '1',
  OTEL_LOG_ASSISTANT_RESPONSES: '1'
};

function resolveBinary() {
  const isWin = process.platform === 'win32';
  const isMac = process.platform === 'darwin';
  const isArm = process.arch === 'arm64';

  if (isWin) {
    const candidates = ['server-windows-amd64.exe', 'server.exe'];
    for (const name of candidates) {
      const p = path.join(PLUGIN_ROOT, 'bin', name);
      if (fs.existsSync(p)) return p;
    }
  } else if (isMac) {
    const name = isArm ? 'server-darwin-arm64' : 'server-darwin-amd64';
    const p = path.join(PLUGIN_ROOT, 'bin', name);
    if (fs.existsSync(p)) return p;
  } else {
    const p = path.join(PLUGIN_ROOT, 'bin', 'server-linux-amd64');
    if (fs.existsSync(p)) return p;
  }
  return path.join(PLUGIN_ROOT, 'bin', isWin ? 'server.exe' : 'server');
}

function checkPort(port, callback) {
  const req = http.get(`http://127.0.0.1:${port}/api/v1/metrics`, (res) => {
    callback(res.statusCode === 200);
  });
  req.on('error', () => callback(false));
  req.setTimeout(300, () => {
    req.destroy();
    callback(false);
  });
}

function ensureSettings() {
  try {
    const home = os.homedir();
    const settingsPath = path.join(home, '.claude', 'settings.json');
    let settings = {};
    if (fs.existsSync(settingsPath)) {
      settings = JSON.parse(fs.readFileSync(settingsPath, 'utf8'));
    }
    settings.env = settings.env || {};
    Object.assign(settings.env, TELEMETRY_VARS);

    const dir = path.dirname(settingsPath);
    if (!fs.existsSync(dir)) fs.mkdirSync(dir, { recursive: true });
    fs.writeFileSync(settingsPath, JSON.stringify(settings, null, 2), 'utf8');

    if (process.env.CLAUDE_ENV_FILE && fs.existsSync(process.env.CLAUDE_ENV_FILE)) {
      const lines = Object.entries(TELEMETRY_VARS)
        .map(([k, v]) => `export ${k}=${v}`)
        .join('\n') + '\n';
      fs.appendFileSync(process.env.CLAUDE_ENV_FILE, lines, 'utf8');
    }
  } catch (err) {
    // Non-fatal if permission error
  }
}

function handleStart() {
  ensureSettings();
  checkPort(PORT, (running) => {
    if (running) {
      console.log(`✓ Observability dashboard active: http://localhost:${PORT}`);
      process.exit(0);
    }

    const bin = resolveBinary();
    if (!fs.existsSync(bin)) {
      console.error(`Binary not found: ${bin}`);
      process.exit(1);
    }

    if (process.platform !== 'win32') {
      try { fs.chmodSync(bin, 0o755); } catch (_) {}
    }

    const child = spawn(bin, ['start'], {
      stdio: 'ignore',
      env: Object.assign({}, process.env, { PORT: String(PORT) })
    });

    child.on('exit', () => {
      checkPort(PORT, (ok) => {
        if (ok) {
          console.log(`🚀 Observability started: http://localhost:${PORT}`);
        }
        process.exit(0);
      });
    });

    child.on('error', () => {
      process.exit(0);
    });
  });
}

function handleStop() {
  const req = http.request({
    hostname: '127.0.0.1',
    port: PORT,
    path: '/api/v1/shutdown',
    method: 'POST',
    timeout: 1000
  }, () => {
    console.log(`✓ Observability daemon stopping.`);
    process.exit(0);
  });
  req.on('error', () => process.exit(0));
  req.on('timeout', () => { req.destroy(); process.exit(0); });
  req.end();
}

const action = process.argv[2] || 'start';
if (action === 'stop') handleStop();
else handleStart();
