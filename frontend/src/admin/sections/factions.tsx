import { type FormEvent, useState } from "react"
import { patchJSON, postJSON } from "../../shared/lib/api"
import { EmptyState, PageSection } from "../../shared/shell"
import type { Faction } from "../../shared/types"
import { Badge } from "../../shared/ui/badge"
import { Input } from "../../shared/ui/input"
import { Textarea } from "../../shared/ui/textarea"
import { EntityCard, EntityHeader, Field, FormPanel, FormPanelContent, RowActions, SubmitButton } from "../ui"
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
      <PageSection title="阵营列表" description="沉淀共享价值、话风和默认偏向">
        <div className="grid gap-3">
          {factions.length ? (
            factions.map((faction) => (
              <EntityCard
                key={faction.id}
                title={faction.name}
                description={faction.description}
                badges={faction.default_bias ? <Badge variant="secondary">{faction.default_bias}</Badge> : null}
                actions={
                  <RowActions
                    onEdit={() => startEdit(faction)}
                    onDelete={() => actions.handleDelete("factions", faction.id, faction.name)}
                  />
                }
              >
                <div className="grid gap-2 text-sm text-muted-foreground md:grid-cols-2">
                  {faction.shared_values ? (
                    <p className="line-clamp-2 leading-relaxed">{faction.shared_values}</p>
                  ) : null}
                  {faction.shared_style ? (
                    <p className="line-clamp-2 leading-relaxed">{faction.shared_style}</p>
                  ) : null}
                </div>
              </EntityCard>
            ))
          ) : (
            <EmptyState title="还没有阵营" />
          )}
        </div>
      </PageSection>

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
