import { LoaderCircle, Pencil, Trash2, X } from "lucide-react"
import type { ReactNode } from "react"
import { Button } from "../shared/ui/button"
import { Card, CardContent } from "../shared/ui/card"
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

export function FormPanel({ children }: { children: ReactNode }) {
  return <Card className="h-fit xl:sticky xl:top-4">{children}</Card>
}

export function FormPanelContent({ children }: { children: ReactNode }) {
  return <CardContent className="p-4">{children}</CardContent>
}

export function StatText({ label, value }: { label: string; value: string | number }) {
  return (
    <div className="flex items-center justify-between rounded-md border bg-muted/30 px-2.5 py-1.5">
      <span className="text-xs text-muted-foreground">{label}</span>
      <span className="text-sm font-medium tabular-nums">{value}</span>
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
    <div className="flex items-center justify-between gap-2">
      <h2 className="text-base font-semibold">
        {editing ? `${editLabel ?? "编辑"} #${editing.id}` : createLabel}
      </h2>
      {editing ? (
        <Button size="icon" variant="ghost" onClick={onCancel} aria-label="取消编辑">
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
    <div className="flex flex-wrap gap-1.5">
      {extra}
      {onEdit ? (
        <Button size="icon" variant="outline" onClick={onEdit} aria-label="编辑">
          <Pencil className="h-3.5 w-3.5" />
        </Button>
      ) : null}
      <Button size="icon" variant="outline" onClick={onDelete} aria-label="删除">
        <Trash2 className="h-3.5 w-3.5" />
      </Button>
    </div>
  )
}

export const selectClassName = "form-select"
