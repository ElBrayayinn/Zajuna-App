'use strict';

const assert = require('node:assert/strict');
const { EventEmitter } = require('node:events');
const { stopProcessTree } = require('./stop-process-tree.cjs');

function fakeChild(pid) {
  const child = new EventEmitter();
  child.pid = pid;
  child.exitCode = null;
  child.signals = [];
  child.kill = (signal) => {
    child.signals.push(signal || 'SIGTERM');
  };
  child.exit = (code = 0) => {
    child.exitCode = code;
    child.emit('exit', code, null);
  };
  return child;
}

async function testAlreadyExitedResolvesImmediately() {
  const calls = [];
  const child = fakeChild(11);
  child.exitCode = 0;
  await stopProcessTree(child, {
    platform: 'win32',
    spawnImpl: (...args) => {
      calls.push(args);
    },
  });
  assert.equal(calls.length, 0);
}

async function testWindowsUsesTaskkillTreeThenForce() {
  const calls = [];
  const child = fakeChild(42);
  const done = stopProcessTree(child, {
    platform: 'win32',
    timeoutMs: 20,
    spawnImpl: (command, args, options) => {
      calls.push({ command, args, options });
    },
  });
  await new Promise((resolve) => setTimeout(resolve, 5));
  assert.equal(calls.length, 1);
  assert.equal(calls[0].command, 'taskkill');
  assert.deepEqual(calls[0].args, ['/PID', '42', '/T']);
  await new Promise((resolve) => setTimeout(resolve, 30));
  await done;
  assert.equal(calls.length, 2);
  assert.deepEqual(calls[1].args, ['/PID', '42', '/T', '/F']);
}

async function testUnixSignalsWholeProcessGroup() {
  const calls = [];
  const child = fakeChild(7);
  const done = stopProcessTree(child, {
    platform: 'linux',
    timeoutMs: 20,
    killImpl: (pid, signal) => calls.push({ pid, signal }),
  });
  await new Promise((resolve) => setTimeout(resolve, 5));
  assert.deepEqual(calls, [{ pid: -7, signal: 'SIGTERM' }]);
  await new Promise((resolve) => setTimeout(resolve, 30));
  await done;
  assert.deepEqual(calls, [
    { pid: -7, signal: 'SIGTERM' },
    { pid: -7, signal: 'SIGKILL' },
  ]);
  assert.deepEqual(child.signals, []);
}

async function testUnixSettlesImmediatelyWhenGroupIsGone() {
  const child = fakeChild(8);
  const done = stopProcessTree(child, {
    platform: 'linux',
    timeoutMs: 20,
    killImpl: () => {
      const error = new Error('No such process');
      error.code = 'ESRCH';
      throw error;
    },
  });
  await done;
  assert.deepEqual(child.signals, []);
}

async function testExitCancelsForceKill() {
  const calls = [];
  const child = fakeChild(9);
  const done = stopProcessTree(child, {
    platform: 'win32',
    timeoutMs: 50,
    spawnImpl: (command, args) => {
      calls.push(args);
    },
  });
  child.exit(0);
  await done;
  assert.equal(calls.length, 1);
  assert.deepEqual(calls[0], ['/PID', '9', '/T']);
}

(async () => {
  await testAlreadyExitedResolvesImmediately();
  await testWindowsUsesTaskkillTreeThenForce();
  await testUnixSignalsWholeProcessGroup();
  await testUnixSettlesImmediatelyWhenGroupIsGone();
  await testExitCancelsForceKill();
  console.log('stop-process-tree tests: 5 passed');
})().catch((error) => {
  console.error(error);
  process.exit(1);
});
