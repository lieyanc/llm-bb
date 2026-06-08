import type { ReactNode } from "react"
import { ThemeToggle } from "./theme-toggle"
import { Card, CardContent } from "./ui/card"

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
    <div className="min-h-screen bg-muted/25">
      <div className="container py-6 pb-12">
        <header className="mb-6 flex flex-wrap items-center justify-between gap-3">
          <h1 className="text-xl font-semibold tracking-tight sm:text-2xl">{title}</h1>
          <div className="flex flex-wrap items-center gap-2">
            {actions}
            <ThemeToggle />
          </div>
        </header>
        <main className="space-y-5">{children}</main>
      </div>
    </div>
  )
}

export function SectionHeader({
  title,
  actions,
}: {
  title: ReactNode
  actions?: ReactNode
}) {
  return (
    <div className="flex flex-wrap items-center justify-between gap-3">
      <h2 className="text-base font-semibold tracking-tight">{title}</h2>
      {actions ? <div className="flex flex-wrap items-center gap-2">{actions}</div> : null}
    </div>
  )
}

export function Panel({
  title,
  actions,
  children,
  className,
}: {
  title?: ReactNode
  actions?: ReactNode
  children: ReactNode
  className?: string
}) {
  return (
    <Card className={className}>
      {title || actions ? (
        <div className="flex flex-wrap items-center justify-between gap-3 border-b px-5 py-4">
          {title ? <h2 className="text-base font-semibold tracking-tight">{title}</h2> : <span />}
          {actions ? <div className="flex flex-wrap items-center gap-2">{actions}</div> : null}
        </div>
      ) : null}
      <CardContent className={title || actions ? "pt-5" : "p-5"}>{children}</CardContent>
    </Card>
  )
}

export function MetricGrid({ children }: { children: ReactNode }) {
  return <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">{children}</div>
}

export function MetricTile({ label, value }: { label: string; value: ReactNode }) {
  return (
    <Card>
      <CardContent className="p-4">
        <div className="text-sm font-medium text-muted-foreground">{label}</div>
        <div className="mt-2 text-2xl font-semibold tracking-tight tabular-nums">{value}</div>
      </CardContent>
    </Card>
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
    <div className="flex min-h-[180px] flex-col items-center justify-center rounded-lg border border-dashed bg-background px-6 py-10 text-center">
      <h3 className="text-sm font-semibold">{title}</h3>
      {description ? <p className="mt-1 text-sm text-muted-foreground">{description}</p> : null}
      {action ? <div className="mt-4">{action}</div> : null}
    </div>
  )
}
