# Recursos y rendimiento del MVP

## Artefactos de plataforma (2026-08-09)

`npm run build:platforms` genera los cores Go reproducibles en
`dist/core-targets/`:

- `windows-x64/zajuna-core.exe`;
- `linux-x64/zajuna-core`;
- `macos-x64/zajuna-core`;
- `macos-arm64/zajuna-core`.

En esta estación Windows también se generó el instalador NSIS x64 en
`dist/Zajuna App Setup 0.1.0.exe`. El staging final incluye tanto
`zajuna-core.exe` como `core/playwright` (driver y Chromium). El smoke de Go, el
smoke visual responsive y la sincronización `go:embed` pasan con el mismo
build. DMG y AppImage deben
producirse en macOS/Linux respectivamente, porque el runtime Chromium de
Playwright y los firmados del instalador son específicos del sistema; la
compilación cruzada de los cuatro cores no sustituye esa prueba nativa.

Este documento separa lo medido en el artefacto actual de Windows x64 de las
estimaciones que todavía deben confirmarse con una prueba de carga. Los
números cambiarán cuando se optimice el runtime de Playwright y se publiquen
los artefactos de macOS y Linux.

## Instalador y almacenamiento

En la verificación del 6 de agosto de 2026, el artefacto `dist/win-unpacked`
quedó en aproximadamente 1.124 MiB e incluyó:

- Electron y la aplicación.
- `zajuna-core.exe` de aproximadamente 18 MiB.
- Driver de Playwright de aproximadamente 100 MiB.
- Chromium, headless shell y dependencias auxiliares de Windows de
  aproximadamente 789 MiB.

El instalador NSIS x64 generado en la misma prueba quedó en aproximadamente
346 MiB (`Zajuna App Setup 0.1.0.exe`), ahora con Chromium incluido en el
staging determinista.

Por eso el instalador debe reservar al menos 2 GiB libres para instalación,
actualización y temporales. El tamaño comprimido del NSIS será menor, pero debe
medirse por plataforma antes de publicar una versión comercial.

## Perfil mínimo recomendado

| Recurso | Mínimo práctico | Recomendado para uso diario |
|---|---:|---:|
| CPU | 2 núcleos | 4 núcleos |
| RAM | 4 GiB | 8 GiB |
| Almacenamiento libre | 2 GiB para la aplicación | 5–10 GiB considerando evidencias |
| GPU dedicada | No | No; integrada es suficiente |
| Red | Acceso HTTPS a Zajuna | Conexión estable y latencia moderada |

La GPU dedicada no es requisito para las capturas headless. Chromium puede usar
aceleración integrada o renderizado por software; la GPU solo será relevante
si en el futuro se captura una interfaz pesada con video, WebGL o muchos
efectos visuales.

## Consumo esperado por operación

Son rangos iniciales de planificación, no un compromiso de rendimiento:

- Core Go + SQLite sin Chromium: aproximadamente 30–100 MiB de RAM.
- Electron con la interfaz abierta: aproximadamente 100–250 MiB adicionales.
- Una captura PNG autenticada: aproximadamente 200–450 MiB adicionales según
  la página, imágenes y descargas.
- Dos capturas simultáneas: aproximadamente 500–900 MiB adicionales; deben
  probarse antes de habilitarlas como comportamiento predeterminado.
- Descubrimiento de mapas por HTTP: consume bastante menos que Chromium; el
  límite principal es la red y el tamaño de las respuestas HTML.

El runtime actual mantiene la concurrencia general de jobs limitada y cada
lote de checklist captura sus objetivos secuencialmente dentro de una sesión
Chromium reutilizada. En la auditoría E2E de Windows, 10 objetivos dirigidos
de la ficha 3135429 terminaron en 43 segundos, con 10 guardados y 0 errores;
una ejecución anterior de 18 objetivos tardó aproximadamente 53 segundos.
Para el MVP se recomienda una captura Chromium autenticada a la vez. Más CPU
y RAM permitirán dos contextos controlados en el futuro, pero no eliminan el
límite de red ni el riesgo de rate limiting/WAF de Zajuna.

## Escalado futuro

Para hacer más capturas simultáneas sin degradar el equipo:

1. Medir una captura aislada y registrar RAM pico, CPU, duración y tamaño.
2. Reutilizar un proceso Playwright/browser por lote, manteniendo un contexto
   aislado por captura; el MVP ya reutiliza la sesión Chromium dentro del lote.
3. Aplicar una cola con límite configurable, backpressure y cancelación.
4. Separar capturas HTML rápidas de capturas PNG/PDF pesadas.
5. Limitar el número de páginas, descargas y tamaño de artefactos.
6. Exponer en la UI el número de workers Chromium activos y pendientes.

Antes de activar dos o más capturas simultáneas en producción se ejecutará una
prueba de carga en Windows, macOS y Linux con páginas pequeñas, páginas reales
de curso y páginas con imágenes/descargas.
