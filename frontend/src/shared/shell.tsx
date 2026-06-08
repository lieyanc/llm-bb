import type { ReactNode } from "react"
import { ThemeToggle } from "./theme-toggle"

export function Shell({
  title,
  actions,
  children,
}: {
  title: ReactNode
  actions?: ReactNode
  children: ReactNode
}) {
  return (
    <div className="container py-6 pb-12">
      <header className="mb-6 flex flex-wrap items-center justify-between gap-3 border-b border-border pb-4">
        <div className="flex items-center gap-3">
          <span className="h-2 w-2 rounded-full bg-success" />
          <h1 className="text-lg font-semibold tracking-tight sm:text-xl">{title}</h1>
        </div>
        <div className="flex flex-wrap items-center gap-2">
          {actions}
          <ThemeToggle />
        </div>
      </header>
      <main className="space-y-6">{children}</main>
    </div>
  )
}

export function MetricGrid({ children }: { children: ReactNode }) {
  return <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">{children}</div>
}

export function MetricTile({ label, value }: { label: string; value: ReactNode }) {
  return (
    <div className="rounded-lg border border-border bg-card px-4 py-3">
      <div className="text-xs text-muted-foreground">{label}</div>
      <div className="mt-1 text-xl font-semibold tabular-nums">{value}</div>
    </div>
  )
}

export function EmptyState({
  title,
  description,
  action,
}: {
  title: string
  description?: string
  action?: ReactNode
}) {
  return (
    <div className="rounded-lg border border-dashed border-border bg-card px-5 py-8 text-center">
      <h3 className="text-sm font-semibold">{title}</h3>
      {description ? <p className="mt-1 text-sm text-muted-foreground">{description}</p> : null}
      {action ? <div className="mt-3">{action}</div> : null}
    </div>
  )
}
