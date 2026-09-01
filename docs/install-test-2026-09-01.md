# Prueba de instalación nativa — 2026-09-01 (MDL-29)

Protocolo y bitácora de la primera instalación de Zajuna App en máquinas
distintas a la estación de desarrollo. Cubre Windows 10/11 x64 y Linux x64.

Este documento se llena **mientras** se ejecuta la prueba. Un casillero sin
resultado escrito significa "no ejecutado", nunca "pasó".

## Qué desbloqueó esta jornada

El workflow `Native installers` existía desde el 2026-08-26 pero **nunca
produjo un artefacto**: sus 7 ejecuciones registradas terminaban en 0 s.

| # | Causa | Corrección |
|---|---|---|
| 1 | El paso «Verify Windows Authenticode» leía el contexto `secrets` dentro de un `if` de paso, donde ese contexto no existe. GitHub invalidaba el workflow completo antes de arrancar. | Los secretos CSC pasan a `env` a nivel de job; el `if` lee `env.CSC_LINK`. |
| 2 | El runner Linux es headless y Electron exige un servidor X aunque la app no abra `BrowserWindow`. | El smoke empaquetado se separa por sistema; Linux corre bajo `xvfb-run`. |
| 3 | electron-builder detecta CI y activa publicación implícita: el AppImage se construía completo y luego abortaba con `GitHub Personal Access Token is not set`. | `scripts/package.cjs` pasa `--publish never` salvo que se pida lo contrario. |
| 4 | El smoke buscaba `dist/linux-unpacked/Zajuna App`. electron-builder nombra el binario Linux con `appInfo.sanitizedName` en minúscula, o sea el campo `name` → `zajuna-app`; `Zajuna App` solo aplica al `.exe`. | El smoke prueba ambos nombres y descubre el ejecutable en `linux-unpacked` si cambia. |
| 5 | El smoke lanzaba el paquete con `stdio: 'ignore'`: un arranque fallido solo se veía como «no expuso /api/health». | Se guarda una cola acotada de stdout/stderr y se adjunta al error. |
| 6 | Con la salida ya visible: Electron aborta con `FATAL: The SUID sandbox helper binary was found, but is not configured correctly` porque `chrome-sandbox` no es setuid root en `dist/linux-unpacked`. | Solo la invocación del smoke pasa `--no-sandbox`; la app entregada no cambia. **Queda abierto en máquina real** (ver B2). |

## Artefactos bajo prueba

| Campo | Valor |
|---|---|
| Rama | `mdl-29-native-installers-fix` |
| Commits | `284e34d`, `d2eb8a1`, `76e4a87`, `0de73d8`, `25512ac` |
| Windows | `Zajuna App Setup 0.1.0.exe` (NSIS, x64) |
| Linux | `Zajuna App-0.1.0.AppImage` (x64) |
| Firma Windows | **Ausente.** Sin certificado Authenticode (`CSC_LINK` no configurado). |
| Integridad | `release-manifest.json` con SHA256 por artefacto. |

Los SHA256 se copian aquí desde `release-manifest.json` antes de instalar, y se
vuelven a calcular en el PC de prueba. Si no coinciden, la prueba se detiene.

| Artefacto | SHA256 esperado | SHA256 en el PC de prueba |
|---|---|---|
| `Zajuna App Setup 0.1.0.exe` | `a375623903f7d49f6a7d036814cb20ee2742a35b1cabc60730d4b9589b906c45` | (pendiente) |
| `Zajuna App-0.1.0.AppImage` | (pendiente) | (pendiente) |

## Qué debe hacer la app si todo va bien

Zajuna App no es un sitio web. El acceso directo lanza un launcher Electron
**sin ventana propia**, que arranca el core Go escuchando solo en
`127.0.0.1:<puerto aleatorio>` y abre el navegador predeterminado contra esa
dirección. Cerrar la pestaña no apaga el core.

- Datos en Windows: `%LOCALAPPDATA%\ZajunaApp` (`config.json`, SQLite, evidencias).
- Datos en Linux: `~/.local/share/zajuna-app` (o `$XDG_DATA_HOME/zajuna-app`).
- Contraseña de Zajuna: almacén del sistema, servicio `zajuna-app`. Nunca en
  `config.json` ni en un `.env`.
- Archivo de endpoint: `zajuna-app-<pid>-<nonce>.json` en el directorio temporal.

## Bloque A — Windows 10/11 x64

### A0. Preparación

