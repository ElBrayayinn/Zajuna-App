# MDL-120 — AppImage Linux corregido y probado (2026-09-11)

Issue: [MDL-120](https://linear.app/medialab-sena/issue/MDL-120).

## Entorno de prueba

**No es Kali Linux/Debian bare-metal**, como pide la tarea: es Ubuntu 26.04
LTS bajo WSL2 (kernel `6.18.33.2-microsoft-standard-WSL2`) en la misma
estación Windows del resto del gate. Es un kernel Linux real (no una
emulación), con FUSE, Chromium y systemd propios, pero no sustituye una
verificación en un Kali/Debian real — eso queda pendiente para MDL-123.

Toolchain instalado en WSL para esta sesión: Node.js 22.23.2 (NodeSource),
Go 1.26.0 (`apt`), `fuse3`/`libfuse2t64`, y las librerías compartidas que
requiere el Chromium embebido de Electron (`libnspr4`, `libnss3`, `libatk...`,
`libgtk-3-0t64`, etc. — ninguna estaba preinstalada en la imagen base).

## Comando de build

```bash
npm ci && npm ci --prefix frontend
npm run package:linux
```

Artefacto: `dist/Zajuna App-0.1.0.AppImage`, 435 464 656 bytes (~415 MiB).
SHA-256: `52496c9437fbd71da42f4f7688f8ae4cd6bef8cf07fad0f9d0e8f0d979195f26`.

## Bugs encontrados y corregidos

1. **`scripts/smoke-packaged.cjs` y `scripts/smoke-external-browser.cjs`**
   buscaban el binario Linux empaquetado como `dist/linux-unpacked/Zajuna App`.
   electron-builder genera el binario como `dist/linux-unpacked/zajuna-app`
   (minúsculas, con guion, tomado del campo `name` de `package.json`). Ambos
   scripts nunca se habían corrido contra un Linux real antes — el smoke
   fallaba con "No existe el ejecutable empaquetado" en cuanto se probó aquí.
   Corregido para usar `zajuna-app`, igual que ya hacía `scripts/smoke-native.cjs`.
2. **Crash del proceso GPU en Linux sin aceleración de hardware.** Sin
   ventana ni renderer (`desktop/main.cjs` nunca crea `BrowserWindow`), el
   proceso GPU de Chromium igual se lanzaba y fallaba en un host sin GPU
   (`Exiting GPU process due to errors during initialization`), lo que
   impedía que el core respondiera a `/api/health` dentro del timeout del
   smoke. Corregido con `app.disableHardwareAcceleration()` al inicio de
   `desktop/main.cjs`, antes de `requestSingleInstanceLock()` — evita que se
   lance el proceso GPU por completo. Verificado: el mismo smoke que fallaba
   antes de este cambio pasa después, sin flags manuales.
   Este mismo bug probablemente afecta al runner `ubuntu-latest` de
   `.github/workflows/native-installers.yml`, que tampoco tiene GPU.

Hay un `main.cjs` duplicado en la raíz del repo (el módulo real es
`desktop/main.cjs`, el que referencia `package.json#main` y el único que
`electron-builder` empaqueta). El de la raíz no se usa en ningún flujo. No lo
toqué porque no forma parte de esta tarea, pero queda como hallazgo: alguien
debería confirmar si se puede eliminar.

## Evidencia de ejecución

```text
$ npm run test:smoke:packaged
Iniciando smoke del paquete: /home/…/dist/linux-unpacked/zajuna-app
Smoke OK: el core empaquetado respondió a /api/health en loopback.
Smoke OK: embed, fallback SPA, assets y 404 API/static verificados.

$ npm run test:smoke:external-browser
Smoke OK: modo externo activo, endpoint loopback http://127.0.0.1:40045.
Smoke OK: el segundo lanzamiento reutilizó la instancia existente.
```

Ejecución directa del AppImage (`chmod +x` ya viene dado por electron-builder):

```text
$ ./"Zajuna App-0.1.0.AppImage" --no-sandbox --user-data-dir=$(pwd)/tmp/test
# monta en /tmp/.mount_Zajuna…, lanza zygotes, network service y el core Go
$ curl http://127.0.0.1:<puerto>/api/health
{"app":"zajuna-app","runtime":"linux","status":"ok","version":"0.1.0"}
```

FUSE: `/dev/fuse` y `fusermount`/`fusermount3` disponibles y usados por el
AppImage sin configuración adicional (más allá de instalar `libfuse2t64`,
que no venía en la imagen base). No se probó `--appimage-extract-and-run`
como modo de diagnóstico alterno en esta corrida por límite de tiempo; queda
para MDL-123.

`--no-sandbox` fue necesario para correr dentro de WSL2 (el sandbox de
Chromium requiere `CLONE_NEWUSER` sin restricciones, que algunas
configuraciones de WSL2 bloquean); no se investigó si un Kali/Debian real
necesita el mismo flag — otro punto para MDL-123.

## No verificado en esta sesión

- Kali Linux/Debian real (bare-metal o VM), solo WSL2/Ubuntu.
- Concurrencia y consumo de recursos (MDL-121, MDL-122).
- `--appimage-extract-and-run`.
- El workflow real `Native installers` en `ubuntu-latest` (sin permisos de
  admin en el repo para dispararlo, ver `release-gate-2026-09-11.md`).
