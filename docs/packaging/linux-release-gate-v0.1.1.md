# MDL-160 — Gate de release Linux v0.1.1

Issue: [MDL-160](https://linear.app/medialab-sena/issue/MDL-160).
Matrices / permisos: [linux-compatibility-matrix.md](linux-compatibility-matrix.md),
[permissions-xdg-sandbox.md](permissions-xdg-sandbox.md).
Bloqueante de auditoría: [MDL-176](https://linear.app/medialab-sena/issue/MDL-176).

## Artefacto evaluado

| Campo | Valor |
|---|---|
| Nombre | `Zajuna.App-0.1.1.AppImage` |
| Versión | `0.1.1` |
| Tamaño (bytes) | `446415066` |
| SHA-256 | `4ca5c5d68d69d57a05297ed7443046b310c6273d53c6be133accdd01d8ffd1f4` |
| Canal | GitHub Releases `medialabctm-hub/Zajuna-App` (cuando se publique) |

```bash
sha256sum Zajuna.App-0.1.1.AppImage
# esperado:
# 4ca5c5d68d69d57a05297ed7443046b310c6273d53c6be133accdd01d8ffd1f4
```

Cualquier smoke o decisión posterior **debe** citar este hash.

## Decisión

**Go con observaciones.**

Sin bloqueador conocido de instalación/arranque/pérdida de datos en los
entornos **sí** ejercidos (WSL2 Ubuntu MDL-120, Kali contenedor MDL-123).
Riesgos residuales explícitos; trabajo abierto en seguridad/sandbox (**MDL-176**).

### Observaciones (obligatorias en notas de release)

1. **Sandbox:** el AppImage fuerza `--no-sandbox` vía wrapper. No satisface al
   pie el AC de MDL-161. Riesgo residual → **MDL-176**.
2. **FUSE2 ausente en Kali/Debian rolling:** `--appimage-extract-and-run`.
3. **Cobertura de distros:** Ubuntu LTS / Debian estable / Kali desktop
   bare-metal siguen **no verificable**.
4. **Firma Windows:** Authenticode/CSC opcional; auto-update sin firma puede
   fallar (SmartScreen).
5. **UI en navegador externo:** smoke = health loopback + navegador cuando aplique.

## Checklist de gates M3

| # | Gate | Resultado | Comentario |
|---|---|---|---|
| 1 | Integridad AppImage | **OK (candidato)** | Nombre + tamaño + SHA-256 |
| 2 | Arranque / health usuario estándar | **OK con requisitos** | MDL-120/123; FUSE/wrapper |
| 3 | Persistencia tras update | **OK por diseño** | `~/.local/share/zajuna-app` |
| 4 | Concurrencia / recursos | **Observación** | No re-medido en este gate |
| 5 | Compatibilidad por distro | **Parcial** | Solo WSL2 Ubuntu + Kali contenedor |
| 6 | Seguridad / sandbox / permisos | **Observación → MDL-176** | `--no-sandbox` embebido |
| 7 | Regresión / evidencia | **Parcial** | Campaña MDL-123 |
| 8 | Hallazgos críticos/altos abiertos | **Pendiente MDL-176** | No “apto sin reservas” |

## Regla de salida

- **Go:** no aplica (sandbox + auditoría + matriz incompleta).
- **Go con observaciones:** **sí**.
- **No-Go:** no aplica salvo hallazgo de pérdida de datos / RCE / no-instalable.

## Próximos pasos

1. Cerrar o aceptar riesgo formal en **MDL-176**.
2. Completar filas bare-metal/VM en **MDL-159**.
3. Re-smoke tras merge de Electron 44 + `artifactName` sin espacios + icono;
   si el hash cambia, abrir gate con cifra nueva.
4. Notas de release citando **Go con observaciones** y este documento.

## Trazabilidad

No se inventaron passes: todo “OK” remite a MDL-120, MDL-123 o al hash declarado.
