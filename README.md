# Zajuna App

Aplicación de escritorio local para sincronizar Zajuna, revisar el checklist,
capturar evidencias y generar reportes.

La aplicación instalada funciona como un launcher silencioso: inicia el core
Go y abre el localhost en el navegador del sistema. Electron no crea ninguna
ventana ni WebView; solo mantiene el proceso local y evita instancias
duplicadas.

La migración completa, las decisiones técnicas, las tareas abiertas y la
matriz de pruebas están en [`docs/desktop-migration.md`](docs/desktop-migration.md).
El cierre de Linear M0/M1 (2026-08-20) está en
[`docs/hardening-2026-08-20.md`](docs/hardening-2026-08-20.md). La validación
autenticada contra Zajuna real, con el contrato de login de dos pasos y el
registro de selectores por curso, está en
[`docs/mdl-33-2026-08-26.md`](docs/mdl-33-2026-08-26.md).

## Arquitectura actual

```text
Electron
  └─ core Go en 127.0.0.1:<puerto dinámico>
       ├─ React 19 + Vite embebido con go:embed
       ├─ API local same-origin
       ├─ SQLite + migraciones
       ├─ workers, scheduler y eventos
       ├─ evidencias, reportes y backups
       └─ Chromium/Playwright empaquetado
```

No requiere n8n, Docker, MySQL, ngrok, JWT ni un servidor remoto. La única
dependencia externa es la conexión HTTPS de Zajuna. La contraseña se guarda en
el almacén seguro del sistema operativo.

## Desarrollo

```powershell
npm install
npm run frontend:install
npm run browser:install
npm run desktop:dev
```

Modo de navegador local:

```powershell
npm run desktop:start
```

El launcher espera `/api/health` antes de abrir el navegador. El core continúa
escuchando únicamente en loopback y conserva su capability cookie por proceso.
Cerrar la pestaña no detiene el core; volver a ejecutar el acceso directo solo
abre de nuevo la URL de la instancia existente.

Para smoke tests sin abrir una pestaña real: `$env:ZAJUNA_SKIP_EXTERNAL_OPEN =
'1'`.

Para trabajar solo en React:

```powershell
cd frontend
npm run dev
```

## Pruebas

```powershell
npm run build --prefix frontend
npm run lint --prefix frontend
go -C core test ./...
go -C core vet ./...
npm run test:downloads
npm audit --omit=dev --audit-level=high
npm run test:browser:core
```

El smoke visual comprueba Resumen en desktop, tablet y móvil; el smoke
empaquetado inicia el ejecutable, verifica `/api/health` en loopback, el
frontend embebido, los deep links SPA, assets y respuestas 404 de API/static.

### Pruebas contra Zajuna real

El E2E autenticado necesita una cuenta de prueba. Las credenciales van solo en
variables de entorno de la sesión: nunca en un archivo ni en un commit.

```powershell
$env:ZAJUNA_E2E = '1'
$env:ZAJUNA_TEST_DOCUMENT_TYPE = 'CC'
$env:ZAJUNA_TEST_USERNAME = '<documento>'
$env:ZAJUNA_TEST_PASSWORD = '<contraseña>'
npm run test:e2e:zajuna
```

Añadiendo `ZAJUNA_MAPS_E2E=1` se descubren las rutas del curso, y con
`ZAJUNA_CAPTURE_E2E=1` más `ZAJUNA_PLAYWRIGHT_DIR` se ejecutan las capturas
reales. Si además se define `ZAJUNA_SELECTOR_REPORT`, la corrida escribe en esa
ruta el registro de selectores por curso (`docs/evidence/mdl-33-selectors.json`),
sin credenciales, nombres, host ni rutas locales. El procedimiento completo está
en [`docs/mdl-33-2026-08-26.md`](docs/mdl-33-2026-08-26.md).

## Empaquetado

El empaquetado debe ejecutarse en el sistema objetivo para incluir el Chromium
correcto:

```powershell
npm run package:windows
npm run package:linux
```

macOS no es un target de distribuciÃ³n: no se generan DMG/PKG ni se mantiene un
script de empaquetado hasta contar con credenciales Developer ID. `npm run
build:platforms` genera los cores Go para Windows/Linux x64 y ARM64.
`scripts/package.cjs` exige que el runner coincida con la plataforma,
staging del core + Playwright, y genera `dist/release-manifest.json` y
`dist/sbom.cyclonedx.json`. El instalador Windows probado está en `dist/` y
actualmente no tiene firma digital.

El smoke específico del modo externo puede ejecutarse contra un paquete o,
durante desarrollo, contra Electron directamente:

```powershell
npm run test:smoke:external-browser
$env:ZAJUNA_EXTERNAL_DEV = '1'
npm run test:smoke:external-browser
```

## Carpetas principales

- `frontend/`: interfaz React y sistema visual.
- `core/`: API, SQLite, workers, capturas y reportes Go.
- `desktop/`: ciclo de vida Electron y supervisor del core.
- `scripts/`: sincronización, build, staging, metadata y smoke.
- `docs/`: arquitectura, API, auditorías, plan de migración y registro de hardening.
