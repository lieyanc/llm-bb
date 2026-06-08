import { type FormEvent, useState } from "react"
import { patchJSON, postJSON } from "../../shared/lib/api"
import { EmptyState, PageSection } from "../../shared/shell"
import type { Faction, Persona, ProviderConfig } from "../../shared/types"
import { Badge } from "../../shared/ui/badge"
import { Input } from "../../shared/ui/input"
import { Label } from "../../shared/ui/label"
import { Switch } from "../../shared/ui/switch"
import { Textarea } from "../../shared/ui/textarea"
import {
  EntityHeader,
  EntityCard,
  Field,
  FormPanel,
  FormPanelContent,
  RowActions,
  StatText,
  SubmitButton,
  selectClassName,
} from "../ui"
import type { AdminActions } from "../use-admin-actions"

function makeEmptyDraft(factions: Faction[], providers: ProviderConfig[]) {
  return {
    name: "",
    public_identity: "",
    speaking_style: "",
    stance: "",
    goal: "",
    taboo: "",
    faction_id: factions[0]?.id ? String(factions[0].id) : "",
    provider_config_id: providers[0]?.id ? String(providers[0].id) : "",
    model_name: "",
    temperature: "0.9",
    max_tokens: "220",
    cooldown_seconds: "120",
    aggression: "50",
    activity_level: "50",
    enabled: true,
  }
}

