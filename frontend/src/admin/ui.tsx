import { LoaderCircle, Pencil, Trash2, X } from "lucide-react"
import type { ReactNode } from "react"
import { cn } from "../shared/lib/utils"
import { Button } from "../shared/ui/button"
import { Card, CardAction, CardContent, CardDescription, CardHeader, CardTitle } from "../shared/ui/card"
import { Label } from "../shared/ui/label"

export function Field({
  label,
  children,
  description,
}: {
  label: string
  children: ReactNode
  description?: ReactNode
}) {
  return (
    <div className="space-y-1.5">
      <Label>{label}</Label>
      {children}
      {description ? <p className="text-xs leading-relaxed text-muted-foreground">{description}</p> : null}
    </div>
  )
}

export function FormSection({ children }: { children: ReactNode }) {
  return <div className="space-y-4">{children}</div>
}

export function FormPanel({ children, className }: { children: ReactNode; className?: string }) {
  return <Card className={cn("h-fit xl:sticky xl:top-20", className)}>{children}</Card>
}

export function FormPanelContent({ children }: { children: ReactNode }) {
  return <CardContent className="p-4">{children}</CardContent>
}

export function StatText({ label, value }: { label: string; value: string | number }) {
  return (
    <div className="min-w-0 rounded-md border bg-muted/25 px-3 py-2">
      <div className="truncate text-xs text-muted-foreground">{label}</div>
      <div className="mt-1 truncate text-sm font-medium tabular-nums">{value}</div>
    </div>
  )
}

export function EntityHeader({
  editing,
  createLabel,
  editLabel,
  onCancel,
}: {
  editing: { id: number } | null
  createLabel: string
  editLabel?: string
  onCancel: () => void
}) {
  return (
    <div className="flex items-start justify-between gap-2">
      <div className="min-w-0">
        <h2 className="text-base font-semibold tracking-tight">
          {editing ? editLabel ?? "编辑" : createLabel}
        </h2>
        {editing ? <p className="mt-1 text-xs text-muted-foreground">#{editing.id}</p> : null}
      </div>
      {editing ? (
        <Button size="icon-sm" variant="ghost" onClick={onCancel} aria-label="取消编辑">
          <X className="h-4 w-4" />
        </Button>
      ) : null}
    </div>
  )
}

export function SubmitButton({
  busy,
  editing,
  createLabel = "创建",
  editLabel = "保存",
}: {
  busy: boolean
  editing: boolean
  createLabel?: string
  editLabel?: string
}) {
  return (
    <Button className="w-full" disabled={busy} type="submit">
      {busy ? <LoaderCircle className="h-4 w-4 animate-spin" /> : null}
      {editing ? editLabel : createLabel}
    </Button>
  )
}

export function RowActions({
  onEdit,
  onDelete,
  extra,
}: {
  onEdit?: () => void
  onDelete: () => void
  extra?: ReactNode
}) {
  return (
    <div className="flex shrink-0 flex-wrap justify-end gap-1.5">
      {extra}
      {onEdit ? (
        <Button size="icon-sm" variant="outline" onClick={onEdit} aria-label="编辑">
          <Pencil className="h-3.5 w-3.5" />
        </Button>
      ) : null}
      <Button size="icon-sm" variant="outline" onClick={onDelete} aria-label="删除">
        <Trash2 className="h-3.5 w-3.5" />
      </Button>
    </div>
  )
}

export function EntityCard({
  title,
  description,
  badges,
  actions,
  children,
  className,
}: {
  title: ReactNode
  description?: ReactNode
  badges?: ReactNode
  actions?: ReactNode
  children?: ReactNode
  className?: string
}) {
  return (
    <Card className={cn("overflow-hidden", className)}>
      <CardHeader className="p-4">
        <div className="min-w-0">
          <div className="flex min-w-0 flex-wrap items-center gap-2">
            <CardTitle className="truncate text-sm">{title}</CardTitle>
            {badges}
          </div>
          {description ? <CardDescription className="mt-1 truncate">{description}</CardDescription> : null}
        </div>
        {actions ? <CardAction>{actions}</CardAction> : null}
      </CardHeader>
      {children ? <CardContent className="p-4 pt-0">{children}</CardContent> : null}
    </Card>
  )
}

export const selectClassName =
  "flex h-9 w-full rounded-md border border-input bg-background px-3 py-1 text-sm text-foreground shadow-sm transition-colors focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring disabled:cursor-not-allowed disabled:opacity-50"
