import { type FormEvent, useState } from "react"
import { patchJSON, postJSON } from "../../shared/lib/api"
import { maskKey, relativeUnix } from "../../shared/lib/format"
import { EmptyState, PageSection } from "../../shared/shell"
import type { ProviderConfig } from "../../shared/types"
import { Badge } from "../../shared/ui/badge"
import { Input } from "../../shared/ui/input"
import { Label } from "../../shared/ui/label"
import { Switch } from "../../shared/ui/switch"
import {
  EntityHeader,
  EntityCard,
  Field,
  FormPanel,
  FormPanelContent,
  RowActions,
  StatText,
  SubmitButton,
} from "../ui"
import type { AdminActions } from "../use-admin-actions"

const emptyDraft = {
  name: "",
  base_url: "",
  api_key: "",
  default_model: "",
  timeout_ms: "20000",
  enabled: true,
}

export function ProvidersSection({
  providers,
  actions,
}: {
  providers: ProviderConfig[]
  actions: AdminActions
}) {
  const [editing, setEditing] = useState<ProviderConfig | null>(null)
  const [draft, setDraft] = useState(emptyDraft)

  function startEdit(provider: ProviderConfig) {
    setEditing(provider)
    setDraft({
      name: provider.name,
      base_url: provider.base_url,
      api_key: provider.api_key,
      default_model: provider.default_model,
      timeout_ms: String(provider.timeout_ms),
      enabled: provider.enabled,
    })
  }

  function cancelEdit() {
    setEditing(null)
    setDraft(emptyDraft)
  }

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    const payload = { ...draft, timeout_ms: Number(draft.timeout_ms) }
    const key = editing ? `edit-provider-${editing.id}` : "create-provider"
    const label = editing ? "Provider 更新" : "Provider 创建"
    await actions.runAction(key, label, () =>
      editing
        ? patchJSON(`/api/admin/providers/${editing.id}`, payload)
        : postJSON("/api/admin/providers", payload),
    )
  }

  const busy = actions.busyAction?.startsWith("edit-provider-") || actions.busyAction === "create-provider"

  return (
    <div className="grid gap-4 xl:grid-cols-[minmax(0,1fr)_360px]">
      <PageSection title="模型接入" description="配置 OpenAI-compatible Provider 和默认模型">
        <div className="grid gap-3">
          {providers.length ? (
            providers.map((provider) => (
              <EntityCard
                key={provider.id}
                title={provider.name}
                description={
                  <span className="font-mono text-xs">{provider.base_url || "未填写 Base URL"}</span>
                }
                badges={provider.enabled ? null : <Badge variant="outline">停用</Badge>}
                actions={
                  <RowActions
                    onEdit={() => startEdit(provider)}
                    onDelete={() => actions.handleDelete("providers", provider.id, provider.name)}
                  />
                }
              >
                <div className="grid gap-1.5 sm:grid-cols-2">
                  <StatText label="默认模型" value={provider.default_model || "-"} />
                  <StatText label="超时(ms)" value={provider.timeout_ms} />
                  <StatText label="API Key" value={maskKey(provider.api_key)} />
                  <StatText
                    label="更新"
                    value={relativeUnix(
                      provider.updated_at ? Math.floor(new Date(provider.updated_at).getTime() / 1000) : 0,
                    )}
                  />
                </div>
              </EntityCard>
            ))
          ) : (
            <EmptyState title="还没有 Provider" />
          )}
        </div>
      </PageSection>

      <FormPanel>
        <FormPanelContent>
          <EntityHeader createLabel="创建 Provider" editing={editing} onCancel={cancelEdit} />
          <form className="mt-4 space-y-3" onSubmit={handleSubmit}>
            <Field label="名称">
              <Input required value={draft.name} onChange={(e) => setDraft((c) => ({ ...c, name: e.target.value }))} />
            </Field>
            <Field label="Base URL">
              <Input
                placeholder="https://example.com/v1"
                value={draft.base_url}
                onChange={(e) => setDraft((c) => ({ ...c, base_url: e.target.value }))}
              />
            </Field>
            <Field label="API Key">
              <Input
                value={draft.api_key}
                onChange={(e) => setDraft((c) => ({ ...c, api_key: e.target.value }))}
              />
            </Field>
            <div className="grid grid-cols-2 gap-3">
              <Field label="默认模型">
                <Input
                  value={draft.default_model}
                  onChange={(e) => setDraft((c) => ({ ...c, default_model: e.target.value }))}
                />
              </Field>
              <Field label="超时(ms)">
                <Input
                  type="number"
                  value={draft.timeout_ms}
                  onChange={(e) => setDraft((c) => ({ ...c, timeout_ms: e.target.value }))}
                />
              </Field>
            </div>
            <div className="flex items-center justify-between rounded-md border bg-muted/25 px-3 py-2">
              <Label htmlFor="provider-enabled">启用</Label>
              <Switch
                id="provider-enabled"
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