1. PC de prueba **sin** Node, Go ni el repo. Se prueba el instalador, no el build.
2. Anotar edición y build de Windows: `winver`.
3. Confirmar que no hay una instalación previa: Configuración → Aplicaciones →
   buscar «Zajuna». Si aparece, esta ya no es una instalación limpia.

### A1. Descargar y verificar integridad

```powershell
Get-FileHash '.\Zajuna App Setup 0.1.0.exe' -Algorithm SHA256
```

Comparar con `release-manifest.json`. **Si no coincide, detener la prueba.**

### A2. Advertencia de SmartScreen (hallazgo esperado)

El instalador **no está firmado**, así que SmartScreen mostrará «Windows
protegió su PC». Eso es el resultado correcto de esta versión, no un fallo de
la prueba: es la evidencia de que el release comercial sigue bloqueado hasta
que exista el certificado Authenticode.

- Capturar la advertencia. Es el artefacto que cierra ese criterio de MDL-29.
- La instalación continúa porque es un PC de la oficina que tú administras y
  el binario lo construyó nuestro propio CI. Es una decisión de QA registrada.
- **Esta instrucción no debe aparecer en la página pública de descargas** —
  MDL-28 eliminó justamente esa guía para usuarios finales.

### A3. Instalación limpia

| Paso | Qué observar | Resultado |
|---|---|---|
| Ejecutar el instalador | Termina sin error | |
| Carpeta de instalación | Anotar la ruta real | |
| Menú Inicio | Existe la entrada «Zajuna App» | |

### A4. Primer arranque

| Paso | Qué observar | Resultado |
|---|---|---|
| Abrir «Zajuna App» | Abre el navegador predeterminado en `127.0.0.1:<puerto>` | |
| Anotar el puerto | | |
| Pantalla inicial | Aparece **Setup** (tipo de documento y contraseña) | |

Salud del backend, sustituyendo el puerto observado:

```powershell
Invoke-RestMethod http://127.0.0.1:PUERTO/api/health | ConvertTo-Json
```

Se espera `status: ok`, `app: zajuna-app`, `runtime: windows`. Guardar la salida.

### A5. Credenciales en el almacén del sistema

Completar Setup con la cuenta de prueba de Zajuna. Después:

| Comprobación | Cómo | Resultado |
|---|---|---|
| La contraseña **no** está en `config.json` | `Get-Content "$env:LOCALAPPDATA\ZajunaApp\config.json"` | |
| Existe la credencial | Administrador de credenciales → Credenciales de Windows → `zajuna-app` | |

La contraseña no se escribe en este documento ni en Linear.

### A6. Instancia única y cierre limpio

| Paso | Qué observar | Resultado |
|---|---|---|
| Abrir el acceso directo por segunda vez | Reabre la misma URL; **no** aparece un segundo puerto | |
| `Get-Process zajuna-core` | Un solo proceso | |
| Cerrar la app (bandeja / cerrar el proceso del launcher Electron) | | |
| `Get-Process zajuna-core -ErrorAction SilentlyContinue` | **Vacío** — sin procesos huérfanos | |
| `Get-ChildItem $env:TEMP -Filter 'zajuna-app-*.json'` | **Vacío** — el archivo de endpoint se borró | |

### A7. Actualización sobre la instalación existente

Reinstalar el mismo `.exe` encima (simula la actualización).

| Comprobación | Resultado |
|---|---|
| El instalador no exige desinstalar primero | |
| La app arranca después | |
| `config.json` y la base SQLite se conservan | |

### A8. Desinstalación

Configuración → Aplicaciones → Zajuna App → Desinstalar.

| Comprobación | Cómo | Resultado |
|---|---|---|
| Binarios eliminados | La carpeta de A3 ya no existe | |
| Acceso directo eliminado | Menú Inicio | |
| Sin procesos vivos | `Get-Process zajuna-core -ErrorAction SilentlyContinue` | |
| Carpeta de datos | `Test-Path "$env:LOCALAPPDATA\ZajunaApp"` — anotar si queda | |
| Credencial | Administrador de credenciales — anotar si queda | |

Que los datos del usuario sobrevivan a la desinstalación es normal en un NSIS
por usuario. Lo que hay que **decidir y registrar** es si eso es lo que
queremos: son evidencias y una base SQLite con datos de fichas reales.

## Bloque B — Linux x64 (Ubuntu / Debian)

### B0. Preparación

1. Anotar distribución y versión: `cat /etc/os-release`.
2. Anotar entorno de escritorio y si hay sesión gráfica:
   `echo $XDG_CURRENT_DESKTOP $DISPLAY`.
