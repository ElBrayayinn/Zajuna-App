import { useState } from 'react'
import { Link } from 'react-router-dom'
import { PageError, PageSkeleton } from '../components/AsyncState'
import { useJobs } from '../hooks/api'
import { friendlyJobMessage, friendlyJobStatus, friendlyJobType, jobStatusClass } from '../lib/format'
import type { Job, JobStatus } from '../types'

type JobFilterValue = 'all' | 'running' | 'queued' | 'waiting_user' | 'retrying' | 'completed' | 'failed' | 'cancelled'

const JOB_FILTER_CHIPS: { value: JobFilterValue; label: string }[] = [
  { value: 'all', label: 'Todos' },
  { value: 'running', label: 'En curso' },
  { value: 'queued', label: 'En espera' },
  { value: 'waiting_user', label: 'Revisión' },
  { value: 'retrying', label: 'Reintentando' },
  { value: 'completed', label: 'Listos' },
  { value: 'failed', label: 'Fallidos' },
  { value: 'cancelled', label: 'Cancelados' },
]

function JobRow({ job }: { job: Job }) {
  const progress = Math.max(0, Math.min(100, Number(job.progress) || 0))
  const status = friendlyJobStatus(job.status)
  const statusClassName = jobStatusClass(job.status)
  const isRunning = job.status === 'running'
  const isWaiting = job.status === 'waiting_user' || job.status === 'retrying'
  const chipClass = `status-chip ${statusClassName}${isWaiting ? ' waiting-pulse' : ''}`

  return (
    <article className="job">
      <div className="job-top">
        <strong>{friendlyJobType(job.type)}</strong>
        <span className={chipClass}>
          {isRunning ? <i className="live-pulse" aria-hidden="true" /> : null}
          {status}
        </span>
      </div>
      <small>
        {friendlyJobMessage(job.message || job.stage)} · {progress}%
      </small>
      <div className={`progress${isRunning ? ' running' : ''}`}>
        <i style={{ width: `${progress}%` }} />
      </div>
      <div className="job-row-actions">
        <Link className="button ghost small" to={`/trabajos/${encodeURIComponent(job.id)}`}>
          Ver detalle
        </Link>
      </div>
    </article>
  )
}

export function Processes() {
  const jobsQuery = useJobs()
  const jobs = jobsQuery.data ?? []
  const [jobFilter, setJobFilter] = useState<JobFilterValue>('all')

  if (jobsQuery.isLoading) return <PageSkeleton label="Cargando trabajos locales" />
  if (jobsQuery.isError) return <PageError message="No pudimos cargar los trabajos locales." action={<button className="button" onClick={() => jobsQuery.refetch()}>Reintentar</button>} />

  const jobCounts: Partial<Record<JobStatus, number>> = {
    running: 0,
    queued: 0,
    waiting_user: 0,
    retrying: 0,
    completed: 0,
    failed: 0,
    cancelled: 0,
  }
  jobs.forEach((job) => {
    if (job.status in jobCounts) {
      jobCounts[job.status] = (jobCounts[job.status] ?? 0) + 1
    }
  })

  const filteredJobs = jobFilter === 'all' ? jobs : jobs.filter((job) => job.status === jobFilter)

  return (
    <div className="grid">
      <section className="card">
        <div className="card-pad">
          <div className="side-title">
            <div>
              <h3>Procesos recientes</h3>
              <p className="helper" style={{ marginTop: 5 }}>
                Aquí puedes seguir la actualización de fichas, la revisión de rutas, las capturas y los reportes.
              </p>
            </div>
          <button className="button ghost small" onClick={() => jobsQuery.refetch()}>
              Actualizar
            </button>
          </div>
          <div className="checklist-filter-tabs" style={{ marginTop: 14 }}>
            {JOB_FILTER_CHIPS.map(({ value, label }) => (
              <button
                key={value}
                type="button"
                className={`checklist-filter-tab ${jobFilter === value ? 'active' : ''}`}
                aria-pressed={jobFilter === value}
                onClick={() => setJobFilter(value)}
              >
                {label} {value === 'all' ? jobs.length : (jobCounts[value] ?? 0)}
              </button>
            ))}
          </div>
          <div className="job-list" style={{ marginTop: 15 }}>
            {filteredJobs.length ? (
              filteredJobs.map((job) => <JobRow key={job.id} job={job} />)
            ) : (
              <div className="empty">No hay procesos con este filtro.</div>
            )}
          </div>
        </div>
      </section>
      <section className="card">
        <div className="card-pad">
          <h3>¿Qué significa cada estado?</h3>
          <div className="route-note" style={{ marginTop: 12 }}>
            <strong>En espera:</strong> el proceso está pendiente de turno.
          </div>
          <div className="route-note">
            <strong>En curso:</strong> la aplicación está trabajando.
          </div>
          <div className="route-note">
            <strong>Necesita tu revisión:</strong> debes confirmar una ruta o resolver una verificación.
          </div>
          <div className="route-note">
            <strong>Listo:</strong> el resultado quedó guardado en este equipo.
          </div>
        </div>
      </section>
    </div>
  )
}
