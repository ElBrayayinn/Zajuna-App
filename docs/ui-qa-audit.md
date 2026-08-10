# Auditoría visual, funcional y de accesibilidad

Fecha: 2026-08-08

> Este archivo conserva el historial de las pasadas visuales. El estado
> consolidado y las tareas actuales viven en [`desktop-migration.md`](desktop-migration.md).

## Alcance

Se revisó la interfaz local servida por zajuna-core contra la maqueta
Mockups Zajuna SENA (offline), el diseño aprobado y los flujos disponibles en
localhost.

Skills aplicadas:

- zajuna-app-local: runtime local, estados y validación.
- zajuna-ui-audit: pantallas, estados y diagnóstico.
- design-critique: jerarquía, distribución y densidad.
- design-system: tokens, componentes, iconos y estados.
- accessibility-review: landmarks, etiquetas, foco y teclado.
- ux-copy: lenguaje humano y acciones rápidas.
- testing-strategy: matriz de pruebas por capa.

## Decisiones aplicadas

1. El Resumen combina orientación y acciones rápidas: ficha activa, métricas,
   checklist segmentado, actividad reciente, sincronización, checklist,
   captura y reporte.
2. Fichas concentra selección, búsqueda, sincronización, checklist y
   preparación de evidencias.
3. Checklist usa un riel de categorías, filtros por estado, selector de
   actividades y revisión opcional de rutas.
4. Evidencias mantiene galería agrupada, filtros, vista previa, carga manual y
   eliminación con confirmación.
5. Se reactivó la composición de la maqueta: barra superior con ruta/búsqueda,
   lateral institucional, tarjetas proporcionadas y acciones contextuales.
6. Se añadió skeleton shimmer para el primer estado de carga sin desplazar la
   cuadrícula final.
7. Los iconos de navegación y acciones se generan como SVG inline con trazo
   consistente.

## Resultado de la revisión

| Área | Resultado | Evidencia |
|---|---|---|
| Resumen | Correcto | 4 métricas, checklist segmentado, actividad reciente, selector de ficha y acciones rápidas. |
| Fichas | Correcto | Ficha activa, sincronización, búsqueda, selección, checklist y preparación de evidencias. |
| Checklist | Correcto | Riel de 15 categorías, filtros, selección de actividades, estados y revisión de rutas. |
| Evidencias | Correcto | Galería agrupada, filtros, vista previa, carga manual y eliminación. |
| Trabajos | Correcto | Lista de trabajos, estados, progreso y actualización. |
| Reportes | Correcto | Generación, apertura de PDF y copia local con filas proporcionadas. |
| Configuración | Correcto | Cuenta, preferencias, estado de conexión y datos locales. |
| Diagnóstico | Correcto | Estado local, guía rápida y atención de trabajos fallidos. |

## Accesibilidad verificada

- 1 landmark main y 1 landmark nav en la vista auditada.
- 8 elementos de navegación con foco de teclado.
- Navegación comprobada con Enter y Espacio.
- 0 campos sin etiqueta en la vista auditada.
- 0 imágenes sin alt.
- Iconos SVG visibles con trazo consistente.
- 0 errores o advertencias de consola durante el recorrido limpio.
- Animación reducida mediante prefers-reduced-motion.

## Pruebas ejecutadas

- go test ./... -count=1 -timeout 90s.
- npm run core:build.
- Smoke test local contra http://127.0.0.1:62301/.
- Navegación manual por las ocho secciones.
- Búsqueda de la ficha 3135429.
- Filtros del Checklist y apertura de rutas.
- Vista previa y carga manual de Evidencias.
- Comprobación de navegación por teclado.

## Deuda visual pendiente (registro histórico)

- Separar el HTML estático en componentes React durante la migración final
  (decisión actualizada: React, no Angular).
- Añadir pruebas visuales automatizadas por viewport (resuelto en la
  actualización del 2026-08-08 mediante `test:visual:core`).
- Completar el contenido especializado de las pestañas secundarias de
  Configuración cuando el backend exponga sus contratos.

## Actualización 2026-08-08 — sistema de movimiento y deuda técnica

Esta auditoría anterior decía "Correcto" en todas las áreas, pero no cubría
animación/movimiento ni detectaba deuda técnica real en el JS. Se corrigió lo
siguiente:

- Se detectaron y eliminaron 17 declaraciones de funciones JS duplicadas
  (hasta 4 versiones de la misma función, ej. `renderDashboardShell`,
  `renderFichasView`) — código muerto que nunca se ejecutaba porque en JS
  solo la última declaración de una función gana. El archivo bajó de 166 KB a
  126 KB. Verificado con `node --check` y probando las 8 vistas sin errores
  de consola antes y después.
