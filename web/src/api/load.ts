export type Load<T> =
  | { status: 'pending' }
  | { status: 'error'; message: string; reload: () => void }
  | { status: 'ready'; data: T }

export function ready<T>(data: T): Load<T> {
  return { status: 'ready', data }
}

export function failed(message: string, reload: () => void = () => {}): Load<never> {
  return { status: 'error', message, reload }
}
