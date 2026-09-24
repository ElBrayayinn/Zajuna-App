# Zajuna App 0.1.5

Base: 0.1.4 ([release-0.1.4.md](release-0.1.4.md)). Integra en una sola
versión cuatro líneas de trabajo: SolucionCache, SolucionCapt,
ReOrganizacionFront y AnalisisProyect.

## Datos locales

Regla de producto sin cambios: **cada versión nueva empieza desde cero**. El
instalador ordena borrar los datos anteriores y el core aplica la misma regla
por versión (`backup.EnforceVersion`), lo que cubre actualizaciones
automáticas y Linux. No hay opción de conservar datos entre versiones: genera
el reporte PDF antes de actualizar si lo necesitas. Las miniaturas nuevas
(`thumbnails/`) también se borran.

## Resumen y flujo guiado (SolucionCache)

- **El Resumen sin ficha activa se quedaba cargando.** Recién instalada, la app
  no tiene ficha activa y `/api/checklist/dashboard` responde 404; la consulta
  volvía a «cargando» al montar cada botón y entraba en bucle (más de 900
  peticiones en 5 s). Ahora no se repite al montar (`retryOnMount: false`).
- Guía de 5 pasos (sincronizar, buscar rutas, seleccionar actividades,
  preparar evidencias, revisar) en el Resumen y en las acciones.
- Solo las actividades técnicas se pueden seleccionar; las transversales
  aparecen bloqueadas.
- Página nueva **Revisión** (`/revision`): revisión automática de cada
  evidencia (aprobada, pendiente, rechazada) y decisión manual.

## Capturas y evidencias (SolucionCapt)

- Foros: 9.1.5–9.1.7 exigen réplicas del instructor; 14.1.x exigen un debate
  de «conclusión»; 9.1.3/9.1.4 exigen fechas visibles.
- Una ausencia real de contenido (la página correcta cargó y no hay nada que
  pruebe el ítem) es un error tipado (`capture.ErrContentAbsent`) y retira la
  evidencia vieja. Una página de error o de permisos de Moodle sigue siendo un
  fallo y conserva la evidencia anterior.
- La revisión automática nunca pisa una decisión manual.
- Cronogramas: la altura se mide dentro del iframe anidado; se abre la pestaña
  de la fase.
- El menú del curso ya no se prepara con clics que Moodle guardaba como
  preferencia del usuario.
- Viewport fijo 2560×1200 por defecto, para capturas reproducibles.

## Diseño y rendimiento (ReOrganizacionFront)

- Menú lateral y paneles derechos fijos al desplazar; enlaces y acciones
  alineados en todas las páginas.
- Carga diferida por página: el paquete inicial pasa de 445 KB a ~310 KB.
- `GET /api/checklist/targets` pasa de ~1 s a ~12 ms (índice de rutas).
- `GET /api/evidences/{id}/thumbnail`: miniaturas JPEG en caché.
- Consulta adaptativa: 5 s con trabajos activos, 30 s en reposo.

## API local y trabajos (AnalisisProyect)

- Las mutaciones exigen siempre un `Origin` loopback. La cookie de capacidad
  solo se entrega en navegaciones de documento del navegador.
- Dos capturas de la misma ficha no corren a la vez; esperar se puede
  cancelar.
- Los resultados de fallo y cancelación devuelven su `Output`.
- `capture.fullPage`, `capture.reuseSession` y `session.autoRenew` de
  Configuración se aplican a la captura. `autoRenew` vuelve a iniciar sesión
  y reintenta una vez cuando la sesión expira.

## Ajustes de integración

- `/checklist` desbordaba a 125 % de escala (test WCAG); corregido.
- Líneas base visuales regeneradas y revisadas en escritorio, tablet y móvil.
