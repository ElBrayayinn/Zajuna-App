/**
 * DEPRECATED — entrypoint histórico.
 *
 * El proceso Electron real vive en `desktop/main.cjs` (ver package.json#main).
 * Este archivo se conserva solo para no romper scripts o atajos antiguos que
 * aún apunten a la raíz. No añadir lógica nueva aquí.
 *
 * @deprecated Use desktop/main.cjs
 */
'use strict';

module.exports = require('./desktop/main.cjs');
