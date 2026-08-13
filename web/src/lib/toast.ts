// toast.ts — thin wrapper over sonner matching the vanilla toast() API.

import { toast as sonner } from "sonner"

type ToastOpts = {
  type?: "success" | "error" | "warning"
  title: string
  caption?: string
  sticky?: boolean
  action?: { label: string; onClick: () => void }
}

export function toast(opts: ToastOpts) {
  const msg = opts.caption ? `${opts.title} — ${opts.caption}` : opts.title
  switch (opts.type) {
    case "error":
      sonner.error(msg, { duration: opts.sticky ? Infinity : 4000, action: opts.action ? { label: opts.action.label, onClick: opts.action.onClick } : undefined })
      return
    case "warning":
      sonner.warning(msg, { duration: opts.sticky ? Infinity : 4000, action: opts.action ? { label: opts.action.label, onClick: opts.action.onClick } : undefined })
      return
    default:
      sonner.success(msg, { duration: opts.sticky ? Infinity : 4000, action: opts.action ? { label: opts.action.label, onClick: opts.action.onClick } : undefined })
  }
}

export const toastHost = { show: () => {} }

export const toastHelper = {
  success: (title: string, opts?: { description?: string; duration?: number }) =>
    sonner.success(opts?.description ? `${title} — ${opts.description}` : title, { duration: opts?.duration ?? 4000 }),
  warning: (title: string, opts?: { description?: string; duration?: number }) =>
    sonner.warning(opts?.description ? `${title} — ${opts.description}` : title, { duration: opts?.duration ?? 4000 }),
  error: (title: string, opts?: { description?: string; duration?: number }) =>
    sonner.error(opts?.description ? `${title} — ${opts.description}` : title, { duration: opts?.duration ?? 4000 }),
}

export { sonner }
