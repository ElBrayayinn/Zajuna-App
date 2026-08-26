# Acta de comité — accesibilidad MDL-32 (2026-08-26)

**Tipo:** cierre de evidencia de auditoría WCAG 2.1 AA (teclado, zoom, reflow y NVDA).  
**Candidato:** worktree `mdl-32-m2-ejecutar-auditor-a-manual-wcag-2.1-aa`.  
**Runtime:** Chromium empaquetado de Playwright + NVDA 2026.1.1 portable.

## Decisión

Queda cerrada la evidencia de teclado, zoom 200 %, reflow a 320 CSS px y **NVDA en Windows** sobre las nueve rutas. **No se declara conformidad WCAG 2.1 AA completa:** VoiceOver queda para un runner macOS en otro día.

## Evidencia ejecutada

- `TestWCAGKeyboardMatrix` (PASS): skip link, `main`, `h1`, toasts live, 44 px, 320 px, zoom 200 %, flechas en Configuración.
- `TestNVDAScreenReaderPass` (PASS, 2026-08-26 11:27): NVDA en español anunció el título de ventana, el enlace “Saltar al contenido”, la región “Navegación principal” y los enlaces Resumen, Fichas, Checklist, Actividades y Evidencias.
- `TestDashboardBrowserSmoke` con hashes desktop `46f04e57…`, tablet `b2f5b778…`, mobile `33f27618…`.

### Fragmentos de voz NVDA (log IO)

- `Zajuna App · Operación local - Google Chrome for Testing`
- `Saltar al contenido`, `misma página`, `enlace`
- `Navegación principal`, `navegación región`, `OPERACIÓN`
- `enlace Resumen` / `Checklist` / `Actividades` (`página actual`) / `Evidencias`

NVDA también leyó la UI de Orca cuando el foco salió de Chromium; eso no es un defecto del producto.

## Hallazgos

| Severidad | Hallazgo | Acción |
|---|---|---|
| P2 (remediado) | NVDA concatenaba el contador: “Fichas0” | `aria-label="Fichas, N"` y cifra `aria-hidden` |
| Residual | Acciones `.button.small` &lt; 44 CSS px | No bloquea AA 2.1 |
| Bloqueo | VoiceOver | macOS otro día |

No se abren issues hijas P0/P1.

## Conformidad condicionada

Condicionada a VoiceOver en macOS. El release no debe afirmar “100 % WCAG 2.1 AA” hasta esa pasada.
