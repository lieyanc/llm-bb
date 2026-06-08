import { type FormEvent, useState } from "react"
import { patchJSON, postJSON } from "../../shared/lib/api"
import { EmptyState } from "../../shared/shell"
import type { Faction } from "../../shared/types"
import { Badge } from "../../shared/ui/badge"
import { Card, CardHeader } from "../../shared/ui/card"
import { Input } from "../../shared/ui/input"
import { Textarea } from "../../shared/ui/textarea"
import { EntityHeader, Field, FormPanel, FormPanelContent, RowActions, SubmitButton } from "../ui"
import type { AdminActions } from "../use-admin-actions"

const emptyDraft = {
  name: "",
  default_bias: "",
  description: "",
  shared_values: "",
  shared_style: "",
}

export function FactionsSection({
  factions,
  actions,
}: {
  factions: Faction[]
  actions: AdminActions
}) {
  const [editing, setEditing] = useState<Faction | null>(null)
  const [draft, setDraft] = useState(emptyDraft)

  function startEdit(faction: Faction) {
    setEditing(faction)
    setDraft({
      name: faction.name,
      default_bias: faction.default_bias,
      description: faction.description,
      shared_values: faction.shared_values,
      shared_style: faction.shared_style,
    })
  }

  function cancelEdit() {
    setEditing(null)
    setDraft(emptyDraft)
  }

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    const key = editing ? `edit-faction-${editing.id}` : "create-faction"
    const label = editing ? "阵营更新" : "阵营创建"
    await actions.runAction(key, label, () =>
      editing
        ? patchJSON(`/api/admin/factions/${editing.id}`, draft)
        : postJSON("/api/admin/factions", draft),
    )
  }

  const busy = actions.busyAction?.startsWith("edit-faction-") || actions.busyAction === "create-faction"

  return (
    <div className="grid gap-4 xl:grid-cols-[minmax(0,1fr)_360px]">
      <div className="grid gap-3">
        {factions.length ? (
          factions.map((faction) => (
            <Card key={faction.id}>
              <CardHeader className="flex-row items-start justify-between gap-2 space-y-0 p-4">
                <div className="min-w-0">
                  <div className="flex items-center gap-2">
                    <h3 className="text-sm font-semibold">{faction.name}</h3>
                    {faction.default_bias ? <Badge variant="secondary">{faction.default_bias}</Badge> : null}
                  </div>
                  {faction.description ? (
                    <p className="mt-1 text-sm text-foreground/85">{faction.description}</p>
                  ) : null}
                </div>
                <RowActions
                  onEdit={() => startEdit(faction)}
                  onDelete={() => actions.handleDelete("factions", faction.id, faction.name)}
                />
              </CardHeader>
            </Card>
          ))
        ) : (
          <EmptyState title="还没有阵营" />
        )}
      </div>

      <FormPanel>
        <FormPanelContent>
          <EntityHeader createLabel="创建阵营" editing={editing} onCancel={cancelEdit} />
          <form className="mt-4 space-y-3" onSubmit={handleSubmit}>
            <Field label="名称">
              <Input required value={draft.name} onChange={(e) => setDraft((c) => ({ ...c, name: e.target.value }))} />
            </Field>
            <Field label="默认偏向">
              <Input
                value={draft.default_bias}
                onChange={(e) => setDraft((c) => ({ ...c, default_bias: e.target.value }))}
              />
            </Field>
            <Field label="描述">
              <Textarea
                value={draft.description}
                onChange={(e) => setDraft((c) => ({ ...c, description: e.target.value }))}
              />
            </Field>
            <Field label="共同价值">
              <Textarea
                value={draft.shared_values}
                onChange={(e) => setDraft((c) => ({ ...c, shared_values: e.target.value }))}
              />
            </Field>
            <Field label="共同话风">
              <Textarea
                value={draft.shared_style}
                onChange={(e) => setDraft((c) => ({ ...c, shared_style: e.target.value }))}
              />
            </Field>
            <SubmitButton busy={Boolean(busy)} editing={Boolean(editing)} />
          </form>
        </FormPanelContent>
      </FormPanel>
    </div>
  )
}
