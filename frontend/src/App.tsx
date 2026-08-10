import { Navigate, Route, Routes } from 'react-router-dom'
import { useSetupStatus } from './hooks/api'
import { Setup } from './pages/Setup'
import { AppShell } from './components/AppShell'
import { Overview } from './pages/Overview'
import { Fichas } from './pages/Fichas'
import { Checklist } from './pages/Checklist'
import { Activities } from './pages/Activities'
import { Evidences } from './pages/Evidences'
import { Processes } from './pages/Processes'
import { Reports } from './pages/Reports'
import { Settings } from './pages/Settings'
import { Diagnostics } from './pages/Diagnostics'
import { PageSkeleton, PageError } from './components/AsyncState'
import { JobDetail } from './pages/JobDetail'
import { ChecklistItemDetail } from './pages/ChecklistItemDetail'
import { Notifications } from './pages/Notifications'

function App() {
  const { data: setup, isLoading, isError, refetch } = useSetupStatus()

  if (isLoading) {
    return <PageSkeleton label="Comprobando configuración local" />
  }

  if (isError) {
    return <PageError message="No pudimos contactar al core local." action={<button className="button" onClick={() => refetch()}>Reintentar</button>} />
  }

  if (!setup?.setupComplete) {
    return <Setup />
  }

  return (
    <Routes>
      <Route element={<AppShell />}>
        <Route index element={<Navigate to="/resumen" replace />} />
        <Route path="/resumen" element={<Overview />} />
        <Route path="/fichas" element={<Fichas />} />
        <Route path="/checklist" element={<Checklist />} />
        <Route path="/checklist/:itemCode" element={<ChecklistItemDetail />} />
        <Route path="/actividades" element={<Activities />} />
        <Route path="/evidencias" element={<Evidences />} />
        <Route path="/trabajos" element={<Processes />} />
        <Route path="/trabajos/:jobId" element={<JobDetail />} />
        <Route path="/reportes" element={<Reports />} />
        <Route path="/configuracion" element={<Settings />} />
        <Route path="/diagnostico" element={<Diagnostics />} />
        <Route path="/notificaciones" element={<Notifications />} />
        <Route path="*" element={<Navigate to="/resumen" replace />} />
      </Route>
    </Routes>
  )
}

export default App
