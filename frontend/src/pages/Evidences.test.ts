import { describe, expect, it } from 'vitest'
import { dedupeEvidencesByContent } from './Evidences'
import type { Evidence } from '../types'

const row = (id: string, itemCode: string, extra: Partial<Evidence> = {}): Evidence => ({
  id,
  itemCode,
  name: `${id}.png`,
  format: 'png',
  ...extra,
})

describe('dedupeEvidencesByContent', () => {
  it('collapses rows with the same sha256 and lists every covered item', () => {
    const rows = [
      row('a', '1.2.2', { sha256: 'AAA', fileKey: 'k1' }),
      row('b', '1.2.1', { sha256: 'aaa', fileKey: 'k1' }),
      row('c', '10.1.1', { sha256: 'aaa', fileKey: 'k2' }),
      row('d', '6.1', { sha256: 'bbb', fileKey: 'k3' }),
    ]
    const result = dedupeEvidencesByContent(rows)
    expect(result).toHaveLength(2)
    expect(result[0].evidence.id).toBe('a')
    expect(result[0].evidenceIds).toEqual(['a', 'b', 'c'])
    expect(result[0].itemCodes).toEqual(['1.2.1', '1.2.2', '10.1.1'])
    expect(result[1].itemCodes).toEqual(['6.1'])
  })

  it('falls back to the opaque file key when sha256 is missing', () => {
    const rows = [
      row('a', '2.1.1', { fileKey: 'shared' }),
      row('b', '2.1.2', { fileKey: 'shared' }),
      row('c', '2.1.3'),
    ]
    const result = dedupeEvidencesByContent(rows)
    expect(result).toHaveLength(2)
    expect(result[0].itemCodes).toEqual(['2.1.1', '2.1.2'])
    expect(result[1].evidenceIds).toEqual(['c'])
  })
})
