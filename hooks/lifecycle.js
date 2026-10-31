const http = require('http');
const fs = require('fs');
const path = require('path');
const os = require('os');
const { spawn } = require('child_process');

const PORT = 3456;
const PLUGIN_ROOT = process.env.CLAUDE_PLUGIN_ROOT || path.resolve(__dirname, '..');

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
    settings.env.CLAUDE_CODE_ENABLE_TELEMETRY = '1';
    settings.env.OTEL_METRICS_EXPORTER = 'otlp';
    settings.env.OTEL_LOGS_EXPORTER = 'otlp';
    settings.env.OTEL_TRACES_EXPORTER = 'otlp';
    settings.env.OTEL_EXPORTER_OTLP_PROTOCOL = 'http/json';
    settings.env.OTEL_EXPORTER_OTLP_ENDPOINT = `http://localhost:${PORT}`;
    settings.env.OTEL_EXPORTER_OTLP_TRACES_ENDPOINT = `http://localhost:${PORT}/v1/traces`;

    const dir = path.dirname(settingsPath);
    if (!fs.existsSync(dir)) fs.mkdirSync(dir, { recursive: true });
    fs.writeFileSync(settingsPath, JSON.stringify(settings, null, 2), 'utf8');
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

    const child = spawn(bin, [], {
      detached: true,
      stdio: 'ignore',
      env: Object.assign({}, process.env, { PORT: String(PORT) })
    });
    child.unref();

    let attempts = 0;
    const interval = setInterval(() => {
      attempts++;
      checkPort(PORT, (ok) => {
        if (ok || attempts >= 20) {
          clearInterval(interval);
          if (ok) {
            console.log(`🚀 Observability started: http://localhost:${PORT}`);
          }
          process.exit(0);
        }
      });
    }, 50);
  });
}

function handleStop() {
  const req = http.request({
    hostname: '127.0.0.1',
    port: PORT,
    path: '/api/v1/shutdown',
    method: 'POST',
    timeout: 800
  }, () => {
    console.log(`✓ Observability daemon stopped.`);
    process.exit(0);
  });
  req.on('error', () => process.exit(0));
  req.on('timeout', () => { req.destroy(); process.exit(0); });
  req.end();
}

const action = process.argv[2] || 'start';
if (action === 'stop') handleStop();
else handleStart();

