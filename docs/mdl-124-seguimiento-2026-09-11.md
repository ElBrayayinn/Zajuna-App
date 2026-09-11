# MDL-124 — seguimiento post-lanzamiento (2026-09-11)

Tras publicar Zajuna App v0.1.1 con la corrección del ítem 3.1, una corrida real
del checklist completo sobre un curso en producción dejó pendientes 7 criterios
distribuidos en 5 secciones: Seguimiento (8/10), Sesiones en Línea (1/3),
Anuncios (8/9), Documentos (4/5) y Netiqueta (0/1). Esta nota documenta el
diagnóstico en vivo (`ZAJUNA_CAPTURE_E2E` + registro de selectores) contra el
mismo curso y separa dos causas raíz distintas.

## Causa 1 — pistas de texto frágiles (corregida en este cambio)

Los ítems 7.1.1, 7.2, 7.3.2, 7.4.1, 7.4.2, 7.4.3, 7.4.4, 8.2, 8.3, 13.1.3 y
13.2.2 usaban la redacción literal del checklist ("Reporte de Curso", "Actas
de Comité", "Planes de Mejoramiento", "Registro de Novedades", "Llamados de
Atención", "Subsecciones por Fase", "Subsecciones por Fase y Mes",
"Documentos de retención", "Formatos de cierre") como filtro de texto
obligatorio (`RequireSelector`) sobre el selector `#region-main .course-content
.section`.

En el curso real, ese selector resuelve la misma página principal del curso
que otros ítems ya corregidos en MDL-124 (menú, configuración,
disponibilidad): coincide con 284 nodos candidatos y 0 de ellos contiene el
texto exacto del checklist, porque estos criterios describen la
**organización del menú** (una subsección oculta, una subsección ausente, una
carpeta de actas) y no texto que se muestre literalmente en pantalla. Al no
encontrar coincidencia, la app abortaba la captura completa del ítem en lugar
de usar un selector de respaldo — el mismo patrón ya identificado y corregido
para el ítem 3.1.

Corrección: se eliminó la pista de texto obligatoria para estos 11 ítems
(`core/internal/checklist/targets.go`, `captureLabelHints`), dejando que la
ruta ya resuelta por el mapa de curso identifique el destino, igual que se
hizo con el 3.1. Verificado con dos corridas en vivo consecutivas contra el
curso real: los 11 ítems pasaron de fallar de forma dura a capturarse
correctamente.

## Causa 2 — foro incorrecto en Anuncios/Netiqueta/Conclusión (pendiente, ticket nuevo)

Los ítems de Anuncios (11.x), Netiqueta (15.1), Conclusión de foros (14.x) y
parte de Foros (9.1.5–9.1.7) exigen que la publicación encontrada sea del
instructor autenticado (filtro por nombre). En el curso real, el selector
resuelto muestra discusiones de **estudiantes** con títulos genéricos de
presentación/preguntas frecuentes, no las publicaciones reales de Anuncios o
Netiqueta del instructor. Esto indica que estos ítems están resolviendo al
foro equivocado del curso (uno general de estudiantes) en lugar del foro
específico de Anuncios/Netiqueta/Conclusión — un problema de resolución de
rutas, no de comparación de texto.

Esta causa **no se corrige en este cambio**: queda fuera del alcance original
de MDL-124 (ítem 3.1, cronograma, menú de curso) y requiere revisar la lógica
de mapeo de rutas de foros (`core/internal/coursemaps`,
`eligibleRouteForGroup` en `targets.go`) contra varios cursos reales. Se abre
como ticket nuevo de Linear.

## Evidencia

Registro de selectores anonimizado (sin ficha ID) de la corrida posterior a la
corrección: `docs/evidence/mdl-124-seguimiento-verify-2026-09-11.json`.
