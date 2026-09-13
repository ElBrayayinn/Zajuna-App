'use strict';

const { spawn } = require('node:child_process');

function stopProcessTree(child, options = {}) {
  const spawnImpl = options.spawnImpl || spawn;
  const platform = options.platform || process.platform;
  const killImpl = options.killImpl || ((pid, signal) => process.kill(pid, signal));
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
      const signal = force ? 'SIGKILL' : 'SIGTERM';
      // A plain child.kill() only signals the direct child, not any
      // grandchildren it spawned (e.g. a Playwright/Chromium capture worker
      // launched by the Go core). The core is spawned detached on POSIX so
      // its pid is also its process group id; signaling -pid reaches the
      // whole group instead of orphaning those descendants.
      try {
        if (child.pid) {
          killImpl(-child.pid, signal);
        } else {
          child.kill(signal);
        }
      } catch (error) {
        if (error && error.code === 'ESRCH') {
          settle();
          return;
        }
        try {
          child.kill(signal);
        } catch {
          settle();
        }
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