export function PersonasSection({
  personas,
  factions,
  providers,
  actions,
}: {
  personas: Persona[]
  factions: Faction[]
  providers: ProviderConfig[]
  actions: AdminActions
}) {
  const [editing, setEditing] = useState<Persona | null>(null)
  const [draft, setDraft] = useState(() => makeEmptyDraft(factions, providers))

  function startEdit(persona: Persona) {
    setEditing(persona)
    setDraft({
      name: persona.name,
      public_identity: persona.public_identity,
      speaking_style: persona.speaking_style,
      stance: persona.stance,
      goal: persona.goal,
      taboo: persona.taboo,
      faction_id: persona.faction_id ? String(persona.faction_id) : "",
      provider_config_id: persona.provider_config_id ? String(persona.provider_config_id) : "",
      model_name: persona.model_name,
      temperature: String(persona.temperature),
      max_tokens: String(persona.max_tokens),
      cooldown_seconds: String(persona.cooldown_seconds),
      aggression: String(persona.aggression),
      activity_level: String(persona.activity_level),
      enabled: persona.enabled,
    })
  }

  function cancelEdit() {
    setEditing(null)
    setDraft(makeEmptyDraft(factions, providers))
  }

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    const payload = {
      ...draft,
      faction_id: Number(draft.faction_id || 0),
      provider_config_id: Number(draft.provider_config_id || 0),
      temperature: Number(draft.temperature),
      max_tokens: Number(draft.max_tokens),
      cooldown_seconds: Number(draft.cooldown_seconds),
      aggression: Number(draft.aggression),
      activity_level: Number(draft.activity_level),
    }
    const key = editing ? `edit-persona-${editing.id}` : "create-persona"
    const label = editing ? "角色更新" : "角色创建"
    await actions.runAction(key, label, () =>
      editing
        ? patchJSON(`/api/admin/personas/${editing.id}`, payload)
        : postJSON("/api/admin/personas", payload),
    )
  }

  const busy = actions.busyAction?.startsWith("edit-persona-") || actions.busyAction === "create-persona"

  return (
    <div className="grid gap-4 xl:grid-cols-[minmax(0,1.15fr)_360px]">
      <PageSection title="角色列表" description="配置角色性格、模型参数和活跃状态">
        <div className="grid gap-3 lg:grid-cols-2">
          {personas.length ? (
            personas.map((persona) => (
              <PersonaRow
                key={persona.id}
                persona={persona}
                actions={actions}
                onEdit={() => startEdit(persona)}
              />
            ))
          ) : (
            <EmptyState title="还没有角色" />
          )}
        </div>
      </PageSection>

      <FormPanel>
        <FormPanelContent>
          <EntityHeader createLabel="创建角色" editing={editing} onCancel={cancelEdit} />
          <form className="mt-4 space-y-3" onSubmit={handleSubmit}>
            <Field label="名称">
              <Input required value={draft.name} onChange={(e) => setDraft((c) => ({ ...c, name: e.target.value }))} />
            </Field>
            <Field label="公开身份">
              <Input
                value={draft.public_identity}
                onChange={(e) => setDraft((c) => ({ ...c, public_identity: e.target.value }))}
              />
            </Field>
            <Field label="说话风格">
              <Textarea
                value={draft.speaking_style}
                onChange={(e) => setDraft((c) => ({ ...c, speaking_style: e.target.value }))}
              />
            </Field>
            <Field label="立场">
              <Textarea
                value={draft.stance}
                onChange={(e) => setDraft((c) => ({ ...c, stance: e.target.value }))}
              />
            </Field>
            <Field label="目标">
              <Textarea value={draft.goal} onChange={(e) => setDraft((c) => ({ ...c, goal: e.target.value }))} />
            </Field>
            <Field label="禁区">
              <Textarea value={draft.taboo} onChange={(e) => setDraft((c) => ({ ...c, taboo: e.target.value }))} />
            </Field>

            <div className="grid grid-cols-2 gap-3">
              <Field label="阵营">
                <select
                  className={selectClassName}
                  value={draft.faction_id}
                  onChange={(e) => setDraft((c) => ({ ...c, faction_id: e.target.value }))}
                >
                  <option value="">未绑定</option>
                  {factions.map((f) => (
                    <option key={f.id} value={f.id}>
                      {f.name}
                    </option>
                  ))}
                </select>
              </Field>
              <Field label="Provider">
                <select
                  className={selectClassName}
                  value={draft.provider_config_id}
                  onChange={(e) => setDraft((c) => ({ ...c, provider_config_id: e.target.value }))}
                >
                  <option value="">未绑定</option>
                  {providers.map((p) => (
                    <option key={p.id} value={p.id}>
                      {p.name}
                    </option>
                  ))}
                </select>
              </Field>
              <Field label="模型">
                <Input
                  value={draft.model_name}
                  onChange={(e) => setDraft((c) => ({ ...c, model_name: e.target.value }))}
                />
              </Field>
              <Field label="temperature">
                <Input
                  step="0.1"
                  type="number"
                  value={draft.temperature}
                  onChange={(e) => setDraft((c) => ({ ...c, temperature: e.target.value }))}
                />
              </Field>
              <Field label="max_tokens">
                <Input
                  type="number"
                  value={draft.max_tokens}
                  onChange={(e) => setDraft((c) => ({ ...c, max_tokens: e.target.value }))}
                />
              </Field>
              <Field label="冷却(秒)">
                <Input
                  type="number"
                  value={draft.cooldown_seconds}
                  onChange={(e) => setDraft((c) => ({ ...c, cooldown_seconds: e.target.value }))}
                />
              </Field>
              <Field label="攻击性">
                <Input
                  type="number"
                  value={draft.aggression}
                  onChange={(e) => setDraft((c) => ({ ...c, aggression: e.target.value }))}
                />
              </Field>
              <Field label="活跃度">
                <Input
                  type="number"
                  value={draft.activity_level}
                  onChange={(e) => setDraft((c) => ({ ...c, activity_level: e.target.value }))}
                />
              </Field>
            </div>

            <div className="flex items-center justify-between rounded-md border bg-muted/25 px-3 py-2">
              <Label htmlFor="persona-enabled">启用</Label>
              <Switch
                id="persona-enabled"
                checked={draft.enabled}
                onCheckedChange={(v) => setDraft((c) => ({ ...c, enabled: v }))}
              />
            </div>

            <SubmitButton busy={Boolean(busy)} editing={Boolean(editing)} />
          </form>
        </FormPanelContent>
      </FormPanel>
    </div>
  )
}

function PersonaRow({
  persona,
  actions,
  onEdit,
}: {
  persona: Persona
  actions: AdminActions
  onEdit: () => void
}) {
  return (
    <EntityCard
      title={persona.name}
      description={persona.public_identity}
      badges={persona.enabled ? null : <Badge variant="outline">停用</Badge>}
      actions={
        <RowActions
          onEdit={onEdit}
          onDelete={() => actions.handleDelete("personas", persona.id, persona.name)}
        />
      }
    >
      {persona.speaking_style ? (
        <p className="mb-3 line-clamp-2 text-sm text-muted-foreground">{persona.speaking_style}</p>
      ) : null}
      <div className="grid gap-1.5 sm:grid-cols-2">
        <StatText label="temp" value={persona.temperature} />
        <StatText label="max" value={persona.max_tokens} />
        <StatText label="攻击" value={persona.aggression} />
        <StatText label="活跃" value={persona.activity_level} />
      </div>
    </EntityCard>
  )
}
