'use strict';

const { spawn } = require('node:child_process');

function stopProcessTree(child, options = {}) {
  const spawnImpl = options.spawnImpl || spawn;
  const platform = options.platform || process.platform;
  const timeoutMs = options.timeoutMs ?? 5000;

  return new Promise((resolve) => {
    if (!child || child.exitCode !== null) {
      resolve();
      return;
    }

    let settled = false;
    const settle = () => {
      if (settled) return;
      settled = true;
      clearTimeout(forceTimer);
      resolve();
    };

    const killTree = (force) => {
      if (platform === 'win32' && child.pid) {
        const args = ['/PID', String(child.pid), '/T'];
        if (force) args.push('/F');
        spawnImpl('taskkill', args, { stdio: 'ignore', windowsHide: true });
        return;
      }
      try {
        child.kill(force ? 'SIGKILL' : 'SIGTERM');
      } catch {
        settle();
      }
    };

    const forceTimer = setTimeout(() => {
      try {
        killTree(true);
      } catch {
        // El proceso ya pudo haber terminado entre los dos intentos.
      }
      settle();
    }, timeoutMs);

    if (typeof child.once === 'function') {
      child.once('exit', settle);
    }
    try {
      killTree(false);
    } catch {
      settle();
    }
  });
}

module.exports = { stopProcessTree };
