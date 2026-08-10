import { createContext, useCallback, useContext, useRef, useState, type ReactNode } from 'react'

interface ToastEntry {
  id: number
  message: string
  error: boolean
}

interface ToastContextValue {
  toast: (message: string, error?: boolean) => void
}

const ToastContext = createContext<ToastContextValue | null>(null)

export function ToastProvider({ children }: { children: ReactNode }) {
  const [entries, setEntries] = useState<ToastEntry[]>([])
  const nextId = useRef(0)

  const toast = useCallback((message: string, error = false) => {
    const id = nextId.current++
    setEntries((current) => [...current, { id, message, error }])
    setTimeout(() => {
      setEntries((current) => current.filter((entry) => entry.id !== id))
    }, 4200)
  }, [])

  return (
    <ToastContext.Provider value={{ toast }}>
      {children}
      {entries.map((entry) => (
        <div key={entry.id} className={`toast${entry.error ? ' error' : ''}`} role={entry.error ? 'alert' : 'status'}>
          {entry.message}
        </div>
      ))}
    </ToastContext.Provider>
  )
}

export function useToast() {
  const context = useContext(ToastContext)
  if (!context) throw new Error('useToast debe usarse dentro de ToastProvider')
  return context.toast
}

export function friendlyError(message: string) {
  return String(message || '')
    .replace(/autenticación[^:]*:?\s*/i, 'No pudimos conectar con Zajuna: ')
    .replace(/selector[^.]*\.?/i, 'No encontramos la información esperada en esta página.')
    .replace(/(WAF|página bloqueada)[^.]*\.?/i, 'Zajuna bloqueó temporalmente esta consulta. Inténtalo de nuevo más tarde.')
}
