import type {
  ActivitiesResponse,
  Dashboard,
  ChecklistItemDetail,
  Diagnostics,
  EvidenceGroup,
  Evidence,
  Ficha,
  Job,
  JobEvent,
  Notification,
  AppSettings,
  Backup,
  Schedule,
  Report,
  RouteReview,
  SetupStatus,
  SetupSaveResponse,
  TargetsResponse,
} from '../types'

export class ApiError extends Error {
  readonly status: number
  readonly path: string

  constructor(message: string, status: number, path: string) {
    super(message)
    this.name = 'ApiError'
    this.status = status
    this.path = path
  }
}

// El core Go a veces devuelve claves en PascalCase (Code, Label, Total...) junto a
// respuestas en camelCase. El vanilla original normalizaba todo antes de usarlo
// (ver normalizeKeys() en index.html) — replicamos exactamente esa normalización
// aquí para que TODOS los datos lleguen en camelCase sin importar el endpoint.
function normalizeKeys<T>(value: T): T {
  if (Array.isArray(value)) return value.map((item) => normalizeKeys(item)) as unknown as T
  if (!value || typeof value !== 'object') return value
  const output: Record<string, unknown> = {}
  Object.entries(value as Record<string, unknown>).forEach(([key, item]) => {
    const normalized = normalizeKey(key)
    output[normalized] = normalizeKeys(item)
  })
  return output as T
}

function normalizeKey(key: string) {
  if (!key) return key
  // Preserve acronym fields from untagged Go structs (`ID`, `URL`, `SHA256`)
  // instead of producing unusable keys such as `iD`.
  if (/^[A-Z][A-Z0-9]*$/.test(key)) return key.toLowerCase()
  const acronym = key.match(/^([A-Z]{2,})(?=[A-Z][a-z]|[0-9]|$)/)?.[1]
  if (acronym) return acronym.toLowerCase() + key.slice(acronym.length)
  return key.charAt(0).toLowerCase() + key.slice(1)
}

async function request<T>(path: string, options: RequestInit = {}): Promise<T> {
  let response: Response
  try {
    response = await fetch(path, options)
  } catch (error) {
    const message = error instanceof Error ? error.message : 'No se pudo contactar el core local.'
    throw new ApiError(message, 0, path)
  }
  const body = await response.json().catch(() => ({}))
  if (!response.ok) {
    const message = typeof body === 'object' && body && 'error' in body && typeof body.error === 'string'
      ? body.error
      : `La operación falló (${response.status})`
    throw new ApiError(message, response.status, path)
  }
  return normalizeKeys(body) as T
}

const json = (body: unknown): RequestInit => ({
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify(body),
})

export const evidenceDownloadUrl = (id: string) => `/api/evidences/${encodeURIComponent(id)}/download`
export const reportDownloadUrl = (id: string) => `/api/reports/${encodeURIComponent(id)}/download`
export const backupDownloadUrl = (name: string) => `/api/backups/${encodeURIComponent(name)}/download`

