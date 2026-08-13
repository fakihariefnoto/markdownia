/// <reference types="vite/client" />

interface Window {
  __splashStart?: number
  __currentOutline?: Array<{ anchor: string; text: string; level: number }>
  wails?: {
    Call: any
    Events?: {
      On: (name: string, cb: (payload: any) => void) => void
    }
  }
}
