# Gate de integración — 2026-09-11

Issue: [MDL-34](https://linear.app/medialab-sena/issue/MDL-34).
Estación: Windows 11 Home Single Language, Node v24.18.0, npm 11.16.0, Go
1.27.1.
Commit de trabajo: `da45b03` (rama `ElBrayayinn/linear-review-tasks-cleanup`).

**Decisión: release bloqueado.** No se afirma matriz verde. Windows queda sin
firma Authenticode y no hay evidencia nativa de Linux en esta corrida.

| Comando / acción | Resultado | Notas |
|---|---|---|
| `go -C core test ./...` | exit 0 | Todos los paquetes; ya no hay bloqueo de compilación en captura. |
| `go -C core vet ./...` | exit 0 | |
| `npm run build --prefix frontend` | exit 0 | Vite 8.2.1; no se reprodujo el bloqueo de PostCSS global reportado antes. |
| `npm run lint --prefix frontend` | exit 0, 0 warnings | oxlint 1.77.0. |
| `node scripts/prepare-downloads.test.cjs` | 3 passed | Antes 2/3; ya no hay regresión. |
| `node scripts/smoke-native.test.cjs` | 4 passed | |
| `npm audit --omit=dev --audit-level=high` | 0 vulnerabilidades | Ver hallazgo de `devDependencies` abajo. |
| `npm run test:browser:core` (incluye `TestDashboardBrowserSmoke`) | exit 0 tras corrección | Baseline visual desktop/tablet/mobile estaba desactualizado desde cambios de accesibilidad ya fusionados (post `32f7cbf`); se regeneró y revisó manualmente antes de fijarlo. |
| `npm run package:windows` | exit 0 | `dist/Zajuna App Setup 0.1.0.exe`, ~330 MB. |
| `npm run test:smoke:packaged` | exit 0 | Health, embebido, fallback SPA y 404 API/estático OK. |
| `npm run test:smoke:external-browser` | exit 0 | Modo externo y reutilización de instancia OK. |
| `npm run smoke:native` | exit 0, `releaseBlocked: true` | Sin `CSC_LINK`; firma inválida/ausente reportada correctamente. |
| `npm run test:desktop` | 4 passed | `stop-process-tree`. |
| Instalación silenciosa (`/S`), uso y desinstalación (`/S`) del instalador NSIS | OK | Instaló en `LOCALAPPDATA\Programs\zajuna-app`, `/api/health` respondió, cerró sin procesos huérfanos y la desinstalación no dejó carpeta ni entrada de registro. |
| Authenticode Windows | `NotSigned` | Sin `CSC_LINK`/`CSC_KEY_PASSWORD` en esta estación; comportamiento esperado y documentado en `signing.md`. |
| `npm run package:linux` | no ejecutado | `scripts/package.cjs` exige que el runner coincida con la plataforma; esta estación es Windows. |
| Workflow manual `Native installers` (`workflow_dispatch`) | no disparado | La cuenta que ejecuta este gate no tiene permisos de admin sobre `medialabctm-hub/Zajuna-App` (HTTP 403 al intentar `gh workflow run`). Requiere que alguien con permisos lo dispare. |
| `TestAuthenticatedZajunaE2E` | no ejecutado | Requiere credenciales reales de Zajuna en variables de entorno; no disponibles en esta estación. |
| NVDA / VoiceOver | no ejecutado en esta corrida | Ya cubierto y cerrado por MDL-32 (`accessibility-audit.md`). |

## Hallazgo nuevo: CVEs de alta severidad en `devDependencies`

`npm audit` (sin `--omit=dev`) reporta 5 vulnerabilidades altas: Electron
37.10.3 (múltiples CVEs, incluye uso indirecto de `extract-zip` vulnerable) y
la cadena de `electron-builder` (`xmldom`, `fast-uri`, `js-yaml`). El comando
documentado (`--omit=dev`) no las ve porque `electron` está declarado como
`devDependency`, aunque electron-builder empaqueta ese mismo binario dentro
del instalador final — por lo que sí viaja al usuario final pese a no contarse
en el audit de producción.

`npm audit fix --force` solo resuelve esto subiendo a `electron@44.3.0`, un
salto mayor (37→44) que puede romper `main.cjs`/`desktop/` y excede el cambio
mínimo de este gate. Se deja como hallazgo para una tarea propia de
actualización de Electron con su propia verificación, en vez de forzarlo aquí.

## Corrección aplicada en este gate

`core/cmd/zajuna-core/ui_smoke_test.go`: los hashes de `visualBaselines`
(desktop/tablet/mobile) estaban desactualizados frente a cambios de UI ya
fusionados en `main` (diagnóstico dinámico en el sidebar, estado vacío de
"Resumen", ajustes de accesibilidad y portabilidad en `global.css`) sin que
nadie regenerara el baseline. Se revisaron manualmente las tres capturas
nuevas (sin regresiones, sin overflow, sin elementos sin nombre accesible) y
se actualizaron los tres hashes.

El workflow `.github/workflows/ci.yml` cubre frontend, Go y descargas en cada
PR. El job `native` (Windows y Linux; **sin macOS**, ver `macos-deferred.md`)
solo corre con `workflow_dispatch` y no se pudo disparar desde esta estación
por permisos.

Acta: [`committee-minutes-2026-09-11.md`](committee-minutes-2026-09-11.md).
Firma: [`signing.md`](signing.md). Gate previo:
[`release-gate-2026-08-25.md`](release-gate-2026-08-25.md).
