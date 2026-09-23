import { describe, expect, it } from 'vitest'
import { computeWorkflow, currentWorkflowStep } from './workflow'

const base = { fichasCount: 0, hasActiveFicha: false, syncRunning: false, mapReady: false, discoverRunning: false, selectedActivities: 0, evidenceCount: 0, captureRunning: false }

describe('computeWorkflow', () => {
  it('starts at step 1 with nothing done', () => {
    expect(currentWorkflowStep(computeWorkflow(base))?.key).toBe('sync')
  })

  it('points to exactly one next step in order', () => {
    const steps = computeWorkflow({ ...base, fichasCount: 3, hasActiveFicha: true, mapReady: true })
    expect(steps.map((step) => step.state)).toEqual(['done', 'done', 'current', 'pending', 'pending'])
  })

  it('shows a running step instead of the next one', () => {
    const steps = computeWorkflow({ ...base, fichasCount: 3, hasActiveFicha: true, discoverRunning: true })
    expect(currentWorkflowStep(steps)?.key).toBe('routes')
    expect(steps[1].state).toBe('running')
  })

  it('finishes when every evidence is reviewed without problems', () => {
    const steps = computeWorkflow({ ...base, fichasCount: 1, hasActiveFicha: true, mapReady: true, selectedActivities: 4, evidenceCount: 20, reviewOpen: 0, reviewTotal: 20 })
    expect(steps.every((step) => step.state === 'done')).toBe(true)
    expect(currentWorkflowStep(steps)).toBeUndefined()
  })

  it('keeps review as the next step while evidences need attention', () => {
    const steps = computeWorkflow({ ...base, fichasCount: 1, hasActiveFicha: true, mapReady: true, selectedActivities: 4, evidenceCount: 20, reviewOpen: 7, reviewTotal: 20 })
    expect(currentWorkflowStep(steps)?.key).toBe('review')
  })
})
