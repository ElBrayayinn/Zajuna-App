# Acta de comité — 2026-09-11

Proyecto Linear: [Zajuna-App](https://linear.app/medialab-sena/project/zajuna-app-6d193f12dc3d).
Issue madre: [MDL-25](https://linear.app/medialab-sena/issue/MDL-25).

## Decisión

**Release comercial: bloqueado.** M0/M1/M2 de código están cerrados en
`main`. Falta firma Authenticode de Windows (requiere `CSC_LINK` en CI) y
evidencia nativa de Linux (requiere disparar el job `native` con permisos de
admin). macOS sigue fuera de alcance por decisión previa (`macos-deferred.md`,
2026-08-25) hasta contar con Developer ID.

## Hechos de la estación (Windows)

- `main` sincronizado en `da45b03`.
- Se corrió la matriz completa localmente: Go, frontend, oxlint, descargas,
  smoke nativo, browser/visual y el ciclo empaquetar → instalar → usar →
  desinstalar del instalador Windows (ver `release-gate-2026-09-11.md`).
- El baseline visual (`TestDashboardBrowserSmoke`) estaba desactualizado desde
  cambios de UI ya fusionados; se corrigió con revisión manual de las tres
  capturas.
- Se intentó disparar `workflow_dispatch` de `Native installers` vía `gh` para
  obtener evidencia de Linux; la cuenta no tiene permisos de admin sobre el
  repo (HTTP 403). Queda pendiente que alguien con esos permisos lo ejecute.
- Se encontró un hallazgo nuevo: `devDependencies` (Electron 37.10.3 y la
  cadena de `electron-builder`) tienen 5 CVEs de severidad alta que el audit
  de producción no ve. No se fuerza el upgrade (37→44) en este gate porque
  excede el cambio mínimo; queda como tarea de seguimiento propia.

## Issues

| Issue | Estado al cierre de esta acta | Nota |
|---|---|---|
| MDL-25 … MDL-28, MDL-30, MDL-31, MDL-32, MDL-33, MDL-115 | Done | Confirmado en Linear. |
| MDL-29 | In Progress | Windows: instalación/desinstalación limpia y smoke verificados sin firma; Linux sin evidencia nativa en esta corrida; macOS fuera de alcance. |
| MDL-34 | In Progress | Matriz local verde con dos hallazgos documentados (baseline visual corregido; CVEs de `devDependencies` reportados). Falta firma Windows y evidencia nativa Linux para aprobar el release. |
| MDL-124 | Todo | Aún no se aborda en esta sesión. |

## Próximos pasos

1. Configurar `CSC_LINK`/`CSC_KEY_PASSWORD` en los secretos del repo para
   firmar el instalador Windows.
2. Que alguien con permisos de admin dispare `workflow_dispatch` de
   `Native installers` para obtener evidencia real de Linux (y Windows en CI).
3. Abrir una tarea de actualización de Electron (37→44) con su propia
   verificación, en vez de forzarla dentro del gate.
4. Continuar con MDL-124 (reglas de captura del ítem 3.1, cronograma y menú
   de curso).
