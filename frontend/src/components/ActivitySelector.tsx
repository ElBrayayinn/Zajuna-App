import { useEffect, useMemo, useState } from 'react'
import { Link } from 'react-router-dom'
import { useSaveActivities } from '../hooks/api'
import { useToast } from '../hooks/useToast'
import { friendlyError } from '../lib/friendlyError'
import type { ActivitiesResponse, Activity } from '../types'

type ActivityFilter = 'all' | 'technical' | 'transversal' | 'selected'

// Los ítems ligados a actividades (6.1, 10.1.1 y 10.1.2) admiten hasta este
// número de evidencias; el core usa las primeras en orden de fase.
const DEFAULT_SLOTS_PER_ITEM = 5

function sameSelection(a: Set<string>, b: Set<string>) {
  if (a.size !== b.size) return false
  for (const id of a) if (!b.has(id)) return false
  return true
}

function savedSelection(data?: ActivitiesResponse) {
  return new Set((data?.activities || []).filter((activity) => activity.selected).map((activity) => activity.id))
}

function groupByPhase(activities: Activity[]) {
  const groups = new Map<string, Activity[]>()
  activities.forEach((activity) => {
    const key = activity.phaseName?.trim() || 'Sin fase identificada'
    const list = groups.get(key) || []
    list.push(activity)
    groups.set(key, list)
  })
  return [...groups.entries()]
}

/**
 * Guided activity picker shared by the Activities page and the Checklist.
 * It owns the draft selection so both screens behave the same way: the user
 * sees what to do, can bulk-select, and knows when changes are unsaved.
 * Render it with key={fichaId} so the draft resets when the ficha changes.
 */
