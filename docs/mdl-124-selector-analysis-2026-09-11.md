# MDL-124 — Análisis técnico de los selectores fallidos (2026-09-11)

Este documento no cierra [MDL-124](https://linear.app/medialab-sena/issue/MDL-124).
La propia tarea es explícita: el recorte correcto de evidencia SENA lo decide
un instructor, no el código. Esto es el diagnóstico técnico previo a esa
conversación, construido solo a partir de la evidencia ya versionada
(`docs/evidence/mdl-33-selectors.json` y `-curso-b.json`) — no requirió
credenciales en vivo.

## Ítem 3.1 (`disponibilidad`)

Selector primario: `#region-main .course-content .section`, hints:
`["material de trabajo", "evidencias"]` (`core/internal/checklist/targets.go:607`).

Diagnóstico real capturado:

```text
#region-main .course-content .section raw=284 hint=0
#region-main .course-content              raw=1   hint=0
#region-main                              raw=1   hint=0
#page-user-profile                        raw=0   hint=0
.course-content                           raw=1   hint=0
#page-content                             raw=1   hint=0
```

Dos hechos distintos, no uno solo:

1. **`.section` sobre-matchea.** 284 nodos es un número imposible de
   "secciones de curso" reales (un curso Moodle típico tiene entre 10 y 40).
   La clase `.section` de Moodle se reutiliza en muchos contenedores internos
   (bloques, formularios, fragmentos de tema) además de las secciones de
   curso propiamente dichas. El selector necesita algo más específico
   (p. ej. `#region-main .course-content > .section` con hijo directo, o un
   selector de Moodle más moderno si el tema usa `li.section` /
   `[data-for="section"]`).
2. **Ningún nodo, ni siquiera el contenedor completo (`.course-content`,
   raw=1), contiene literalmente "material de trabajo" ni "evidencias".**
   Esto no lo arregla ajustar el selector: la página real probablemente
   describe esa disponibilidad con otro texto (enlaces, nombres de recurso,
   tablas) que no coincide con esas palabras exactas. Sin ver el HTML real o
   sin que un instructor confirme qué contenido cuenta como evidencia de
   "disponibilidad del material de trabajo", cambiar el `labelHint` sería
   adivinar.

## `cronograma_general` y `cronograma_vigente`

Ambos apuntan a rutas `mod/resource/view.php` (una página de recurso
individual — el PDF/Sheet embebido), no a la vista general del curso. Ahí
`#region-main .course-content` ya da `raw=1` en el JSON pero termina
resolviendo por fallback en `#region-main` (el selector con `.section` da 0).
Esto sugiere que la página de un recurso individual no expone la misma
estructura de "secciones" que la vista de curso — es esperable, porque no es
la misma plantilla Moodle. El selector debería apuntar al contenedor real del
recurso (algo como `#region-main .resourceworkaround`, `.generalbox` o el
iframe embebido), no reusar el selector pensado para la vista de curso.

## `menu_curso`

Cae a `#page-content` (el penúltimo o último fallback) en vez de resolver por
`#region-main .course-content`/`.course-content`. Mismo patrón: en esta ruta
(`course/view.php`) el contenedor esperado no aparece con ese nombre de
clase, o el hint `"secciones"` no aparece en su texto.

## Por qué no se cambia el código en esta sesión

Ajustar `.section` a un selector más estricto es una mejora defendible por sí
sola (reduce falsos positivos), pero **no resuelve el fallo real**: sin ver el
HTML real de la página de Zajuna o sin que un instructor confirme qué texto
debe buscarse, cualquier `labelHint` nuevo sería una suposición — exactamente
lo que la tarea pide no hacer. Cambiarlo sin poder correr
`ZAJUNA_CAPTURE_E2E=1` contra un curso real (se necesitan credenciales que no
están disponibles en esta estación) tampoco se podría verificar.

## Qué sí se puede hacer ya, cuando haya credenciales o instructor

1. Correr el registro con `ZAJUNA_SELECTOR_REPORT` en los mismos dos cursos e
   inspeccionar el HTML real de la ruta de 3.1 (¿qué contiene ese `.section`
   que sí es evidencia real de disponibilidad de material?).
2. Con un instructor: decidir el recorte correcto (¿tabla de enlaces?
   ¿nombre del recurso? ¿la sección completa del tema?) y su texto ancla.
3. Ajustar `captureGroupPlan`/`captureSelectorChain` en
   `core/internal/checklist/targets.go` con el selector acordado.
4. Repetir el registro en ambos cursos y confirmar `usedFallback: false` para
   estos tres grupos, sin credenciales ni rutas locales en el resultado.
