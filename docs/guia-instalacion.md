# Guía de instalación de Zajuna App

Esta guía acompaña **cada pantalla** desde la descarga hasta el primer uso.
Está pensada para un instructor o usuario final en **Windows 10/11 de 64 bits**.
No hace falta saber programar ni abrir una terminal, salvo para comprobar el
checksum si el equipo de entrega te lo pide.

Hoy el instalador **no tiene firma Authenticode**. Por eso, en algunos equipos
aparece **SmartScreen**. Eso es esperado y temporal: se cierra cuando exista
certificado ([MDL-29](https://linear.app/medialab-sena/issue/MDL-29)). Mientras
tanto, **no desactives** Windows Defender ni SmartScreen de forma permanente.

```text
Descargar → (opcional) SHA256 → Ejecutar .exe
    → ¿SmartScreen? → Más información → continuar
    → Asistente / barra de instalación
    → Abrir Zajuna App → Setup (documento + contraseña)
    → Resumen en el navegador
```

macOS no se entrega. Linux (AppImage) va al final, como anexo.

---

## 1. Antes de instalar

Comprueba esto en el equipo:

| Requisito | Qué significa en la práctica |
|---|---|
| Windows 10 u 11, **64 bits** | En Configuración → Sistema → Acerca de, la edición debe ser x64. |
| **2 GiB libres** como mínimo | El instalador pesa unos 350 MB, pero al descomprimir (Electron + Chromium) ocupa más de 1 GiB. Reserva 2 GiB; 5–10 GiB si vas a capturar muchas evidencias. |
| Conexión a internet | Solo para entrar a Zajuna (HTTPS). La app en sí corre en tu PC (`127.0.0.1`). |
| Cuenta de Zajuna | Tipo de documento, número y contraseña. No se piden en el instalador; salen en el **primer arranque**. |

Descarga **solo** el archivo que te entregue el equipo de Medialab / el canal
oficial. El nombre típico es:

```text
Zajuna App Setup 0.1.0.exe
```

El número de versión puede cambiar. Si te pasan también un
`release-manifest.json`, úsalo en el paso 2. Si no hay manifiesto, salta a
«Ejecutar el instalador» y no inventes un checksum.

---

## 2. Comprobar el archivo (si hay manifiesto)

1. Deja el `.exe` y el `release-manifest.json` en la misma carpeta (por
   ejemplo Descargas).
2. Abre PowerShell en esa carpeta: clic derecho en un espacio vacío +
   **Abrir en Terminal** (o *Abrir la ventana de PowerShell aquí*).
3. Ejecuta, cambiando el nombre si tu archivo es otro:

```powershell
Get-FileHash -Algorithm SHA256 ".\Zajuna App Setup 0.1.0.exe"
```

4. Compara el valor `Hash` con el `sha256` del manifiesto. Deben coincidir
   **exactamente** (mayúsculas o minúsculas no importan).
5. Si no coinciden: **no instales**. Borra el archivo y pide de nuevo el
   instalador oficial.

---

## 3. Ejecutar el instalador

1. En el Explorador de archivos, ve a Descargas (o la carpeta donde lo
   guardaste).
2. Doble clic en `Zajuna App Setup ….exe`.
3. Windows puede pedir permiso de administrador (**Control de cuentas de
   usuario**). Si el diálogo muestra *Zajuna App*, pulsa **Sí**.

Si en vez del instalador aparece una pantalla azul/amarilla de Windows, ve al
paso 4. Si el asistente arranca de una vez, salta al paso 5.

---

## 4. SmartScreen: «Windows protegió su PC»

**Por qué sale.** El instalador actual no lleva firma Authenticode (a veces
llamada AuthCode). Windows no reconoce al editor y, en **algunos** equipos,
bloquea el primer clic. En otros no aparece nada: depende de la versión de
Windows, de SmartScreen y de la política del PC.

**Pantalla que ves.** Un recuadro de Windows, a menudo con un escudo o un
icono de advertencia. Título típico: *Windows protegió su PC*. Texto típico:
*Microsoft Defender SmartScreen impidió que se iniciara una aplicación no
reconocida*. El botón grande suele ser **No ejecutar**.

**Qué pulsas, en este orden:**

1. **Más información** (abajo a la izquierda; en inglés: *More info*).
   Hasta que no pulses eso, no aparece la opción de continuar.
2. Revisa que el nombre del archivo sea el instalador oficial
   (`Zajuna App Setup ….exe`).
3. **Ejecutar de todas formas** (en inglés: *Run anyway*).
4. Si pide administrador, **Sí**.

Eso **no** apaga SmartScreen para siempre. Solo autoriza este archivo en esta
ocasión.

Otras variantes que puedes ver:

- En **Edge**, al descargar: aviso de que el archivo no se suele descargar.
  Usa **…** / **Conservar** / **Mostrar más** y confirma conservar el archivo.
  Luego ábrelo desde Descargas; ahí puede salir el diálogo del paso 4.
- El botón puede decir *Run anyway* si Windows está en inglés.
- Si **no** ves «Más información» y solo hay *No ejecutar*: el equipo tiene
  una política que bloquea ejecutables sin firma. No intentes desactivar
  Defender. Pide al administrador de esa máquina una excepción, o espera el
  instalador firmado (MDL-29).

---

## 5. Asistente de instalación

El empaquetado es NSIS (electron-builder). En la mayoría de equipos verás un
instalador de **un clic** o una barra de progreso corta:

1. Acepta el Control de cuentas de usuario si aún no lo hiciste.
2. Espera a que copie los archivos (puede tardar uno o dos minutos: lleva
   Chromium para las capturas).
3. Al terminar, deja marcada la opción de **ejecutar Zajuna App** si aparece,
   o pulsa **Finalizar**.

Windows crea un acceso en el menú Inicio llamado **Zajuna App**. En muchos
equipos también hay acceso en el escritorio.

Si el asistente muestra *Siguiente* / *Elegir carpeta* / *Instalar*, deja la
ruta por defecto (`Archivos de programa\Zajuna App`) salvo que te hayan
indicado otra.

---

## 6. Primer arranque (Setup)

Zajuna App **no abre una ventana propia**. Electron arranca en segundo plano,
enciende el core en `127.0.0.1` y abre tu **navegador predeterminado**.

1. Si el instalador no la lanzó, abre **Zajuna App** desde el menú Inicio.
2. Espera a que el navegador muestre la app. La dirección será algo como
   `http://127.0.0.1:#####/` (el puerto cambia).
3. En el primer uso verás **Conecta tu cuenta de Zajuna** (título *Tu espacio
   de trabajo local* a la izquierda). Completa:
   - **Tipo de documento:** Cédula de ciudadanía (CC), Tarjeta de identidad
     (TI) o Cédula de extranjería (CE).
   - **Número de documento.**
   - **Contraseña de Zajuna.**
4. Pulsa **Continuar**. Debe aparecer el aviso *Cuenta conectada
   correctamente* y pasar a **Resumen**.

La app no abre un icono en la bandeja: si cierras el navegador, el proceso
sigue en segundo plano (paso 7).

La contraseña se guarda en el almacén del sistema (Credential Manager en
Windows), no en un archivo a la vista. Las fichas, evidencias y reportes
quedan en este equipo.

Si Zajuna pide CAPTCHA u otra verificación, la guía de la propia pantalla te
indica que la resuelvas en el navegador. No hace falta “saltar” ese paso.

---

## 7. Cómo saber que quedó bien

- El navegador muestra **Zajuna App · Operación local** y la vista **Resumen**.
- En el menú lateral puedes abrir Fichas, Checklist, Actividades, Evidencias,
  Reportes, Configuración y Diagnóstico.
- Si cierras **solo la pestaña**, la app **sigue corriendo**. Volver a pulsar
  el acceso directo reabre la misma URL; no instala una segunda copia.

**Cómo salir de verdad.** Cerrar Chrome/Edge no apaga el core. En Windows:

1. Abre el Administrador de tareas (`Ctrl` + `Mayús` + `Esc`).
2. Busca **Zajuna App**.
3. Finaliza esa tarea.

La próxima vez que uses el acceso directo, volverá a arrancar limpio.

---

## 8. Si algo falla

| Qué ves | Qué hacer |
|---|---|
| SmartScreen sin «Más información» | Política del equipo. No desactives Defender. Pide excepción o el instalador firmado. |
| «No hay espacio suficiente» | Libera al menos 2 GiB en `C:` y vuelve a ejecutar el Setup. |
| El antivirus pone el `.exe` en cuarentena | Restaura el archivo desde la cuarentena **solo** si viene del canal oficial. No añadas exclusiones globales. |
| El instalador se corta a medias | Desinstala (paso 9), borra la carpeta a medias en Archivos de programa si quedó, y reintenta. |
| El navegador no abre | Abre **Zajuna App** otra vez. Si sigue igual, revisa Diagnóstico cuando logres entrar, o el log en `%APPDATA%\zajuna-app\logs\zajuna-core.log`. |
| La página queda en blanco / no carga | Confirma que no cerraste el proceso de Zajuna App. Prueba de nuevo el acceso directo. |
| Error al guardar la cuenta | Revisa documento y contraseña de Zajuna, y que haya internet. El Setup no sustituye el login de la plataforma. |
| «Ya hay una instancia» / no pasa nada | La app ya está corriendo: mira las pestañas del navegador o el Administrador de tareas. |

---

## 9. Desinstalar y volver a intentar

1. Configuración de Windows → **Aplicaciones** → **Aplicaciones instaladas**.
2. Busca **Zajuna App** → **Desinstalar**.
3. Confirma el desinstalador.
4. Vuelve al paso 3 con el mismo `.exe` oficial (o uno nuevo que te pasen).

La desinstalación puede dejar datos locales (cuenta, evidencias, backups) en
`%APPDATA%\zajuna-app`. Si quieres un equipo limpio, borra esa carpeta
**después** de desinstalar y solo si no necesitas esos datos.

---

## 10. Anexo: Linux (AppImage)

En Linux el artefacto es un AppImage, no un Setup. Entorno validado: Kali
Linux (basado en Debian). Comprueba el SHA256 del manifiesto, marca el
archivo como ejecutable (`chmod +x`) y ábrelo. No hay SmartScreen. Si el
escritorio bloquea un binario no firmado, no eludas esa protección: usa el
canal oficial y el checksum.

### Requisitos del equipo

Las capturas usan Chromium/Puppeteer, así que el consumo depende también de
cuántas instancias corran en paralelo:

| Recurso | Mínimo | Recomendado |
|---|---|---|
| Procesador | 4 núcleos | 8 núcleos |
| Memoria RAM | 4 GB | 8 GB |
| Almacenamiento | 2 GB libres dedicados a la app | + espacio adicional para capturas (varía según resolución, formato y frecuencia) |

Con 8 núcleos y 8 GB hay margen para dos o tres capturas simultáneas; con el
mínimo, evita abrir muchas a la vez. No trates los 2 GB como el espacio total
de operación: reserva más si vas a capturar mucho y supervisa la carpeta de
evidencias.

### Primera ejecución

```bash
cd ~/Downloads   # o ~/Descargas
ls -lh
chmod +x ZajunaApp.AppImage
ls -l ZajunaApp.AppImage      # confirmar el permiso
./ZajunaApp.AppImage
```

No la ejecutes como root salvo necesidad técnica documentada; debe iniciarse
con el usuario habitual del sistema.

Comprobación rápida de recursos antes de instalar:

```bash
nproc      # CPU disponible
free -h    # memoria RAM
df -h .    # espacio libre
```

### Instalación opcional en una ubicación global

AppImage no necesita "instalarse" para funcionar, pero si quieres iniciarla
desde cualquier terminal:

```bash
sudo mkdir -p /opt/ZajunaApp
sudo mv ~/Downloads/ZajunaApp.AppImage /opt/ZajunaApp/ZajunaApp.AppImage
sudo chmod +x /opt/ZajunaApp/ZajunaApp.AppImage
sudo ln -sf /opt/ZajunaApp/ZajunaApp.AppImage /usr/local/bin/zajunaapp
```

Después se inicia con `zajunaapp` desde cualquier carpeta.

### Actualizar

Reemplaza el archivo conservando el mismo nombre y repite el permiso de
ejecución:

```bash
sudo cp ZajunaApp.AppImage /opt/ZajunaApp/ZajunaApp.AppImage
sudo chmod +x /opt/ZajunaApp/ZajunaApp.AppImage
```

### Solución de problemas comunes

**"Permission denied" al ejecutar** — falta el permiso de ejecución:

```bash
chmod +x ZajunaApp.AppImage
./ZajunaApp.AppImage
```

**El AppImage pide FUSE o no puede montarse** — algunas instalaciones mínimas
de Linux no traen `libfuse2`/`fuse3` por defecto:

```bash
sudo apt update
apt search fuse | grep -E "libfuse|fuse"
```

Si el sistema no tiene esa compatibilidad, usa la extracción temporal como
diagnóstico o solución alterna:

```bash
./ZajunaApp.AppImage --appimage-extract-and-run
```

Para reportar un problema, incluye distribución y versión de Linux,
arquitectura, el mensaje exacto de la terminal y los pasos previos. Nunca
incluyas credenciales, tokens ni información sensible.

macOS no forma parte de este release. Ver [`macos-deferred.md`](macos-deferred.md).

---

## Relación con otras tareas

- [MDL-29](https://linear.app/medialab-sena/issue/MDL-29): firma Authenticode.
  Cuando exista, este aviso de SmartScreen debería desaparecer y esta guía se
  actualizará.
- [MDL-28](https://linear.app/medialab-sena/issue/MDL-28): la página pública
  `downloads.html` **no** enseña a continuar SmartScreen; trata el artefacto
  sin firma como bloqueo de release comercial. Esta guía es el camino
  **operativo** para candidatos internos / instructores mientras no haya firma.

Desarrollo en esta estación (npm, Go, Electron): [`run-local.md`](run-local.md).
