const fs = require('node:fs');
const path = require('node:path');

// Chromium's "running as root without --no-sandbox" guard, and its Ozone
// platform selection, are both decided before Electron's own main script
// (desktop/main.cjs) gets to run — app.commandLine.appendSwitch() from JS is
// too late to affect either check, so the flags must be present in the real
// process argv from exec() time. Real Linux users routinely launch the
// AppImage as root (common on Kali) and/or on a host with no X server (SSH
// sessions, minimal installs) even though this app never opens a
// BrowserWindow, so both flags are required unconditionally, not just for
// root.
//
// electron-builder has no config option for this, so afterPack renames the
// real Electron binary and drops a shell wrapper in its place (under the
// same name the AppImage/.desktop entry expects) that always execs it with
// both flags before any user-supplied arguments.
async function afterPack(context) {
  if (context.electronPlatformName !== 'linux') return;

  const execName = context.packager.executableName;
  const appOutDir = context.appOutDir;
  const realBinary = path.join(appOutDir, execName);
  const renamedBinary = path.join(appOutDir, `${execName}.bin`);

  if (fs.existsSync(renamedBinary)) return; // already wrapped (re-run safety)
  fs.renameSync(realBinary, renamedBinary);

  const wrapper = [
    '#!/bin/sh',
    'DIR="$(dirname "$(readlink -f "$0")")"',
    `exec "$DIR/${execName}.bin" --no-sandbox --ozone-platform=headless "$@"`,
    '',
  ].join('\n');
  fs.writeFileSync(realBinary, wrapper, { mode: 0o755 });
}

module.exports = { afterPack };
