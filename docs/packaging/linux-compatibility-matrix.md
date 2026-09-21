# MDL-159 — Matriz de compatibilidad Linux (AppImage)

Issue: [MDL-159](https://linear.app/medialab-sena/issue/MDL-159).
Fuentes de evidencia únicamente:

- [MDL-120](../mdl-120-linux-appimage-2026-09-11.md) — AppImage en **Ubuntu 24.04 bajo WSL2** (no bare-metal).
- [MDL-123](../mdl-123-linux-kali-2026-09-13.md) — **Kali rolling en contenedor Docker** con `/dev/fuse`.
- [`linux-kali-fuse2.md`](linux-kali-fuse2.md) — nota FUSE2 / extract-and-run.

## Reglas (no negociables)

- **Nunca** marcar “soportada” sin corrida documentada (comandos + resultado).
- Contenedor o WSL **no** equivalen a bare-metal/GPU/escritorio nativo.
- Sin evidencia → **no verificable** (≠ aprobada).
- Fallo reproducible → issue propia.

| Etiqueta | Significado |
|---|---|
| Soportada | Casos críticos OK sin ajustes no documentados, en el tipo de entorno probado. |
| Soportada con requisito | OK tras requisito seguro y documentado. |
| No soportada | Falla crítica reproducible o exige desactivar un control de forma inaceptable. |
| No verificable | No hay entorno o evidencia suficiente. |

## Matriz (estado al documentar v0.1.1)

Artefacto de referencia: `Zajuna.App-0.1.1.AppImage`
(ver [linux-release-gate-v0.1.1.md](linux-release-gate-v0.1.1.md)).

| Distribución | Versión / medio | Notas | Arch | Escritorio | FUSE | Resultado | Capacidades | Evidencia |
|---|---|---|---|---|---|---|---|---|
| Kali Linux | rolling (~2026.3) **Docker** + `/dev/fuse` | No bare-metal | x86_64 | headless | fuse3; **sin libfuse2** | **Soportada con requisito** (extract-and-run + wrapper) | Arranque, health, root/no-root sin X11 | MDL-123 |
| Debian | testing/sid (vía hallazgos Kali) | Analogía FUSE solo | x86_64 | — | sin libfuse2 en rolling | **Soportada con requisito** *solo síntoma FUSE2*; **no** QA Debian completa | Diagnóstico FUSE | MDL-123 + fuse2 note |
| Ubuntu | 24.04 LTS **WSL2** | No GNOME nativo | x86_64 | WSL | fuse tras `libfuse2t64` | **Soportada con requisito** *en WSL2* | Smoke packaged, browser, health | MDL-120 |
| Ubuntu LTS bare-metal / VM escritorio | 22.04 / 24.04 | — | x86_64 | GNOME u otro | típ. libfuse2 | **No verificable** | — | Sin corrida fuera de WSL |
| Debian estable bare-metal / VM | bookworm | — | x86_64 | — | — | **No verificable** | — | Sin campaña |
| Fedora / openSUSE / Arch / otras | — | — | — | — | — | **No verificable** | — | Sin evidencia |
| Kali bare-metal / VM con escritorio | — | — | — | X11/Wayland | — | **No verificable** | — | MDL-123 lo deja explícito |

## Qué no se promete

- “Linux genérico” o “todas las LTS”.
- GPU / Wayland nativo.
- Contenedor Kali = Kali con escritorio.
- WSL2 = Ubuntu en máquina física.

## Pendientes para subir de “no verificable”

1. Ubuntu 24.04 LTS VM/bare-metal (GNOME), mismo AppImage hasheado.
2. Debian bookworm VM, con y sin `libfuse2`.
3. Kali VM/bare-metal con sesión gráfica.
4. Otra distro solo con QA real.

Cada fila nueva debe citar fecha, SHA-256 del AppImage, comandos y resultado.
