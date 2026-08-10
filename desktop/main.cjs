const { app, shell } = require('electron');
const { spawn } = require('node:child_process');
const crypto = require('node:crypto');
const fs = require('node:fs/promises');
const path = require('node:path');
const os = require('node:os');

const isDevelopment = !app.isPackaged;
// Electron is only the silent launcher/supervisor. It does not create a
// BrowserWindow: React is rendered by the user's default browser at loopback.
const hasSingleInstanceLock = app.requestSingleInstanceLock();
const skipExternalOpen = process.env.ZAJUNA_SKIP_EXTERNAL_OPEN === '1';
let coreProcess;
let endpointFile;
let coreEndpoint;
let stopCorePromise;
let coreStartPromise;
let coreRecoveryPromise;
let coreLogWrite = Promise.resolve();
let quitting = false;

const CORE_LOG_LIMIT = 1024 * 1024;

function redactCoreLog(value) {
  return String(value)
    .replace(/((?:sesskey|token|access_token|refresh_token|password|secret|authorization)[=:\s]+)[^\s&,'"]+/gi, '$1[REDACTED]')
    .replace(/([?&](?:sesskey|token|access_token|refresh_token|password|secret|authorization)=)[^&\s]+/gi, '$1[REDACTED]');
}

function coreLogPath() {
  return path.join(app.getPath('userData'), 'logs', 'zajuna-core.log');
}

function appendCoreLog(chunk) {
  const text = redactCoreLog(chunk);
  coreLogWrite = coreLogWrite
    .then(async () => {
      const logPath = coreLogPath();
      await fs.mkdir(path.dirname(logPath), { recursive: true });
      try {
        const stats = await fs.stat(logPath);
        if (stats.size >= CORE_LOG_LIMIT) {
          await fs.rm(`${logPath}.1`, { force: true });
          await fs.rename(logPath, `${logPath}.1`);
        }
      } catch {
        // El archivo puede no existir todavía.
      }
      await fs.appendFile(logPath, text, 'utf8');
    })
    .catch(() => {
      // La telemetría local no debe impedir que la aplicación funcione.
    });
  return coreLogWrite;
}

async function openExternalBrowser(url) {
  if (skipExternalOpen) return;
  try {
    const opened = await shell.openExternal(url);
    if (!opened) {
      await appendCoreLog(`[shell] El sistema no confirmó la apertura del navegador externo para ${url}.\n`);
    }
  } catch (error) {
    // A browser association can be missing on a fresh machine. Keep the core
    // alive so the user can still open the endpoint from the diagnostic log.
    await appendCoreLog(`[shell] No se pudo abrir el navegador externo: ${error.message}\n`);
  }
}

function coreBinaryPath() {
  const binaryName = process.platform === 'win32' ? 'zajuna-core.exe' : 'zajuna-core';
  return isDevelopment
    ? path.join(__dirname, '..', 'core', 'bin', binaryName)
    : path.join(process.resourcesPath, 'core', binaryName);
}

function validateLocalEndpoint(value) {
  if (!value || typeof value.url !== 'string' || !Number.isInteger(value.port)) {
    throw new Error('El núcleo publicó un endpoint local inválido.');
  }
  let parsed;
  try {
    parsed = new URL(value.url);
  } catch {
    throw new Error('El núcleo publicó una URL local inválida.');
  }
  if (
    parsed.protocol !== 'http:' ||
    parsed.hostname !== '127.0.0.1' ||
    parsed.port !== String(value.port) ||
    parsed.username ||
    parsed.password ||
    parsed.search ||
    parsed.hash ||
    parsed.pathname !== '/'
  ) {
    throw new Error('El núcleo debe publicar únicamente una URL HTTP de loopback.');
  }
  return { url: parsed.origin, port: value.port };
}

async function startCore() {
  if (coreStartPromise) return coreStartPromise;
  const startPromise = startCoreOnce();
  coreStartPromise = startPromise;
  try {
    return await startPromise;
  } finally {
    if (coreStartPromise === startPromise) coreStartPromise = undefined;
  }
}

async function startCoreOnce() {
  if (coreProcess && coreProcess.exitCode === null && coreEndpoint) {
    return coreEndpoint;
  }

  if (coreProcess && coreProcess.exitCode !== null) {
    coreProcess = undefined;
  }

  const binary = coreBinaryPath();
  try {
    await fs.access(binary);
  } catch {
    throw new Error(
      'No se encontró el núcleo local en ' + binary + '. Ejecuta "npm run build" antes de abrir Electron.',
    );
  }

  const endpointNonce = crypto.randomBytes(16).toString('hex');
  endpointFile = path.join(os.tmpdir(), `zajuna-app-${process.pid}-${endpointNonce}.json`);
  await fs.rm(endpointFile, { force: true });

  const child = spawn(binary, ['--port=0', '--no-browser', '--endpoint-file=' + endpointFile], {
    cwd: path.dirname(binary),
    stdio: ['ignore', 'ignore', 'pipe'],
    windowsHide: true,
  });
  coreProcess = child;

  let coreError = '';
  child.stderr?.on('data', (chunk) => {
    const redacted = redactCoreLog(chunk.toString());
    coreError = (coreError + redacted).slice(-8000);
    void appendCoreLog(redacted);
  });

  return new Promise((resolve, reject) => {
    let settled = false;
    let pollTimer;
    const finish = (callback, value) => {
      if (settled) return;
      settled = true;
      clearTimeout(timeout);
      if (pollTimer) clearTimeout(pollTimer);
      callback(value);
    };
    const timeout = setTimeout(() => {
      const message = 'El núcleo local no inició a tiempo.' + (coreError ? ' ' + coreError.trim() : '');
      stopCore().finally(() => finish(reject, new Error(message)));
    }, 10000);

    child.once('error', (error) => {
      finish(reject, new Error(`No se pudo iniciar el núcleo local: ${error.message}`));
    });

    child.once('exit', (code, signal) => {
      if (coreProcess === child) {
        coreProcess = undefined;
        coreEndpoint = undefined;
      }
      if (!settled) {
        const message = 'El núcleo local terminó inesperadamente.' + (coreError ? ' ' + coreError.trim() : '');
        finish(reject, new Error(message));
      } else if (!quitting) {
        void recoverCore(`exit code=${code ?? 'null'} signal=${signal ?? 'null'}`);
      }
    });

    const poll = async () => {
      if (settled) return;
      try {
        const contents = await fs.readFile(endpointFile, 'utf8');
        const endpoint = validateLocalEndpoint(JSON.parse(contents));
        await waitForCoreReady(endpoint);
        coreEndpoint = endpoint;
        finish(resolve, endpoint);
      } catch {
        if (settled) return;
        if (child.exitCode !== null) {
          const message = 'El núcleo local terminó inesperadamente.' + (coreError ? ' ' + coreError.trim() : '');
          finish(reject, new Error(message));
          return;
        }
        pollTimer = setTimeout(poll, 50);
      }
    };

    poll();
  });
}

async function recoverCore(reason) {
  if (quitting || coreRecoveryPromise) return coreRecoveryPromise;

  coreRecoveryPromise = (async () => {
    await appendCoreLog(`\n[supervisor] El núcleo terminó (${reason}); intentando recuperación.\n`);

    let endpoint;
    let lastError;
    for (let attempt = 1; attempt <= 3; attempt += 1) {
      try {
        endpoint = await startCore();
        break;
      } catch (error) {
        lastError = error;
        await appendCoreLog(`[supervisor] Intento ${attempt}/3 fallido: ${error.message}\n`);
        await new Promise((resolve) => setTimeout(resolve, attempt * 500));
      }
    }

    if (!endpoint) {
      const message = 'El núcleo local no pudo recuperarse después de varios intentos.' +
        (lastError ? ` ${lastError.message}` : '');
      await appendCoreLog(`[supervisor] ${message}\n`);
      quitting = true;
      await stopCore();
      app.quit();
      return;
    }
    await openExternalBrowser(endpoint.url);
  })().finally(() => {
    coreRecoveryPromise = undefined;
  });

  return coreRecoveryPromise;
}

async function waitForCoreReady(endpoint, timeoutMs = 5000) {
  const deadline = Date.now() + timeoutMs;
  while (Date.now() < deadline) {
    const controller = new AbortController();
    const abortTimer = setTimeout(() => controller.abort(), 500);
    try {
      const response = await fetch(endpoint.url + '/api/health', { signal: controller.signal });
      if (response.ok) return;
    } catch {
      // El servidor puede haber escrito el endpoint unos milisegundos antes
      // de aceptar la primera petición.
    } finally {
      clearTimeout(abortTimer);
    }
    await new Promise((resolve) => setTimeout(resolve, 50));
  }
  throw new Error('El endpoint local no respondió a /api/health.');
}

async function stopCore() {
  if (stopCorePromise) return stopCorePromise;

  stopCorePromise = (async () => {
    const processToStop = coreProcess;
    coreProcess = undefined;
    coreEndpoint = undefined;

    if (processToStop && processToStop.exitCode === null) {
      await new Promise((resolve) => {
        let settled = false;
        const settle = () => {
          if (settled) return;
          settled = true;
          clearTimeout(forceTimer);
          resolve();
        };
        const forceTimer = setTimeout(() => {
          try {
            processToStop.kill();
          } catch {
            // El proceso ya pudo haber terminado entre los dos intentos.
          }
          settle();
        }, 5000);
        processToStop.once('exit', settle);
        try {
          processToStop.kill('SIGTERM');
        } catch {
          settle();
        }
      });
    }

    if (endpointFile) {
      await fs.rm(endpointFile, { force: true });
      endpointFile = undefined;
    }
  })().finally(() => {
    stopCorePromise = undefined;
  });

  return stopCorePromise;
}

if (!hasSingleInstanceLock) {
  app.exit(0);
} else {
  app.on('second-instance', async () => {
    try {
      const endpoint = coreEndpoint || await startCore();
      await openExternalBrowser(endpoint.url);
    } catch (error) {
      await appendCoreLog(`[launcher] No se pudo reutilizar el core existente: ${error.message}\n`);
    }
  });

  app.whenReady().then(async () => {
    try {
      coreEndpoint = await startCore();
      await openExternalBrowser(coreEndpoint.url);
    } catch (error) {
      await appendCoreLog(`[launcher] No se pudo iniciar Zajuna App: ${error.message}\n`);
      await stopCore();
      app.quit();
    }
  });
}

app.on('before-quit', (event) => {
  if (quitting) return;
  event.preventDefault();
  quitting = true;
  stopCore().finally(() => app.quit());
});

process.on('uncaughtException', (error) => {
  void appendCoreLog(`[launcher] Error no controlado: ${error.message}\n`);
  quitting = true;
  void stopCore().finally(() => app.exit(1));
});
