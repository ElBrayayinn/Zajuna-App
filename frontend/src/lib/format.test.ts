import { describe, expect, it } from 'vitest'
import { friendlyJobMessage, friendlyJobStage, friendlyJobStatus, reportStatusLabel } from './format'
import { friendlyError } from './friendlyError'

describe('friendlyJobStage', () => {
  it('maps known stages to Spanish', () => {
    expect(friendlyJobStage('capturing')).toBe('Preparando evidencias')
    expect(friendlyJobStage('waiting_user')).toBe('Necesita tu revisión')
    expect(friendlyJobStage('')).toBe('Actualización')
  })
})

describe('friendlyJobMessage', () => {
  it('hides raw worker tokens', () => {
    expect(friendlyJobMessage('capture-checklist running')).toMatch(/preparando evidencias/i)
    expect(friendlyJobMessage('sync-fichas queued')).toMatch(/actualizando fichas|en espera/i)
  })
})

describe('status labels', () => {
  it('unifies job and report status for docentes', () => {
    expect(friendlyJobStatus('completed')).toBe('Listo')
    expect(reportStatusLabel('completed')).toBe('Listo')
    expect(reportStatusLabel('failed')).toBe('No se pudo generar')
  })
})

describe('friendlyError', () => {
  it('avoids technical jargon in user-facing errors', () => {
    const msg = friendlyError('No se pudo contactar el core local.')
    expect(msg.toLowerCase()).not.toContain('core local')
    expect(friendlyError('selector css no encontrado')).toMatch(/información esperada/i)
  })
})
