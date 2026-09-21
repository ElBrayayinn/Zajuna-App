import { describe, expect, it } from 'vitest'
import {
  confidenceFor,
  confidenceSummary,
  formatDate,
  friendlyJobMessage,
  friendlyJobStage,
  friendlyJobStatus,
  friendlyJobType,
  initials,
  jobStatusClass,
  profileName,
  reportStatusLabel,
  ROUTE_GROUP_LABELS,
  routeStatusClass,
  routeStatusLabel,
  statusClass,
  statusLabel,
} from './format'
import { friendlyError } from './friendlyError'

describe('friendlyJobStage', () => {
  it('maps known stages to Spanish', () => {
    expect(friendlyJobStage('capturing')).toBe('Preparando evidencias')
    expect(friendlyJobStage('waiting_user')).toBe('Necesita tu revisión')
    expect(friendlyJobStage('capturing-checklist')).toBe('Preparando evidencias del checklist')
    expect(friendlyJobStage('')).toBe('Actualización')
  })
})

describe('friendlyJobMessage', () => {
  it('hides raw worker tokens', () => {
    expect(friendlyJobMessage('capture-checklist running')).toMatch(/preparando evidencias/i)
    expect(friendlyJobMessage('sync-fichas queued')).toMatch(/actualizando fichas|en espera/i)
  })

  it('traduce fallos de selector y capturas incompletas', () => {
    expect(friendlyJobMessage('selector requerido cssSelector')).toMatch(/revisión guiada/i)
    expect(friendlyJobMessage('captura incompleta en ficha')).toMatch(/necesitan confirmación/i)
  })
})

describe('status labels', () => {
  it('unifies job and report status for docentes', () => {
    expect(friendlyJobStatus('completed')).toBe('Listo')
    expect(reportStatusLabel('completed')).toBe('Listo')
    expect(reportStatusLabel('failed')).toBe('No se pudo generar')
    expect(reportStatusLabel('ready')).toBe('Listo')
    expect(reportStatusLabel('processing')).toBe('Preparando')
    expect(jobStatusClass('waiting_user')).toBe('review')
    expect(friendlyJobType('capture-checklist')).toBe('Preparar evidencias')
    expect(friendlyJobType('export-report')).toBe('Generar reporte')
  })
})

describe('format helpers usados por Evidencias y Configuración', () => {
  it('formatea fechas vacías y perfil de cuenta', () => {
    expect(formatDate(undefined)).toBe('—')
    expect(formatDate('')).toBe('—')
    expect(profileName({ zajunaUsername: '123456' })).toBe('Cuenta Zajuna')
    expect(profileName({ zajunaUsername: 'docente.sena', profile: {} })).toBe('docente.sena')
    expect(profileName({ profile: { fullName: 'Ana Pérez' } })).toBe('Ana Pérez')
    expect(initials('Ana Pérez')).toBe('AP')
    expect(initials('12345')).toBe('ZA')
    expect(initials(undefined)).toBe('ZA')
  })

  it('etiqueta estados de checklist y rutas', () => {
    expect(statusLabel('SI')).toBe('Cumplida')
    expect(statusLabel('NO')).toBe('No cumplida')
    expect(statusLabel('PENDIENTE')).toBe('Pendiente')
    expect(statusClass('SI')).toBe('si')
    expect(routeStatusLabel('confirmed')).toBe('Confirmada')
    expect(routeStatusLabel('correction')).toBe('Para corregir')
    expect(routeStatusLabel(undefined)).toBe('Por revisar')
    expect(routeStatusClass('confirmed')).toBe('ok')
    expect(ROUTE_GROUP_LABELS.evidencias_aprendizaje).toBe('Evidencias de aprendizaje')
  })

  it('resume confianza de ítems con evidencias', () => {
    expect(confidenceFor({ captureConfidence: 'manual', evidenceCount: 1 })).toMatchObject({
      key: 'manual',
      label: 'Confirmada',
    })
    expect(confidenceFor({ confidence: 'high', evidenceCount: 2 })).toMatchObject({ key: 'high' })
    expect(confidenceFor({ confidence: 'medium', evidenceCount: 1 })).toMatchObject({ key: 'review' })
    expect(confidenceFor({ evidenceCount: 0 })).toMatchObject({ key: 'empty', label: 'Sin evidencia' })
    expect(confidenceFor({ evidenceCount: 3 })).toMatchObject({ key: 'review' })

    expect(
      confidenceSummary([
        { captureConfidence: 'alta', evidenceCount: 1 } as never,
        { confidence: 'review', evidenceCount: 1 } as never,
        { evidenceCount: 0 } as never,
      ]),
    ).toEqual({ high: 1, review: 1, empty: 1 })
  })
})

describe('friendlyError', () => {
  it('avoids technical jargon in user-facing errors', () => {
    const msg = friendlyError('No se pudo contactar el core local.')
    expect(msg.toLowerCase()).not.toContain('core local')
    expect(msg).toMatch(/la aplicación local/i)
    expect(msg.toLowerCase()).not.toMatch(/\bel aplicación\b/)
    expect(friendlyError('selector css no encontrado')).toMatch(/información esperada/i)
  })

  it('cubre bordes de autenticación, WAF y red', () => {
    expect(friendlyError('autenticación fallida: credenciales')).toMatch(/No pudimos conectar con Zajuna/i)
    expect(friendlyError('WAF challenge en portal')).toMatch(/bloqueó temporalmente/i)
    expect(friendlyError('página bloqueada por seguridad')).toMatch(/bloqueó temporalmente/i)
    expect(friendlyError('ECONNREFUSED')).toMatch(/aplicación local/i)
    expect(friendlyError('fetch failed al núcleo local')).toMatch(/aplicación local/i)
    expect(friendlyError('error en 127.0.0.1')).toMatch(/aplicación local/i)
    expect(friendlyError('del core local no responde')).toMatch(/de la aplicación local/i)
    expect(friendlyError('')).toBe('')
  })
})
