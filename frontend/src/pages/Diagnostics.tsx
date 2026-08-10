import { Link } from 'react-router-dom'
import type { ComponentProps } from 'react'
import { PageError, PageSkeleton } from '../components/AsyncState'
import { Icon } from '../components/Icon'
import { useDiagnostics } from '../hooks/api'
import { formatDate, friendlyJobType } from '../lib/format'
import type { DiagnosticCheck } from '../types'

function iconFor(id: string): ComponentProps<typeof Icon>['name'] {
  if (id === 'chromium') return 'capture'
  if (id === 'storage') return 'folder'
  if (id === 'jobs') return 'activity'
  if (id === 'credentials') return 'shield'
  return 'check'
}

function statusLabel(status: DiagnosticCheck['status']) {
  return status === 'ok' ? 'Correcto' : status === 'warn' ? 'Revisar' : 'Error'
}

function DiagnosticItem({ check }: { check: DiagnosticCheck }) {
  return (
    <div className={`diagnostic-item diagnostic-${check.status}`}>
      <div className="diagnostic-icon"><Icon name={iconFor(check.id)} size={16} /></div>
      <div>
        <strong>{check.title}</strong>
        <p>{check.description}</p>
        {check.detail ? <small>{check.detail}</small> : null}
      </div>
      <span className={`status-chip ${check.status === 'ok' ? 'ok' : check.status === 'warn' ? 'warn' : 'error'}`}>
        {statusLabel(check.status)}
      </span>
    </div>
  )
}

export function Diagnostics() {
  const query = useDiagnostics()

  if (query.isLoading) return <PageSkeleton label="Comprobando el estado local" />
  if (query.isError || !query.data) return <PageError message="No se pudo obtener el diagnóstico local." action={<button className="button ghost small" onClick={() => query.refetch()}>Reintentar</button>} />

  return (
    <div className="diagnostic-layout">
      <section className="card">
        <div className="card-pad">
          <div className="section-toolbar">
            <div>
              <h3>Notificaciones y diagnóstico</h3>
              <p className="helper">Comprobaciones locales, sin enviar credenciales ni contenido a servicios externos.</p>
            </div>
            <button className="button ghost small" onClick={() => query.refetch()} disabled={query.isFetching}>
              {query.isFetching ? 'Comprobando…' : 'Actualizar'}
            </button>
          </div>
          <div className="diagnostic-list">
            {query.data.checks.map((check) => <DiagnosticItem key={check.id} check={check} />)}
          </div>
          <small className="muted diagnostic-updated">Última comprobación: {formatDate(query.data.generatedAt)}</small>
        </div>
      </section>

      <section className="card">
        <div className="card-pad">
          <div className="eyebrow">Incidencias recientes</div>
          <h3 style={{ marginTop: 7 }}>Trabajos que necesitan atención</h3>
          {query.data.incidents.length === 0 ? (
            <div className="empty-state compact"><Icon name="check" size={20} /><p>No hay fallos recientes registrados.</p></div>
          ) : (
            <div className="diagnostic-incidents">
              {query.data.incidents.map((incident) => (
                <Link className="diagnostic-incident" key={incident.jobId} to={`/trabajos/${encodeURIComponent(incident.jobId)}`}>
                  <span><strong>{friendlyJobType(incident.type as import('../types').JobType)}</strong><small>{formatDate(incident.updatedAt)}</small></span>
                  <span className="status-chip error">{incident.errorCode || 'Revisar'}</span>
                </Link>
              ))}
            </div>
          )}
          <div className="route-note" style={{ marginTop: 18 }}>
            <strong>Prueba de Zajuna:</strong> se ejecuta desde la acción explícita de conexión y no durante este sondeo.
          </div>
        </div>
      </section>

      <section className="card">
        <div className="card-pad">
          <div className="eyebrow">Guía rápida</div>
          <h3 style={{ marginTop: 7 }}>Cómo trabajar con la aplicación</h3>
          <div className="route-note" style={{ marginTop: 14 }}><strong>1. Fichas:</strong> elige el curso que quieres preparar.</div>
          <div className="route-note"><strong>2. Checklist:</strong> confirma actividades y rutas.</div>
          <div className="route-note"><strong>3. Evidencias:</strong> revisa capturas y archivos locales.</div>
          <div className="route-note"><strong>4. Reportes:</strong> genera el PDF cuando termines.</div>
        </div>
      </section>
    </div>
  )
}