3. Dos dependencias del sistema que **sí** afectan esta prueba:
   - **FUSE.** Un AppImage necesita FUSE 2. En Ubuntu 22.04+ y 24.04 ya no
     viene: `sudo apt install libfuse2` (o `libfuse2t64` en 24.04).
     Alternativa sin instalar nada: `--appimage-extract-and-run`.
   - **Llavero.** La contraseña usa el Secret Service de D-Bus
     (`gnome-keyring` / KWallet). Sin sesión gráfica con llavero desbloqueado,
     guardar la contraseña **falla**. Si ocurre, es un hallazgo real y hay que
     anotar el error exacto, no rodearlo.

### B1. Descargar y verificar integridad

```bash
sha256sum "Zajuna App-0.1.0.AppImage"
chmod +x "Zajuna App-0.1.0.AppImage"
```

Comparar con `release-manifest.json`. **Si no coincide, detener la prueba.**

### B2. Primer arranque

```bash
./"Zajuna App-0.1.0.AppImage"
```

Si falla por FUSE:

```bash
./"Zajuna App-0.1.0.AppImage" --appimage-extract-and-run
```

**Hallazgo abierto que esta prueba debe resolver.** En el runner de CI el
paquete sin instalar aborta con:

```text
FATAL: The SUID sandbox helper binary was found, but is not configured
correctly. ... chrome-sandbox is owned by root and has mode 4755
```

Chromium cae al sandbox SUID cuando el sistema no permite user namespaces sin
privilegios, que es el caso de Ubuntu 24.04 por AppArmor. En CI se rodeó con
`--no-sandbox` porque ahí solo se valida el core Go, pero **no sabemos si el
AppImage arranca en un escritorio Linux real**. Si aparece el mismo mensaje en
el PC de prueba, anótalo textualmente: pasa a ser una decisión de producto
(shipear `--no-sandbox` en el AppRun del AppImage, o exigir `chrome-sandbox`
setuid), no un ajuste de prueba. Para seguir con el resto del protocolo ese día:

```bash
./"Zajuna App-0.1.0.AppImage" --no-sandbox
```

| Paso | Qué observar | Resultado |
|---|---|---|
| Abre el navegador predeterminado en `127.0.0.1:<puerto>` | | |
| Anotar el puerto | | |
| Aparece **Setup** | | |

```bash
curl -s http://127.0.0.1:PUERTO/api/health
```

Se espera `status: ok`, `runtime: linux`. Guardar la salida.

### B3. Credenciales

| Comprobación | Cómo | Resultado |
|---|---|---|
| La contraseña **no** está en `config.json` | `cat ~/.local/share/zajuna-app/config.json` | |
| Existe la credencial | `secret-tool search service zajuna-app` | |
| Si el llavero falla | Anotar el error textual de la app | |

### B4. Instancia única y cierre limpio

| Paso | Qué observar | Resultado |
|---|---|---|
| Ejecutar el AppImage otra vez | Reabre la misma URL; sin segundo puerto | |
| `pgrep -a zajuna-core` | Un solo proceso | |
| Cerrar la app | | |
| `pgrep -a zajuna-core` | **Vacío** — sin huérfanos | |
| `ls /tmp/zajuna-app-*.json` | **Vacío** | |

### B5. Actualización

Reemplazar el AppImage por el siguiente build y ejecutarlo.

| Comprobación | Resultado |
|---|---|
| Arranca con el binario nuevo | |
| `~/.local/share/zajuna-app` conserva `config.json` y la base | |

### B6. Desinstalación

Un AppImage no tiene desinstalador: se borra el archivo.

| Comprobación | Cómo | Resultado |
|---|---|---|
| Sin procesos vivos tras borrarlo | `pgrep -a zajuna-core` | |
| Datos que quedan | `du -sh ~/.local/share/zajuna-app` | |
| Credencial que queda | `secret-tool search service zajuna-app` | |
| Lanzador en el menú | Anotar si el AppImage registró un `.desktop` huérfano | |

## Resultado consolidado

| Criterio de MDL-29 | Windows | Linux |
|---|---|---|
| Instalación limpia | | |
| Arranque y `/api/health` | | |
| Frontend React embebido servido | | |
| Contraseña fuera de `config.json` | | |
| Instancia única | | |
| Cierre sin procesos huérfanos | | |
| Actualización conservando datos | | |
| Desinstalación / borrado | | |
| Firma verificada | **No.** Sin certificado Authenticode. | No aplica. |

## Bloqueos y decisiones

(pendiente de llenar al cerrar la jornada)
