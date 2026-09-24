# Evidencias y actualizaciones de la aplicación

## Qué ocurre al actualizar

Actualizar Zajuna App (AppImage o instalador Windows) **reemplaza el binario**,
pero **no borra** la carpeta de datos del usuario:

| Plataforma | Carpeta de datos |
|---|---|
| Linux | `~/.local/share/zajuna-app` (o `$XDG_DATA_HOME/zajuna-app`) |
| Windows | `%LOCALAPPDATA%\ZajunaApp` |

Ahí viven `zajuna.db`, `evidences/`, reportes y respaldos. Por diseño, tras un
update las evidencias **siguen disponibles**.

## Una evidencia vigente por ranura

Desde el esquema v13, la base guarda **una evidencia actual** por
`(ficha, ítem, ranura, origen)`. Las capturas nuevas reemplazan la anterior del
mismo slot; el checklist muestra solo las vigentes y respeta `max_evidences`
del catálogo.

Una misma captura puede respaldar varios ítems (varias filas con el mismo
archivo). Al reemplazar, recortar por `max_evidences`, podar el checklist o
eliminar con `DELETE /api/evidences/{id}`, el archivo solo se borra cuando ya
ninguna fila lo referencia.

## Cómo reiniciar evidencias

Si un docente necesita partir de cero (sin desinstalar):

1. **API:** `POST /api/evidences/clear`  
   Cuerpo opcional: `{ "fichaId": "<id>" }` para limitar a una ficha.  
   Sin `fichaId` elimina todas las evidencias locales y archivos huérfanos.
2. **Interfaz:** Ajustes → Datos → «Borrar evidencias locales».

La actualización de la app **nunca** ejecuta este reinicio sola.

## Restauración de respaldos

Tras restaurar un backup, el núcleo reconciliá `evidences/` con las filas de la
base y elimina archivos que ya no estén referenciados.
