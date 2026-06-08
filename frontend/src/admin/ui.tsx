import { LoaderCircle, Pencil, Trash2, X } from "lucide-react"
import type { ReactNode } from "react"
import { Button } from "../shared/ui/button"
import { Label } from "../shared/ui/label"

export function Field({ label, children }: { label: string; children: ReactNode }) {
  return (
    <div className="space-y-1.5">
      <Label>{label}</Label>
      {children}
    </div>
  )
}

export function FormSection({ children }: { children: ReactNode }) {
  return <div className="space-y-3">{children}</div>
}

export function StatText({ label, value }: { label: string; value: string | number }) {
  return (
    <div className="flex items-center justify-between rounded-md border border-border/60 bg-card px-2.5 py-1.5">
      <span className="text-xs text-muted-foreground">{label}</span>
      <span className="text-sm font-medium tabular-nums">{value}</span>
    </div>
  )
}

export function EntityHeader({
  title,
  editing,
  createLabel,
  editLabel,
  onCancel,
}: {
  title: string
  editing: { id: number } | null
  createLabel: string
  editLabel?: string
  onCancel: () => void
}) {
  return (
    <div className="flex items-center justify-between gap-2">
      <h2 className="text-base font-semibold">
        {editing ? `${editLabel ?? "编辑"} #${editing.id}` : createLabel}
      </h2>
      {editing ? (
        <Button size="sm" variant="ghost" onClick={onCancel}>
          <X className="h-3.5 w-3.5" />
        </Button>
      ) : null}
      {!editing ? <span className="text-xs text-muted-foreground">{title}</span> : null}
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
    <div className="flex flex-wrap gap-1.5">
      {extra}
      {onEdit ? (
        <Button size="sm" variant="outline" onClick={onEdit}>
          <Pencil className="h-3.5 w-3.5" />
        </Button>
      ) : null}
      <Button size="sm" variant="outline" onClick={onDelete}>
        <Trash2 className="h-3.5 w-3.5" />
      </Button>
    </div>
  )
}

export const selectClassName = "form-select"
