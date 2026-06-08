import { type FormEvent, useState } from "react"
import { postJSON } from "../../shared/lib/api"
import { EmptyState, PageSection } from "../../shared/shell"
import type { Persona, Relationship } from "../../shared/types"
import { Input } from "../../shared/ui/input"
import { Textarea } from "../../shared/ui/textarea"
import {
  EntityCard,
  Field,
  FormPanel,
  FormPanelContent,
  FormSection,
  RowActions,
  StatText,
  SubmitButton,
  selectClassName,
} from "../ui"
import type { AdminActions } from "../use-admin-actions"

const emptyDraft = {
  source_persona_id: "",
  target_persona_id: "",
  affinity: "0",
  hostility: "0",
  respect: "0",
  focus_weight: "0",
  notes: "",
}

export function RelationshipsSection({
  relationships,
  personas,
  actions,
}: {
  relationships: Relationship[]
  personas: Persona[]
  actions: AdminActions
}) {
  const [draft, setDraft] = useState(emptyDraft)
  const personaMap = new Map(personas.map((p) => [p.id, p.name]))

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    const payload = {
      source_persona_id: Number(draft.source_persona_id),
      target_persona_id: Number(draft.target_persona_id),
      affinity: Number(draft.affinity),
      hostility: Number(draft.hostility),
      respect: Number(draft.respect),
      focus_weight: Number(draft.focus_weight),
      notes: draft.notes,
    }
    await actions.runAction("create-rel", "关系保存", () => postJSON("/api/admin/relationships", payload))
  }

  const busy = actions.busyAction === "create-rel"

  return (
    <div className="grid gap-4 xl:grid-cols-[minmax(0,1.15fr)_360px]">
      <PageSection title="关系矩阵" description="调节角色之间的亲近、敌意、尊重和关注权重">
        <div className="grid gap-3">
          {relationships.length ? (
            relationships.map((rel) => {
              const src = personaMap.get(rel.source_persona_id) ?? `#${rel.source_persona_id}`
              const dst = personaMap.get(rel.target_persona_id) ?? `#${rel.target_persona_id}`
              return (
                <EntityCard
                  key={rel.id}
                  title={`${src} -> ${dst}`}
                  description={rel.notes}
                  actions={
                    <RowActions
                      onDelete={() => actions.handleDelete("relationships", rel.id, `${src} -> ${dst}`)}
                    />
                  }
                >
                  <div className="grid gap-1.5 sm:grid-cols-4">
                    <StatText label="亲近" value={rel.affinity} />
                    <StatText label="敌意" value={rel.hostility} />
                    <StatText label="尊重" value={rel.respect} />
                    <StatText label="权重" value={rel.focus_weight} />
                  </div>
                </EntityCard>
              )
            })
          ) : (
            <EmptyState title="还没有关系" />
          )}
        </div>
      </PageSection>

      <FormPanel>
        <FormPanelContent>
          <h2 className="text-base font-semibold">配置关系</h2>
          <form className="mt-4 space-y-3" onSubmit={handleSubmit}>
            <FormSection>
              <div className="grid grid-cols-2 gap-3">
                <Field label="来源">
                  <select
                    className={selectClassName}
                    required
                    value={draft.source_persona_id}
                    onChange={(e) => setDraft((c) => ({ ...c, source_persona_id: e.target.value }))}
                  >
                    <option value="">选择</option>
                    {personas.map((p) => (
                      <option key={p.id} value={p.id}>
                        {p.name}
                      </option>
                    ))}
                  </select>
                </Field>
                <Field label="目标">
                  <select
                    className={selectClassName}
                    required
                    value={draft.target_persona_id}
                    onChange={(e) => setDraft((c) => ({ ...c, target_persona_id: e.target.value }))}
                  >
                    <option value="">选择</option>
                    {personas.map((p) => (
                      <option key={p.id} value={p.id}>
                        {p.name}
                      </option>
                    ))}
                  </select>
                </Field>
                <Field label="亲近">
                  <Input
                    type="number"
                    value={draft.affinity}
                    onChange={(e) => setDraft((c) => ({ ...c, affinity: e.target.value }))}
                  />
                </Field>
                <Field label="敌意">
                  <Input
                    type="number"
                    value={draft.hostility}
                    onChange={(e) => setDraft((c) => ({ ...c, hostility: e.target.value }))}
                  />
                </Field>
                <Field label="尊重">
                  <Input
                    type="number"
                    value={draft.respect}
                    onChange={(e) => setDraft((c) => ({ ...c, respect: e.target.value }))}
                  />
                </Field>
                <Field label="关注权重">
                  <Input
                    type="number"
                    value={draft.focus_weight}
                    onChange={(e) => setDraft((c) => ({ ...c, focus_weight: e.target.value }))}
                  />
                </Field>
              </div>
              <Field label="备注">
                <Textarea
                  value={draft.notes}
                  onChange={(e) => setDraft((c) => ({ ...c, notes: e.target.value }))}
                />
              </Field>
            </FormSection>
            <SubmitButton busy={Boolean(busy)} editing={false} createLabel="保存" />
          </form>
        </FormPanelContent>
      </FormPanel>
    </div>
  )
}