export function ActivitySelector({ data, fichaId }: { data: ActivitiesResponse; fichaId: string }) {
  const toast = useToast()
  const saveActivities = useSaveActivities()
  const saved = useMemo(() => savedSelection(data), [data])
  const [draft, setDraft] = useState<Set<string>>(saved)
  // touched: el usuario editó el borrador y aún no coincide con lo guardado.
  const [touched, setTouched] = useState(false)
  const [query, setQuery] = useState('')
  const [filter, setFilter] = useState<ActivityFilter>('all')
  const [justSaved, setJustSaved] = useState(false)
  const dirty = !sameSelection(draft, saved)

  // Sin ediciones locales reflejamos lo guardado en el core (otra pestaña,
  // refetch). Tras guardar, el refetch hace coincidir ambos y se libera.
  useEffect(() => {
    if (!touched) setDraft(saved)
    else if (sameSelection(draft, saved)) setTouched(false)
  }, [saved, touched, draft])

  const activities = data.activities || []
  const technicalIds = activities.filter((activity) => activity.technical).map((activity) => activity.id)
  const transversalCount = activities.length - technicalIds.length
  const slotsPerItem = Number(data.slotsPerItem) || DEFAULT_SLOTS_PER_ITEM
  const normalizedQuery = query.trim().toLowerCase()
  const visible = activities.filter((activity) => {
    const matchesQuery =
      !normalizedQuery ||
      [activity.id, activity.title, activity.phaseName].some((value) => String(value || '').toLowerCase().includes(normalizedQuery))
    const matchesFilter =
      filter === 'all' ||
      (filter === 'technical' && activity.technical) ||
      (filter === 'transversal' && !activity.technical) ||
      (filter === 'selected' && draft.has(activity.id))
    return matchesQuery && matchesFilter
  })
  const visibleIds = visible.map((activity) => activity.id)
  const narrowed = visible.length !== activities.length
  const allVisibleSelected = visibleIds.length > 0 && visibleIds.every((id) => draft.has(id))
  const hasSavedSelection = saved.size > 0

  function update(next: Set<string>) {
    setDraft(next)
    setTouched(true)
    setJustSaved(false)
  }

  function toggle(id: string) {
    const next = new Set(draft)
    if (next.has(id)) next.delete(id)
    else next.add(id)
    update(next)
  }

  function setMany(ids: string[], selected: boolean) {
    const next = new Set(draft)
    ids.forEach((id) => (selected ? next.add(id) : next.delete(id)))
    update(next)
  }

  function handleSave() {
    const selectedActivityIds = Array.from(draft)
    if (!selectedActivityIds.length) {
      toast('Marca al menos una actividad. Si todas son tuyas, usa “Seleccionar todas”.', true)
      return
    }
    saveActivities.mutate(
      { fichaId, selectedActivityIds },
      {
        onSuccess: () => {
          setJustSaved(true)
          toast(`Guardamos ${selectedActivityIds.length} actividad${selectedActivityIds.length === 1 ? '' : 'es'} a tu cargo.`)
        },
        onError: (error) => toast(friendlyError(error.message), true),
      },
    )
  }

  const status = saveActivities.isPending
    ? { tone: 'ok', text: 'Guardando tu selección…' }
    : dirty
    ? { tone: 'warn', text: 'Tienes cambios sin guardar. Pulsa “Guardar selección” para aplicarlos.' }
    : hasSavedSelection
      ? { tone: 'ok', text: `Selección guardada: ${saved.size} de ${activities.length} actividades. Las evidencias se prepararán solo para estas.` }
      : { tone: 'warn', text: 'Aún no has guardado una selección. Sin este paso no se preparan evidencias de fechas ni de retroalimentación.' }

  return (
    <section className="card activity-selector">
      <div className="card-pad">
        <div className="side-title">
          <div>
            <div className="eyebrow">Antes de preparar evidencias</div>
            <h3 style={{ marginTop: 7 }}>¿Qué actividades orientas tú en esta ficha?</h3>
          </div>
          <span className="badge">{draft.size} de {activities.length} marcadas</span>
        </div>

        <ol className="activity-guide" aria-label="Cómo completar este paso">
          <li>
            <b>1</b>
            <span>
              <strong>Marca solo las actividades que tú calificas.</strong> Con ellas revisamos fechas límite (ítem 6.1) y
              retroalimentación (ítems 10.1.1 y 10.1.2).
            </span>
          </li>
          <li>
            <b>2</b>
            <span>
              <strong>¿Eres el único instructor?</strong> Usa “Seleccionar todas”. Si compartes la ficha, “Solo técnicas”
              suele ser lo correcto: las <em>transversales</em> normalmente las orienta otro instructor.
            </span>
          </li>
          <li>
            <b>3</b>
            <span>
              <strong>Guarda la selección</strong> y después pulsa “Preparar evidencias”. Cada ítem admite hasta {slotsPerItem}{' '}
              evidencias; si marcas más, usamos las primeras en orden de fase.
            </span>
          </li>
        </ol>

        <div className={`activity-status ${status.tone}`} role="status">{status.text}</div>

        <div className="activity-bulk" role="group" aria-label="Selección rápida">
          <button type="button" className="button ghost small" onClick={() => setMany(activities.map((a) => a.id), true)} disabled={!activities.length || draft.size === activities.length}>
            Seleccionar todas ({activities.length})
          </button>
          <button
            type="button"
            className="button ghost small"
            onClick={() => update(new Set(technicalIds))}
            disabled={!technicalIds.length || !transversalCount}
            title="Marca solo las actividades de la competencia técnica y desmarca las transversales"
          >
            Solo técnicas ({technicalIds.length})
          </button>
          {narrowed && visibleIds.length ? (
            <button type="button" className="button ghost small" onClick={() => setMany(visibleIds, !allVisibleSelected)}>
              {allVisibleSelected ? `Desmarcar las ${visibleIds.length} visibles` : `Marcar las ${visibleIds.length} visibles`}
            </button>
          ) : null}
          <button type="button" className="button ghost small" onClick={() => update(new Set())} disabled={!draft.size}>
            Quitar todas
          </button>
          {dirty ? (
            <button type="button" className="button ghost small" onClick={() => update(new Set(saved))}>
              Deshacer cambios
            </button>
          ) : null}
        </div>

        <div className="activity-toolbar">
          <input
            type="search"
            value={query}
            onChange={(event) => setQuery(event.target.value)}
            placeholder="Buscar por código, nombre o fase"
            aria-label="Buscar actividades por código, nombre o fase"
          />
          <select aria-label="Filtrar actividades" value={filter} onChange={(event) => setFilter(event.target.value as ActivityFilter)}>
            <option value="all">Todas</option>
            <option value="technical">Técnicas</option>
            <option value="transversal">Transversales</option>
            <option value="selected">Marcadas</option>
          </select>
        </div>

        <div className="activity-list">
          {!activities.length ? (
            <div className="empty">
              El curso no tiene actividades detectadas. Pulsa “Buscar rutas” para volver a leer el contenido del curso.
            </div>
          ) : visible.length ? (
            groupByPhase(visible).map(([phase, entries]) => {
              const ids = entries.map((entry) => entry.id)
              const phaseSelected = ids.filter((id) => draft.has(id)).length
              return (
                <div className="activity-phase" key={phase}>
                  <div className="activity-phase-head">
                    <strong>{phase}</strong>
                    <span>{phaseSelected} de {ids.length}</span>
                    <button type="button" className="link-button" onClick={() => setMany(ids, phaseSelected !== ids.length)}>
                      {phaseSelected === ids.length ? 'Desmarcar fase' : 'Marcar fase'}
                    </button>
                  </div>
                  {entries.map((activity) => (
                    <label className="activity-row" key={activity.id}>
                      <input
                        type="checkbox"
                        name="activity-id"
                        value={activity.id}
                        checked={draft.has(activity.id)}
                        onChange={() => toggle(activity.id)}
                      />
                      <span>
                        <span className="activity-title">{activity.title || 'Actividad sin título'}</span>
                        <span className="activity-meta">
                          <span>Referencia {activity.id}</span>
                        </span>
                      </span>
                      <span
                        className={`badge ${activity.technical ? '' : 'muted'}`}
                        title={activity.technical ? 'Competencia técnica del programa' : 'Competencia transversal: normalmente la orienta otro instructor'}
                      >
                        {activity.technical ? 'Técnica' : 'Transversal'}
                      </span>
                    </label>
                  ))}
                </div>
              )
            })
          ) : (
            <div className="empty">Ninguna actividad coincide con la búsqueda o el filtro.</div>
          )}
        </div>

        <div className="activity-actions">
          <span className="helper">
            {narrowed ? `Mostrando ${visible.length} de ${activities.length} actividades` : `${activities.length} actividades encontradas`}
          </span>
          <div className="inline">
            {justSaved && !dirty ? (
              <Link className="button ghost small" to="/resumen">
                Siguiente: preparar evidencias →
              </Link>
            ) : null}
            <button
              className="button primary small"
              type="button"
              onClick={handleSave}
              disabled={saveActivities.isPending || (!dirty && hasSavedSelection)}
            >
              {saveActivities.isPending ? 'Guardando…' : 'Guardar selección'}
            </button>
          </div>
        </div>
      </div>
    </section>
  )
}
