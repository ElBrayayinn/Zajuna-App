# MDL-123 — Verificación en Kali Linux real y corrección de bugs (2026-09-13)

Issue: [MDL-123](https://linear.app/medialab-sena/issue/MDL-123), pendiente
desde [MDL-120](https://linear.app/medialab-sena/issue/MDL-120) (que solo
probó en WSL2/Ubuntu).

## Entorno de prueba

Contenedor Docker con la imagen oficial `kalilinux/kali-rolling` (Kali
2026.3), con acceso a `/dev/fuse` (`--device /dev/fuse --cap-add SYS_ADMIN`).
No es bare-metal, pero es el userland real de Kali (paquetes, versión de
FUSE, systemd, comportamiento por defecto como root de un contenedor), no una
emulación ni un derivado de Ubuntu como en MDL-120.

Toolchain instalado: Node.js 24.19.0, npm 11.19.0, Go 1.26.7, `fuse3`
(paquete `fuse`/`fuse3`, ver hallazgo 1), y las librerías compartidas de
Chromium (`libnspr4`, `libnss3`, `libatk-bridge2.0-0`, `libgtk-3-0`, etc.).

## Bugs encontrados y corregidos

### 1. Kali no empaqueta `libfuse2` — bloquea el 100% de instalaciones

`apt-cache policy libfuse2` no tiene candidato en Kali rolling; solo existen
`fuse3`/`libfuse3-4`. El runtime AppImage por defecto de `electron-builder`
necesita `libfuse.so.2`, así que el AppImage falla siempre al arrancar:

```
dlopen(): error loading libfuse.so.2
AppImages require FUSE to run...
```

No hay corrección de código posible (es una dependencia del sistema
operativo que Kali no ofrece). Documentado en
[`guia-instalacion.md`](guia-instalacion.md#solución-de-problemas-comunes):
usar `--appimage-extract-and-run` en Kali/Debian rolling.

### 2. `--no-sandbox` vía JS no evita el crash de root

`desktop/main.cjs` ya desactivaba el sandbox con
`app.commandLine.appendSwitch('no-sandbox')` (MDL-120, commit `5df737f`).
Verificado en Kali real que **esto no evita** el guard nativo de Electron:

```
[FATAL] electron_main_delegate.cc:312] Running as root without --no-sandbox
is not supported.
```

La causa: ese chequeo corre en el arranque nativo de Chromium, antes de que
Electron ejecute el script de JS del proceso principal — cualquier
`appendSwitch` llamado desde `desktop/main.cjs` llega demasiado tarde para
esta invocación. Solo funciona si `--no-sandbox` está en el argv real del
proceso desde `exec()`, que es exactamente lo que hacía
`scripts/smoke-packaged.cjs:94` — por eso el smoke de CI nunca lo detectó.

**Corregido**: `scripts/package.cjs` ahora genera un config de
`electron-builder` en `.cjs` (no `.json`, para poder incluir una función) con
un hook `afterPack` (`scripts/linux-no-sandbox-wrapper.cjs`) que renombra el
binario real de Electron a `<exec>.bin` dentro del AppImage y coloca en su
lugar un wrapper de shell que siempre re-ejecuta con `--no-sandbox
--ozone-platform=headless` antes de cualquier argumento del usuario. Esto
garantiza que el flag esté en el argv real desde el arranque del proceso,
sin depender de que un usuario real pase flags manualmente.

### 3. La app exige un servidor X aunque nunca abre ventana

Incluso sin ser root, sin flags, Electron fallaba igual en cualquier host sin
sesión gráfica (fue el caso al probar como usuario no-root en el contenedor,
sin `$DISPLAY`):

```
[ERROR] ui/ozone/platform/x11/ozone_platform_x11.cc:250] Missing X server or
$DISPLAY
[ERROR] ui/aura/env.cc:257] The platform failed to initialize.  Exiting.
```

`desktop/main.cjs` nunca crea un `BrowserWindow` (React se abre en el
navegador del usuario vía loopback), pero Electron igual inicializa el
backend gráfico Ozone/X11 por defecto. Igual que el bug 2, un
`app.commandLine.appendSwitch('ozone-platform', 'headless')` desde JS **no
evita el crash** — se probó explícitamente y falla igual. Corregido con el
mismo wrapper del bug 2 (`--ozone-platform=headless` en el argv real).

Cualquier Kali sin sesión gráfica activa (servidor, SSH, instalación mínima)
habría hecho fallar la app antes de este fix, sin necesitar Xvfb ni ningún
servidor X real.

### 4. `stopProcessTree` en Linux/macOS solo mataba el proceso directo

`desktop/stop-process-tree.cjs` usaba `child.kill(signal)` en POSIX, que solo
señala el PID del propio `zajuna-core`, no su árbol de procesos. Si el core
tiene una captura Playwright/Chromium en curso cuando la app se cierra y
llega el `SIGKILL` forzado (5s), el proceso Go muere sin ejecutar sus
`defer browser.Close()`/`pw.Stop()`, dejando un Chromium huérfano.

**Corregido**: el core ahora se lanza con `detached: true` en POSIX
(`desktop/main.cjs`), quedando como líder de su propio grupo de procesos, y
`stopProcessTree` señala `-pid` (grupo completo) en vez de solo `pid`. Tests
actualizados en `desktop/stop-process-tree.test.cjs`.

## Verificación de los fixes

Reconstruido el AppImage en el mismo contenedor Kali con los cuatro cambios
aplicados y probado el binario extraído (`dist/linux-unpacked/zajuna-app`)
sin ningún flag de línea de comandos, como root y como usuario normal, sin
servidor X en ambos casos:

```
$ whoami
root
$ ./zajuna-app --user-data-dir=/tmp/zt-final
$ curl http://127.0.0.1:<puerto>/api/health
{"app":"zajuna-app","runtime":"linux","status":"ok","version":"0.1.1"}
```

Mismo resultado como usuario no-root. Antes del fix, ambos casos fallaban
con el FATAL del bug 2 (root) o el crash de Ozone/X11 del bug 3 (no-root).

## No verificado en esta sesión

- Bare-metal/VM real de Kali (solo contenedor Docker sobre el mismo host
  Windows/WSL2 del resto del gate).
- El flujo de `--appimage-extract-and-run` con los binarios corregidos (se
  verificó el binario extraído directamente, no el AppImage sin extraer,
  dado el bug 1).
- El workflow `Native installers` en CI con estos cambios.