- Se portaron las 6 animaciones de la maqueta que faltaban (`zjStripes`,
  `zjGrowIn`, `zjLive`/`zjLiveAlert`, `zjRiseIn`, `zjToast`, `zjSoft`); el
  detalle de cada una está en [`design-system.md`](design-system.md#sistema-de-movimiento).
  A diferencia de la maqueta (que usa `infinite` para lucir en una captura
  fija), aquí cada una respeta si el estado real es "mientras dure" o
  "disparo único", para no romperse con el sondeo de 5 s de la aplicación.
- Se reforzó el responsive: `.settings-grid` y `.active-ficha-grid` no
  colapsaban a una columna en móvil; `.job-top` no envolvía el texto largo de
  un trabajo. Verificado sin desbordamiento horizontal en las 8 vistas a
  ancho móvil.
- La pasada de fidelidad contra las 16 pantallas se completó por bloques durante
  la migración a React; queda una revisión manual final de estados que dependen
  de backend real y de cada plataforma.

## Actualización 2026-08-08 (2) — auditoría de fidelidad de 16 pantallas y arreglos rápidos

Se ejecutó una auditoría comparando cada una de las 16 pantallas de la
maqueta contra su vista real, encontrando que la afirmación "Correcto" de la
auditoría original no cubría fidelidad visual real. Hallazgos completos
archivados; resumen:

- 5 causas raíz compartidas identificadas (page-head oculto sin
  condicionar, breadcrumb estático, botón primario con el verde equivocado,
  paleta de status-chip sin distinguir estados). Corregidas 3 de 5 sin
  tocar backend: breadcrumb ahora dinámico (Operación/Sistema + ficha o
  pestaña activa), page-head ya no se duplica en Checklist ni Diagnóstico,
  botón primario usa el verde de marca (#39A900) en los CTA reales de
  Setup/Resumen/Fichas/Checklist/Reportes.
- Bug crítico corregido: `.evidence-gallery-grid` solo tenía definición
  dentro del media query móvil — en escritorio la galería de evidencias se
  apilaba sin grid ni bordes. Ahora tiene grid base de tarjetas.
- Aplicados los 9 patrones visuales repetidos que no requerían backend:
  peso 800 en encabezados, paleta semántica de status-chip (`running`
  verde, `queued` gris neutro, `warn` ámbar distinto de `error`), tarjetas
  del Resumen sin sombra genérica, tipografía monoespaciada en códigos de
  ítem, chevron custom en selects, resaltado de fila de actividad
  seleccionada, riel de 15 categorías (antes limitado a 10) con leyenda,
  chips de rutas/actividades en vez de texto plano, badge "Tarjeta
  enfocada", icono cuadrado del aviso de Setup.
- Pendiente (requiere backend Go y/o vistas nuevas, fuera del alcance de
  esta pasada): pantalla de Detalle de tarea completa (07), tarjetas
  "Trabajo en ejecución"/"Programación local" en Resumen (02) — "Requiere tu
  atención" sí se implementó, ver abajo —, rediseño de Evidencias como
  galería de miniaturas (09), contenido real de las pestañas
  Capturas/Almacenamiento/Copias/Notificaciones en Configuración
  (11-15), paneles reales de Diagnóstico (16).

## Actualización 2026-08-08 (3) — hallazgos estructurales resueltos con datos existentes

Segunda pasada de implementación sobre los hallazgos del audit anterior que
no requerían backend nuevo:

- **Checklist (04)**: ítems agrupados por categoría con divisor sticky
  navy + contador; `renderTask` rediseñado a grid de 5 columnas (código,
  descripción, indicador de slots por segmentos, control de estado
  segmentado Sí/No/Pendiente siempre visible en vez de botón cíclico único);
  barra de progreso de 3 colores con swatches en la cabecera; badge de fase.
- **Resumen (02)**: nueva tarjeta "Requiere tu atención" con ítems
  `status=NO` y trabajos `failed`, calculada desde datos ya cargados
  (`dashboard.items`, `state.jobs`), con acción "Revisar" que navega al
  Checklist filtrado por categoría.
- **Fichas (03)**: tabla ampliada a 7 columnas (se agregaron Curso y
  Actualizada, usando `ficha.courseId`/`ficha.updatedAt` ya existentes),
  barra de cumplimiento en vez de texto plano, pie "Mostrando X de Y
  fichas", badge sólido "FICHA ACTIVA".
- **Trabajos (08)**: fila de chips de filtro por estado con conteo
  (Todos/En curso/En espera/Revisión/Listos/Fallidos).
- **Reportes (10)**: tarjeta "Nuevo reporte" con selector de formato
  (PDF/HTML) y límite de evidencias, conectada a los parámetros
  `format`/`evidenceLimit` que el endpoint ya aceptaba.
- **Evidencias (09)**: chips de filtro por formato de archivo (detectados
  dinámicamente de los datos, sin lista fija).
- **Actividades (06)**: ahora es una vista propia en el sidebar con su
  propio breadcrumb, además de seguir disponible incrustada en Resumen y
  Checklist.
- Bug encontrado y corregido durante la verificación (preexistente, no
  introducido en esta pasada): el nombre de un reporte largo desbordaba
  horizontalmente en Resumen a ciertos anchos intermedios (~940px) porque
  el `<div>` envolvente dentro de `.report-row-copy` no tenía
  `min-width:0`, así que el `text-overflow:ellipsis` del `<strong>` nunca
  se activaba. Verificado sin overflow en las 9 vistas a ancho de
  escritorio y móvil tras el fix.
- Todo verificado con `node --check` antes de cada build y sin errores de
  consola en las 9 vistas (se sumó "Actividades" como novena vista).

## Actualización 2026-08-08 (5) — bloques funcionales conectados

La deuda estructural anterior quedó actualizada con contratos locales y
pruebas:

- **Detalle (07)**: `/checklist/:itemCode` consulta un agregado real que
  devuelve ítem, slots/evidencias y eventos de estado; SQLite schema v12
  persiste cada cambio manual con origen y fecha. `/trabajos/:id` expone el
  timeline persistido, cancelación y estados terminales.
- **Resumen (02)**: la tarjeta de trabajo en ejecución muestra eventos
  recientes y la tarjeta de programación local crea, pausa y sondea schedules
  del core.
- **Configuración (11-15)**: Capturas, renovación de sesión y Notificaciones
  usan `GET/PUT /api/settings`; Almacenamiento y Copias muestran estado real y
  conservan las operaciones locales disponibles.
- **Diagnóstico (16)**: `GET /api/diagnostics` comprueba core, SQLite,
  credencial (solo presencia), Chromium, almacenamiento y fallos de jobs; no
  realiza pruebas remotas durante el polling ni devuelve mensajes sensibles.
- **Evidencias (09)**: `GET /api/evidences?fichaId=…` alimenta miniaturas
  seleccionables y mantiene intacta la agrupación para reportes.

El centro histórico de notificaciones ya está conectado a trabajos locales y
permite marcar avisos como leídos. La gestión de backups ahora valida ZIP,
descarga/elimina y prepara restauraciones seguras para el siguiente arranque;
la pestaña de almacenamiento permite configurar cuántas copias recientes se
conservan y la antigüedad mínima para la limpieza.
El smoke visual `npm run test:visual:core` captura y verifica Resumen en
desktop, tablet y móvil, incluido overflow horizontal, colapso del lateral y
nombres accesibles de controles interactivos. Cada captura se compara con un
SHA-256 de baseline versionado en `ui_smoke_test.go`.
La única deuda de este bloque es la comprobación manual con lector de pantalla
y la ronda completa de revisión WCAG en todas las rutas (ver
`docs/accessibility-audit.md`).

## Actualización 2026-08-08 (4) — migración a React

Se migró la interfaz completa del vanilla JS a React 19 + TypeScript + Vite
en `frontend/`, con el mismo alcance funcional que tenía el vanilla en ese
momento (paridad, no ampliación — las brechas de backend documentadas
arriba siguen pendientes igual). Proceso:

1. Base construida primero (una sola vez, no paralelizable): cliente de API
   tipado con normalización de claves, tipos, hooks de React Query con
   polling de 5 s reemplazando el `refresh()` manual, CSS global portado
   carácter por carácter del `&lt;style&gt;` del vanilla, shell con sidebar/topbar/
   breadcrumb dinámico sobre rutas reales de React Router, y el flujo de
   Setup.
2. Las 9 vistas se migraron en paralelo (un agente por vista, cada uno con
   el código fuente exacto de su función `render*` vanilla como referencia
   de fidelidad), cada una escribiendo únicamente su propio archivo.
3. Verificación de integración: `tsc -b` sin errores en todo el proyecto,
   `npm run build` sin errores, y las 9 rutas probadas en vivo contra el
   core Go real sin errores de consola.

**Bug real encontrado y corregido durante la verificación**: el core Go
devuelve algunas respuestas con claves en PascalCase (`Code`, `Label`...)
mezcladas con camelCase — el vanilla lo resolvía con una función
`normalizeKeys()` en `loadDashboard()` que la migración inicial no había
replicado, dejando las 15 categorías del checklist sin nombre/número en
Resumen y Checklist. Se agregó la misma normalización dentro de
`frontend/src/api/client.ts`, aplicada a toda respuesta de la API (más
amplio que el vanilla, que solo la aplicaba a los endpoints de checklist)
para prevenir el mismo problema en cualquier endpoint futuro.

El empaquetado ya sincroniza `frontend/dist` con el `//go:embed` del core y
mantiene el proxy de Vite solo como flujo de desarrollo. El servidor incluye
fallback para rutas profundas de React y conserva `/api/*` separado. La
validación visual y los nombres accesibles tienen ahora un smoke reproducible;
la conformidad WCAG completa todavía requiere la pasada manual indicada en
`docs/accessibility-audit.md`.
