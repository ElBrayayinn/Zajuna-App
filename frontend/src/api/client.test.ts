import { describe, expect, it } from 'vitest'
import { ApiError, backupDownloadUrl, evidenceDownloadUrl, reportDownloadUrl } from './client'

describe('URL helpers de descarga', () => {
  it('codifica ids de evidencia para la galería', () => {
    expect(evidenceDownloadUrl('abc 123')).toBe('/api/evidences/abc%20123/download')
    expect(evidenceDownloadUrl('ev/with/slash')).toBe('/api/evidences/ev%2Fwith%2Fslash/download')
  })

  it('codifica ids de reporte y nombres de copia', () => {
    expect(reportDownloadUrl('r1')).toBe('/api/reports/r1/download')
    expect(backupDownloadUrl('copia 2026.zip')).toBe('/api/backups/copia%202026.zip/download')
  })
})

describe('ApiError', () => {
  it('conserva estado y ruta para errores de la API', () => {
    const error = new ApiError('No se pudo completar', 503, '/api/evidences/clear')
    expect(error).toBeInstanceOf(Error)
    expect(error.name).toBe('ApiError')
    expect(error.status).toBe(503)
    expect(error.path).toBe('/api/evidences/clear')
    expect(error.message).toBe('No se pudo completar')
  })
})
