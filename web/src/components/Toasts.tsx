export interface Toast {
  id: number
  text: string
  kind: 'done' | 'error'
  undo?: () => void
}

export function Toasts({ toasts, dismiss }: { toasts: Toast[]; dismiss: (id: number) => void }) {
  return (
    <div className="toasts" role="status" aria-live="polite">
      {toasts.map((toast) => (
        <div key={toast.id} className={`toast toast-${toast.kind}`}>
          <span>{toast.text}</span>
          {toast.undo && (
            <button
              type="button"
              className="toast-action"
              onClick={() => {
                toast.undo?.()
                dismiss(toast.id)
              }}
            >
              Desfazer
            </button>
          )}
          <button type="button" className="toast-close" aria-label="Fechar aviso" onClick={() => dismiss(toast.id)}>
            ×
          </button>
        </div>
      ))}
    </div>
  )
}
