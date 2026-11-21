type TelegramBackButton = {
  show: () => void
  hide: () => void
  onClick: (cb: () => void) => void
  offClick: (cb: () => void) => void
}

type TelegramWebApp = {
  initData?: string
  ready?: () => void
  disableVerticalSwipes?: () => void
  setBackgroundColor?: (color: string) => void
  viewportStableHeight?: number
  onEvent?: (event: string, cb: () => void) => void
  BackButton?: TelegramBackButton
}

declare global {
  interface Window {
    Telegram?: { WebApp?: TelegramWebApp }
  }
}

export function bindTelegram(): void {
  const app = window.Telegram?.WebApp
  app?.ready?.()
  app?.disableVerticalSwipes?.()
  app?.setBackgroundColor?.('#eee7e7')
  const apply = () => {
    const height = app?.viewportStableHeight
    if (height) document.documentElement.style.setProperty('--app-h', `${height}px`)
  }
  apply()
  app?.onEvent?.('viewportChanged', apply)
}