export const api = {
  getSetupStatus: () => request<SetupStatus>('/api/setup/status'),

  getSettings: () => request<AppSettings>('/api/settings'),

  saveSettings: (input: AppSettings) => request<AppSettings>('/api/settings', { ...json(input), method: 'PUT' }),

  getDiagnostics: () => request<Diagnostics>('/api/diagnostics'),

  listNotifications: () => request<Notification[]>('/api/notifications'),

  markNotificationRead: (id: string) =>
    request<{ read: boolean }>(`/api/notifications/${encodeURIComponent(id)}/read`, { method: 'POST' }),

  markAllNotificationsRead: () =>
    request<{ read: boolean }>('/api/notifications/read-all', { method: 'POST' }),

  listBackups: () => request<Backup[]>('/api/backups'),

  deleteBackup: (name: string) => request<{ deleted: boolean }>(`/api/backups/${encodeURIComponent(name)}`, { method: 'DELETE' }),

  cleanupBackups: (input: { keep?: number; olderThanDays?: number } = {}) => request<{ deleted: string[] }>('/api/backups/cleanup', { ...json(input), method: 'POST' }),

  restoreBackup: (name: string) => request<{ staged: boolean; restartRequired: boolean; backupName: string; safetyBackup: string }>(`/api/backups/${encodeURIComponent(name)}/restore`, { method: 'POST' }),

  saveSetup: (input: { zajunaUsername: string; zajunaDocumentType: string; zajunaPassword: string }) =>
    request<SetupSaveResponse>('/api/setup', json(input)),

  listFichas: (limit = 100) => request<Ficha[]>(`/api/fichas?limit=${limit}`),

  syncFichas: (input: { username: string; documentType: string }) =>
    request<Job>('/api/fichas/sync', json(input)),

  setActiveFicha: (fichaId: string) => request<Dashboard>('/api/fichas/active', json({ fichaId })),

  listJobs: (limit = 8) => request<Job[]>(`/api/jobs?limit=${limit}`),

  getJob: (id: string) => request<Job>(`/api/jobs/${encodeURIComponent(id)}`),

  getJobEvents: (id: string) => request<JobEvent[]>(`/api/jobs/${encodeURIComponent(id)}/events`),

  cancelJob: (id: string) =>
    request<{ cancelled: boolean }>(`/api/jobs/${encodeURIComponent(id)}/cancel`, { method: 'POST' }),

  listSchedules: () => request<Schedule[]>('/api/schedules'),

  createSchedule: (input: { workerType: string; input: Record<string, unknown>; intervalSeconds: number; enabled?: boolean }) =>
    request<Schedule>('/api/schedules', json(input)),

  setScheduleEnabled: (id: string, enabled: boolean) =>
    request<{ enabled: boolean }>(`/api/schedules/${encodeURIComponent(id)}/enabled`, json({ enabled })),

  getDashboard: (fichaId?: string) =>
    request<Dashboard>(`/api/checklist/dashboard${fichaId ? `?fichaId=${encodeURIComponent(fichaId)}` : ''}`),

  getChecklistItemDetail: (itemCode: string, fichaId?: string) =>
    request<ChecklistItemDetail>(`/api/checklist/items/${encodeURIComponent(itemCode)}${fichaId ? `?fichaId=${encodeURIComponent(fichaId)}` : ''}`),

  getActivities: (fichaId: string) =>
    request<ActivitiesResponse>(`/api/checklist/activities?fichaId=${encodeURIComponent(fichaId)}`),

  saveActivities: (input: { fichaId: string; selectedActivityIds: string[] }) =>
    request<ActivitiesResponse>('/api/checklist/activities', { ...json(input), method: 'PUT' }),

  getReviews: (fichaId: string) =>
    request<RouteReview[]>(`/api/checklist/reviews?fichaId=${encodeURIComponent(fichaId)}`),

  saveReview: (input: { fichaId: string; routeKey: string; status: string; manualUrl: string; manualSelector: string }) =>
    request<RouteReview[]>('/api/checklist/reviews', { ...json(input), method: 'PUT' }),

  getTargets: (fichaId: string) =>
    request<TargetsResponse>(`/api/checklist/targets?fichaId=${encodeURIComponent(fichaId)}`),

  setItemStatus: (itemCode: string, input: { fichaId: string; status: string }) =>
    request<Dashboard>(`/api/checklist/items/${encodeURIComponent(itemCode)}/status`, { ...json(input), method: 'PATCH' }),

  capture: (input: { fichaId: string; username: string; documentType: string; itemCodes?: string[] }) =>
    request<Job>('/api/checklist/capture', json(input)),

  discoverCourseMaps: (input: { username: string; documentType: string }) =>
    request<Job>('/api/course-maps/discover', json(input)),

  getEvidenceGroups: (fichaId: string) =>
    request<EvidenceGroup[]>(`/api/evidences/groups?fichaId=${encodeURIComponent(fichaId)}`),

  listEvidences: (fichaId?: string) =>
    request<Evidence[]>(`/api/evidences?limit=100${fichaId ? `&fichaId=${encodeURIComponent(fichaId)}` : ''}`),

  rebuildEvidenceGroups: (fichaId: string) =>
    request<EvidenceGroup[]>('/api/evidences/groups/rebuild', json({ fichaId })),

  uploadEvidence: (form: FormData) => request<Evidence>('/api/evidences/upload', { method: 'POST', body: form }),

  deleteEvidence: (id: string) => request<{ ok: boolean }>(`/api/evidences/${encodeURIComponent(id)}`, { method: 'DELETE' }),

  listReports: (limit = 8) => request<Report[]>(`/api/reports?limit=${limit}`),

  generateReport: (input: { title: string; fichaId: string; format: string; evidenceLimit: number }) =>
    request<Job>('/api/reports', json(input)),

  createBackup: () => request<Backup>('/api/backups', { method: 'POST' }),
}
