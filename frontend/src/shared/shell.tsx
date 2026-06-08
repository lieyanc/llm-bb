import type { ReactNode } from "react"
import { ThemeToggle } from "./theme-toggle"
import { cn } from "./lib/utils"
import { Card, CardAction, CardContent, CardDescription, CardHeader, CardTitle } from "./ui/card"

export function Shell({
  title,
  description,
  actions,
  children,
}: {
  title: ReactNode
  description?: ReactNode
  actions?: ReactNode
  children: ReactNode
}) {
  return (
    <div className="min-h-screen bg-background">
      <header className="sticky top-0 z-30 border-b bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/80">
        <div className="container flex min-h-16 flex-wrap items-center justify-between gap-3 py-3">
          <div className="min-w-0">
            <h1 className="truncate text-xl font-semibold tracking-tight sm:text-2xl">{title}</h1>
            {description ? (
              <p className="mt-1 max-w-2xl text-sm leading-relaxed text-muted-foreground">{description}</p>
            ) : null}
          </div>
          <div className="flex shrink-0 flex-wrap items-center justify-end gap-2">
            {actions}
            <ThemeToggle />
          </div>
        </div>
      </header>
      <main className="container space-y-5 py-5 pb-12">{children}</main>
    </div>
  )
}

export function SectionHeader({
  title,
  description,
  actions,
}: {
  title: ReactNode
  description?: ReactNode
  actions?: ReactNode
}) {
  return (
    <div className="flex flex-wrap items-center justify-between gap-3">
      <div className="min-w-0">
        <h2 className="text-base font-semibold tracking-tight">{title}</h2>
        {description ? <p className="mt-1 text-sm text-muted-foreground">{description}</p> : null}
      </div>
      {actions ? <div className="flex flex-wrap items-center gap-2">{actions}</div> : null}
    </div>
  )
}

export function Panel({
  title,
  description,
  actions,
  children,
  className,
}: {
  title?: ReactNode
  description?: ReactNode
  actions?: ReactNode
  children: ReactNode
  className?: string
}) {
  return (
    <Card className={className}>
      {title || actions ? (
        <CardHeader className="border-b p-4">
          {title ? <CardTitle>{title}</CardTitle> : <span />}
          {description ? <CardDescription>{description}</CardDescription> : null}
          {actions ? <CardAction className="flex flex-wrap items-center gap-2">{actions}</CardAction> : null}
        </CardHeader>
      ) : null}
      <CardContent className={title || actions ? "p-4" : "p-4"}>{children}</CardContent>
    </Card>
  )
}

export function MetricGrid({ children }: { children: ReactNode }) {
  return <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">{children}</div>
}

export function MetricTile({
  label,
  value,
  icon,
  helper,
}: {
  label: string
  value: ReactNode
  icon?: ReactNode
  helper?: ReactNode
}) {
  return (
    <Card className="overflow-hidden">
      <CardContent className="p-4">
        <div className="flex items-start justify-between gap-3">
          <div className="min-w-0">
            <div className="text-sm font-medium text-muted-foreground">{label}</div>
            <div className="mt-2 text-2xl font-semibold tracking-tight tabular-nums">{value}</div>
          </div>
          {icon ? (
            <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-md border bg-muted text-muted-foreground">
              {icon}
            </div>
          ) : null}
        </div>
        {helper ? <div className="mt-2 text-xs text-muted-foreground">{helper}</div> : null}
      </CardContent>
    </Card>
  )
}

export function EmptyState({
  title,
  description,
  action,
  className,
}: {
  title: string
  description?: string
  action?: ReactNode
  className?: string
}) {
  return (
    <div
      className={cn(
        "flex min-h-[220px] flex-col items-center justify-center rounded-lg border border-dashed bg-muted/20 px-6 py-10 text-center",
        className,
      )}
    >
      <h3 className="text-sm font-semibold">{title}</h3>
      {description ? <p className="mt-1 max-w-sm text-sm leading-relaxed text-muted-foreground">{description}</p> : null}
      {action ? <div className="mt-4">{action}</div> : null}
    </div>
  )
}

export function PageSection({
  title,
  description,
  actions,
  children,
}: {
  title?: ReactNode
  description?: ReactNode
  actions?: ReactNode
  children: ReactNode
}) {
  return (
    <section className="space-y-3">
      {title || description || actions ? (
        <SectionHeader title={title} description={description} actions={actions} />
      ) : null}
      {children}
    </section>
  )
}
