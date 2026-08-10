import type { ReactNode } from 'react'

export function PageSkeleton({ label = 'Cargando contenido' }: { label?: string }) {
  return (
    <div className="skeleton-view" aria-label={label} aria-busy="true">
      <span className="skeleton skeleton-title" />
      <span className="skeleton skeleton-subtitle" />
      <div className="metric-grid" style={{ marginTop: 22 }}>
        {Array.from({ length: 4 }).map((_, index) => (
          <div className="card skeleton-card" key={index}>
            <span className="skeleton skeleton-line short" />
            <span className="skeleton skeleton-value" />
            <span className="skeleton skeleton-line" />
          </div>
        ))}
      </div>
      <span className="skeleton skeleton-line" style={{ marginTop: 20, height: 180 }} />
      <span className="sr-only">{label}</span>
    </div>
  )
}

export function PageError({ message, action }: { message: string; action?: ReactNode }) {
  return (
    <section className="card async-state error-state" role="alert">
      <div className="card-pad">
        <div className="eyebrow">No pudimos cargar esta vista</div>
        <h2 style={{ marginTop: 8 }}>Algo no salió como esperábamos</h2>
        <p className="helper" style={{ marginTop: 8 }}>
          {message}
        </p>
        {action ? <div style={{ marginTop: 16 }}>{action}</div> : null}
      </div>
    </section>
  )
}

