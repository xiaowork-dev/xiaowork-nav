import { ref } from 'vue'

export type ToastKind = 'success' | 'error' | 'info'

export interface Toast {
  id: number
  kind: ToastKind
  message: string
}

const toasts = ref<Toast[]>([])
let nextId = 1

function push(kind: ToastKind, message: string, timeout = 3200): void {
  const id = nextId++
  toasts.value.push({ id, kind, message })
  window.setTimeout(() => dismiss(id), timeout)
}

function dismiss(id: number): void {
  toasts.value = toasts.value.filter((t) => t.id !== id)
}

/** Global toast notifications. */
export function useToast() {
  return {
    toasts,
    dismiss,
    success: (message: string) => push('success', message),
    error: (message: string) => push('error', message, 5200),
    info: (message: string) => push('info', message),
  }
}

/** Extracts a human-readable message from an unknown error. */
export function errorMessage(error: unknown): string {
  if (error && typeof error === 'object' && 'message' in error) {
    return String((error as { message: unknown }).message)
  }
  return String(error)
}
