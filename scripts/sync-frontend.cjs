const fs = require('node:fs');
const path = require('node:path');

const projectRoot = path.resolve(__dirname, '..');
const frontendDist = path.resolve(projectRoot, 'frontend', 'dist');
const embeddedWeb = path.resolve(projectRoot, 'core', 'cmd', 'zajuna-core', 'web');

function assertInsideProject(target, label) {
  const relative = path.relative(projectRoot, target);
  if (!relative || relative.startsWith('..') || path.isAbsolute(relative)) {
    throw new Error(label + ' está fuera del proyecto Zajuna.App: ' + target);
  }
}

assertInsideProject(frontendDist, 'El build de frontend');
assertInsideProject(embeddedWeb, 'La UI embebida');

const indexPath = path.join(frontendDist, 'index.html');
if (!fs.existsSync(indexPath)) {
  throw new Error(
    'No existe ' + indexPath + '. Ejecuta "npm run frontend:build" antes de sincronizar la UI embebida.',
  );
}
const indexContents = fs.readFileSync(indexPath, 'utf8');
if (!indexContents.includes('id="root"')) {
  throw new Error('El build encontrado no parece ser la aplicación React de Zajuna.App.');
}

fs.rmSync(embeddedWeb, { recursive: true, force: true });
fs.mkdirSync(embeddedWeb, { recursive: true });
fs.cpSync(frontendDist, embeddedWeb, { recursive: true });

console.log('Frontend React sincronizado en ' + path.relative(projectRoot, embeddedWeb));
