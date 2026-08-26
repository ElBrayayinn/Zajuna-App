# Firma y smoke de instaladores

Zajuna App distribuye instaladores únicamente para Windows y Linux. macOS no se
empaqueta ni se publica mientras no exista una identidad Developer ID.

## Windows

El job nativo recibe `CSC_LINK` y `CSC_KEY_PASSWORD` exclusivamente desde los
secretos de GitHub Actions. Cuando ambos estén configurados, electron-builder
firma el instalador NSIS y el workflow comprueba que Authenticode sea `Valid`.
No se debe copiar el certificado, su contraseña ni valores de secretos en
issues, logs o artefactos.

Sin `CSC_LINK` el job puede construir y ejecutar el smoke, pero el instalador
no debe considerarse publicable: falta la firma Authenticode.

## Linux

El job genera el AppImage en `ubuntu-latest`, ejecuta el smoke contra la
aplicación empaquetada y publica el manifiesto de release con SHA-256 junto al
SBOM CycloneDX. El checksum publicado es el mecanismo de integridad del
artefacto Linux.

## Evidencia de runner nativo

El workflow manual `Native installers` ejecuta el empaquetado y smoke en
`windows-latest` y `ubuntu-latest`. Conserva como artefactos el instalador,
`release-manifest.json` y `sbom.cyclonedx.json`; esos son los insumos que se
deben adjuntar al gate de release. No se declara un release aprobado si falta
el artefacto, el smoke o, en Windows, la firma válida.
