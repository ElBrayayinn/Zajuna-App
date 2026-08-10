import { useEffect, useRef } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { ApiError, api } from '../api/client'
import type { JobStatus } from '../types'

const POLL_MS = 5000

function isNotFound(error: unknown) {
  return error instanceof ApiError && error.status === 404
}

function retryTransient(failureCount: number, error: unknown) {
  // A missing active ficha/map is an expected first-run state. Retrying it
  // only delays the empty-state action and creates noisy local traffic.
  return !isNotFound(error) && failureCount < 1
}

export function useSetupStatus() {
  return useQuery({ queryKey: ['setup'], queryFn: api.getSetupStatus })
}

export function useSettings() {
  return useQuery({ queryKey: ['settings'], queryFn: api.getSettings })
}

export function useSaveSettings() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: api.saveSettings,
    onSuccess: (settings) => queryClient.setQueryData(['settings'], settings),
  })
}

export function useDiagnostics() {
  return useQuery({ queryKey: ['diagnostics'], queryFn: api.getDiagnostics, refetchInterval: 15000 })
}

export function useNotifications() {
  return useQuery({ queryKey: ['notifications'], queryFn: api.listNotifications, refetchInterval: POLL_MS })
}

export function useMarkNotificationRead() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: api.markNotificationRead,
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['notifications'] }),
  })
}

export function useMarkAllNotificationsRead() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: api.markAllNotificationsRead,
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['notifications'] }),
  })
}

export function useBackups() {
  return useQuery({ queryKey: ['backups'], queryFn: api.listBackups })
}

export function useDeleteBackup() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: api.deleteBackup,
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['backups'] }),
  })
}

export function useCleanupBackups() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: api.cleanupBackups,
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['backups'] }),
  })
}

export function useRestoreBackup() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: api.restoreBackup,
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['backups'] }),
  })
}

export function useSaveSetup() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: api.saveSetup,
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['setup'] }),
  })
}

export function useFichas() {
  return useQuery({ queryKey: ['fichas'], queryFn: () => api.listFichas(100), refetchInterval: POLL_MS })
}

export function useSyncFichas() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: api.syncFichas,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['fichas'] })
      queryClient.invalidateQueries({ queryKey: ['jobs'] })
    },
  })
}

export function useSetActiveFicha() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: api.setActiveFicha,
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['dashboard'] }),
  })
}

export function useJobs() {
  const queryClient = useQueryClient()
  const previousStatuses = useRef<Record<string, string>>({})
  const query = useQuery({ queryKey: ['jobs'], queryFn: () => api.listJobs(50), refetchInterval: POLL_MS })

  useEffect(() => {
    if (!query.data) return
    const terminal = new Set(['completed', 'failed', 'cancelled'])
    const previous = previousStatuses.current
    const completedCapture = query.data.some((job) => {
      const wasActive = ['queued', 'running', 'waiting_user', 'retrying'].includes(previous[job.id] || '')
      return wasActive && terminal.has(job.status)
    })
    previousStatuses.current = Object.fromEntries(query.data.map((job) => [job.id, job.status]))
    if (!completedCapture) return
    queryClient.invalidateQueries({ queryKey: ['dashboard'] })
    queryClient.invalidateQueries({ queryKey: ['evidenceGroups'] })
    queryClient.invalidateQueries({ queryKey: ['evidences'] })
    queryClient.invalidateQueries({ queryKey: ['activities'] })
    queryClient.invalidateQueries({ queryKey: ['targets'] })
    queryClient.invalidateQueries({ queryKey: ['reviews'] })
    queryClient.invalidateQueries({ queryKey: ['reports'] })
  }, [query.data, queryClient])

  return query
}

export function useJob(jobId?: string) {
  return useQuery({
    queryKey: ['job', jobId],
    queryFn: () => api.getJob(jobId as string),
    enabled: !!jobId,
    retry: retryTransient,
    refetchInterval: (query) => {
      const status = query.state.data?.status
      return status && ['completed', 'failed', 'cancelled'].includes(status) ? false : POLL_MS
    },
  })
}

export function useJobEvents(jobId?: string, enabled = true, jobStatus?: JobStatus) {
  const terminal = jobStatus && ['completed', 'failed', 'cancelled'].includes(jobStatus)
  return useQuery({
    queryKey: ['jobEvents', jobId],
    queryFn: () => api.getJobEvents(jobId as string),
    enabled: !!jobId && enabled,
    retry: retryTransient,
    refetchInterval: enabled && !terminal ? POLL_MS : false,
  })
}

export function useCancelJob() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: api.cancelJob,
    onSuccess: (_data, jobId) => {
      queryClient.invalidateQueries({ queryKey: ['job', jobId] })
      queryClient.invalidateQueries({ queryKey: ['jobEvents', jobId] })
      queryClient.invalidateQueries({ queryKey: ['jobs'] })
    },
  })
}

export function useSchedules() {
  return useQuery({ queryKey: ['schedules'], queryFn: api.listSchedules, refetchInterval: POLL_MS })
}

export function useCreateSchedule() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: api.createSchedule,
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['schedules'] }),
  })
}

export function useSetScheduleEnabled() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ id, enabled }: { id: string; enabled: boolean }) => api.setScheduleEnabled(id, enabled),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['schedules'] }),
  })
}

