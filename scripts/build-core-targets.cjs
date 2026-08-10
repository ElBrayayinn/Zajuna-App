const { spawnSync } = require('node:child_process');
const fs = require('node:fs');
const path = require('node:path');

const projectRoot = path.resolve(__dirname, '..');
const coreRoot = path.join(projectRoot, 'core');
const outputRoot = path.join(projectRoot, 'dist', 'core-targets');

const targets = [
  { id: 'windows-x64', goos: 'windows', goarch: 'amd64', binary: 'zajuna-core.exe' },
  { id: 'windows-arm64', goos: 'windows', goarch: 'arm64', binary: 'zajuna-core.exe' },
  { id: 'linux-x64', goos: 'linux', goarch: 'amd64', binary: 'zajuna-core' },
  { id: 'linux-arm64', goos: 'linux', goarch: 'arm64', binary: 'zajuna-core' },
  { id: 'macos-x64', goos: 'darwin', goarch: 'amd64', binary: 'zajuna-core' },
  { id: 'macos-arm64', goos: 'darwin', goarch: 'arm64', binary: 'zajuna-core' },
];

fs.mkdirSync(outputRoot, { recursive: true });

for (const target of targets) {
  const targetDir = path.join(outputRoot, target.id);
  const outputPath = path.join(targetDir, target.binary);
  fs.mkdirSync(targetDir, { recursive: true });

  console.log(`Compilando core para ${target.id}...`);
  const result = spawnSync(
    'go',
    ['build', '-trimpath', '-buildvcs=false', '-ldflags=-buildid=', '-o', outputPath, './cmd/zajuna-core'],
    {
    cwd: coreRoot,
    env: { ...process.env, GOOS: target.goos, GOARCH: target.goarch, CGO_ENABLED: '0' },
    stdio: 'inherit',
    // Go is an executable on every supported build host; avoiding a nested
    // cmd.exe keeps the synchronous cross-build from leaving a child shell
    // alive after the final target completes on Windows.
      shell: false,
    },
  );

  if (result.error) {
    console.error(`No se pudo ejecutar Go para ${target.id}: ${result.error.message}`);
    process.exit(1);
  }
  if (result.status !== 0) {
    process.exit(result.status ?? 1);
  }
}

console.log(`Cores multiplataforma generados en ${path.relative(projectRoot, outputRoot)}`);
