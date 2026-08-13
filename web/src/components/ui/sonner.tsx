"use client"

import { Toaster as Sonner, type ToasterProps } from "sonner"
import { useTheme } from "@/lib/use-theme"

function Toaster({ ...props }: ToasterProps) {
  const st = useTheme()
  const theme = st.mode === "system"
    ? window.matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light"
    : st.mode
  return (
    <Sonner
      theme={theme as "light" | "dark"}
      className="toaster group"
      style={
        {
          "--normal-bg": "var(--popover)",
          "--normal-text": "var(--popover-foreground)",
          "--normal-border": "var(--border)",
        } as React.CSSProperties
      }
      {...props}
    />
  )
}

export { Toaster }