export function useDashboard(fichaId?: string) {
  return useQuery({
    queryKey: ['dashboard', fichaId ?? 'active'],
    queryFn: () => api.getDashboard(fichaId),
    retry: retryTransient,
    refetchInterval: (query) => (isNotFound(query.state.error) ? false : POLL_MS),
  })
}

export function useChecklistItemDetail(itemCode?: string, fichaId?: string) {
  return useQuery({
    queryKey: ['checklistItemDetail', fichaId, itemCode],
    queryFn: () => api.getChecklistItemDetail(itemCode as string, fichaId),
    enabled: !!itemCode && !!fichaId,
    retry: retryTransient,
  })
}

export function useActivities(fichaId?: string) {
  return useQuery({
    queryKey: ['activities', fichaId],
    queryFn: () => api.getActivities(fichaId as string),
    enabled: !!fichaId,
    retry: retryTransient,
    staleTime: 10_000,
  })
}

export function useSaveActivities() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: api.saveActivities,
    onSuccess: (_data, variables) => {
      queryClient.invalidateQueries({ queryKey: ['activities', variables.fichaId] })
      queryClient.invalidateQueries({ queryKey: ['dashboard'] })
    },
  })
}

export function useReviews(fichaId?: string) {
  return useQuery({
    queryKey: ['reviews', fichaId],
    queryFn: () => api.getReviews(fichaId as string),
    enabled: !!fichaId,
    retry: retryTransient,
  })
}

export function useSaveReview() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: api.saveReview,
    onSuccess: (_data, variables) => {
      queryClient.invalidateQueries({ queryKey: ['reviews', variables.fichaId] })
      queryClient.invalidateQueries({ queryKey: ['targets', variables.fichaId] })
    },
  })
}

export function useTargets(fichaId?: string) {
  return useQuery({
    queryKey: ['targets', fichaId],
    queryFn: () => api.getTargets(fichaId as string),
    enabled: !!fichaId,
    retry: retryTransient,
  })
}

export function useSetItemStatus() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ itemCode, ...input }: { itemCode: string; fichaId: string; status: string }) =>
      api.setItemStatus(itemCode, input),
    onSuccess: (_data, variables) => {
      queryClient.invalidateQueries({ queryKey: ['dashboard'] })
      queryClient.invalidateQueries({ queryKey: ['checklistItemDetail', variables.fichaId, variables.itemCode] })
    },
  })
}

export function useCapture() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: api.capture,
    onSuccess: (_data, variables) => {
      queryClient.invalidateQueries({ queryKey: ['jobs'] })
      queryClient.invalidateQueries({ queryKey: ['dashboard', variables.fichaId] })
      queryClient.invalidateQueries({ queryKey: ['dashboard'] })
      queryClient.invalidateQueries({ queryKey: ['evidenceGroups', variables.fichaId] })
      queryClient.invalidateQueries({ queryKey: ['evidences', variables.fichaId] })
      queryClient.invalidateQueries({ queryKey: ['activities', variables.fichaId] })
      queryClient.invalidateQueries({ queryKey: ['targets', variables.fichaId] })
      queryClient.invalidateQueries({ queryKey: ['reviews', variables.fichaId] })
    },
  })
}

export function useDiscoverCourseMaps() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: api.discoverCourseMaps,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['jobs'] })
      queryClient.invalidateQueries({ queryKey: ['fichas'] })
      queryClient.invalidateQueries({ queryKey: ['dashboard'] })
      queryClient.invalidateQueries({ queryKey: ['activities'] })
      queryClient.invalidateQueries({ queryKey: ['targets'] })
      queryClient.invalidateQueries({ queryKey: ['reviews'] })
    },
  })
}

export function useEvidenceGroups(fichaId?: string) {
  return useQuery({
    queryKey: ['evidenceGroups', fichaId],
    queryFn: () => api.getEvidenceGroups(fichaId as string),
    enabled: !!fichaId,
    retry: retryTransient,
  })
}

export function useEvidences(fichaId?: string) {
  return useQuery({
    queryKey: ['evidences', fichaId],
    queryFn: () => api.listEvidences(fichaId),
    enabled: !!fichaId,
    retry: retryTransient,
  })
}

export function useRebuildEvidenceGroups() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: api.rebuildEvidenceGroups,
    onSuccess: (_data, fichaId) => queryClient.invalidateQueries({ queryKey: ['evidenceGroups', fichaId] }),
  })
}

export function useUploadEvidence() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: api.uploadEvidence,
    onSuccess: (_data, variables) => {
      const fichaId = variables.get('fichaId')
      queryClient.invalidateQueries({ queryKey: ['dashboard'] })
      queryClient.invalidateQueries({ queryKey: ['evidenceGroups'] })
      queryClient.invalidateQueries({ queryKey: ['evidences', fichaId] })
    },
  })
}

export function useDeleteEvidence() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: api.deleteEvidence,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['dashboard'] })
      queryClient.invalidateQueries({ queryKey: ['evidenceGroups'] })
      queryClient.invalidateQueries({ queryKey: ['evidences'] })
    },
  })
}

export function useReports() {
  return useQuery({ queryKey: ['reports'], queryFn: () => api.listReports(8), refetchInterval: POLL_MS })
}

export function useGenerateReport() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: api.generateReport,
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['reports'] }),
  })
}

export function useCreateBackup() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: api.createBackup,
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['backups'] }),
  })
}
