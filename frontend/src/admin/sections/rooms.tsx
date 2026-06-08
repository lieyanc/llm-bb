import { Eye, LoaderCircle, PauseCircle, PlayCircle, RefreshCcw } from "lucide-react"
import { type FormEvent, useState } from "react"
import { patchJSON, postJSON } from "../../shared/lib/api"
import { relativeUnix, statusLabel, statusTone } from "../../shared/lib/format"
import { EmptyState, PageSection } from "../../shared/shell"
import type { AdminDefaults, Persona, RoomMemberView, RoomOverview } from "../../shared/types"
import { Badge } from "../../shared/ui/badge"
import { Button } from "../../shared/ui/button"
import { Checkbox } from "../../shared/ui/checkbox"
import { Input } from "../../shared/ui/input"
import { ScrollArea } from "../../shared/ui/scroll-area"
import { Textarea } from "../../shared/ui/textarea"
import {
  EntityHeader,
  EntityCard,
  Field,
  FormPanel,
  FormPanelContent,
  FormSection,
  RowActions,
  StatText,
  SubmitButton,
} from "../ui"
import type { AdminActions } from "../use-admin-actions"

function makeEmptyDraft(defaults: AdminDefaults["room"]) {
  return {
    name: "",
    topic: "",
    description: "",
    heat: String(defaults.heat),
    conflict_level: String(defaults.conflict_level),
    tick_min_seconds: String(defaults.tick_min_seconds),
    tick_max_seconds: String(defaults.tick_max_seconds),
    daily_token_budget: String(defaults.daily_token_budget),
    summary_trigger_count: String(defaults.summary_trigger_count),
    message_retention_count: String(defaults.message_retention_count),
  }
}

export function RoomsSection({
  rooms,
  personas,
  roomMembers,
  defaults,
  actions,
}: {
  rooms: RoomOverview[]
  personas: Persona[]
  roomMembers: Record<string, RoomMemberView[]>
  defaults: AdminDefaults["room"]
  actions: AdminActions
}) {
  const [editing, setEditing] = useState<RoomOverview | null>(null)
  const [draft, setDraft] = useState(() => makeEmptyDraft(defaults))
  const [selected, setSelected] = useState<number[]>([])

  function startEdit(room: RoomOverview) {
    setEditing(room)
    setSelected((roomMembers[String(room.id)] || []).map((m) => m.persona_id))
    setDraft({
      name: room.name,
      topic: room.topic,
      description: room.description,
      heat: String(room.heat),
      conflict_level: String(room.conflict_level),
      tick_min_seconds: String(room.tick_min_seconds),
      tick_max_seconds: String(room.tick_max_seconds),
      daily_token_budget: String(room.daily_token_budget),
      summary_trigger_count: String(room.summary_trigger_count),
      message_retention_count: String(room.message_retention_count),
    })
  }

  function cancelEdit() {
    setEditing(null)
    setDraft(makeEmptyDraft(defaults))
    setSelected([])
  }

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    const payload = {
      ...draft,
      heat: Number(draft.heat),
      conflict_level: Number(draft.conflict_level),
      tick_min_seconds: Number(draft.tick_min_seconds),
      tick_max_seconds: Number(draft.tick_max_seconds),
      daily_token_budget: Number(draft.daily_token_budget),
      summary_trigger_count: Number(draft.summary_trigger_count),
      message_retention_count: Number(draft.message_retention_count),
      persona_ids: selected,
    }
    const key = editing ? `edit-room-${editing.id}` : "create-room"
    const label = editing ? "房间更新" : "房间创建"
    await actions.runAction(key, label, () =>
      editing ? patchJSON(`/api/admin/rooms/${editing.id}`, payload) : postJSON("/api/admin/rooms", payload),
    )
  }

  const busy = actions.busyAction?.startsWith("edit-room-") || actions.busyAction === "create-room"

  return (
    <div className="grid gap-4 xl:grid-cols-[minmax(0,1.25fr)_360px]">
      <PageSection title="房间列表" description="查看运行状态、手动 tick 或暂停调度">
        <div className="space-y-3">
          {rooms.length ? (
            rooms.map((room) => (
              <RoomRow
                key={room.id}
                room={room}
                members={roomMembers[String(room.id)] || []}
                actions={actions}
                onEdit={() => startEdit(room)}
              />
            ))
          ) : (
            <EmptyState title="还没有房间" />
          )}
        </div>
      </PageSection>

      <FormPanel>
        <FormPanelContent>
          <EntityHeader createLabel="创建房间" editing={editing} onCancel={cancelEdit} />
          <form className="mt-4 space-y-3" onSubmit={handleSubmit}>
            <FormSection>
              <Field label="名称">
                <Input required value={draft.name} onChange={(e) => setDraft((c) => ({ ...c, name: e.target.value }))} />
              </Field>
              <Field label="主题">
                <Input value={draft.topic} onChange={(e) => setDraft((c) => ({ ...c, topic: e.target.value }))} />
              </Field>
              <Field label="描述">
                <Textarea
                  value={draft.description}
                  onChange={(e) => setDraft((c) => ({ ...c, description: e.target.value }))}
                />
              </Field>
            </FormSection>

            <div className="grid grid-cols-2 gap-3">
              <Field label="热度">
                <Input
                  type="number"
                  value={draft.heat}
                  onChange={(e) => setDraft((c) => ({ ...c, heat: e.target.value }))}
                />
              </Field>
              <Field label="冲突值">
                <Input
                  type="number"
                  value={draft.conflict_level}
                  onChange={(e) => setDraft((c) => ({ ...c, conflict_level: e.target.value }))}
                />
              </Field>
              <Field label="Tick 最小">
                <Input
                  type="number"
                  value={draft.tick_min_seconds}
                  onChange={(e) => setDraft((c) => ({ ...c, tick_min_seconds: e.target.value }))}
                />
              </Field>
              <Field label="Tick 最大">
                <Input
                  type="number"
                  value={draft.tick_max_seconds}
                  onChange={(e) => setDraft((c) => ({ ...c, tick_max_seconds: e.target.value }))}
                />
              </Field>
              <Field label="日预算">
                <Input
                  type="number"
                  value={draft.daily_token_budget}
                  onChange={(e) => setDraft((c) => ({ ...c, daily_token_budget: e.target.value }))}
                />
              </Field>
              <Field label="摘要阈值">
                <Input
                  type="number"
                  value={draft.summary_trigger_count}
                  onChange={(e) => setDraft((c) => ({ ...c, summary_trigger_count: e.target.value }))}
                />
              </Field>
            </div>

            <Field label="消息保留">
              <Input
                type="number"
                value={draft.message_retention_count}
                onChange={(e) => setDraft((c) => ({ ...c, message_retention_count: e.target.value }))}
              />
            </Field>

            <div className="space-y-2">
              <div className="flex items-center justify-between">
                <span className="text-sm font-medium">成员</span>
                <Badge variant="outline">{selected.length}</Badge>
              </div>
              <ScrollArea className="h-[220px] rounded-md border bg-muted/20 p-2">
                <div className="space-y-1 pr-3">
                  {personas.map((persona) => {
                    const checked = selected.includes(persona.id)
                    return (
                      <label
                        key={persona.id}
                        className="flex cursor-pointer items-center gap-2 rounded-md px-2 py-1.5 hover:bg-accent"
                      >
                        <Checkbox
                          checked={checked}
                          onCheckedChange={(v) =>
                            setSelected((c) => (v ? [...c, persona.id] : c.filter((i) => i !== persona.id)))
                          }
                        />
                        <span className="truncate text-sm">{persona.name}</span>
                      </label>
                    )
                  })}
                </div>
              </ScrollArea>
            </div>

            <SubmitButton busy={Boolean(busy)} editing={Boolean(editing)} />
          </form>
        </FormPanelContent>
      </FormPanel>
    </div>
  )
}

function RoomRow({
  room,
  members,
  actions,
  onEdit,
}: {
  room: RoomOverview
  members: RoomMemberView[]
  actions: AdminActions
  onEdit: () => void
}) {
  const runCtl = (act: "tick" | "pause" | "resume") =>
    actions.runAction(`${act}-${room.id}`, act === "tick" ? "Tick" : act === "pause" ? "暂停" : "恢复", () =>
      postJSON(`/api/admin/rooms/${room.id}/${act}`),
    )
  const busyCtl =
    actions.busyAction === `tick-${room.id}` ||
    actions.busyAction === `pause-${room.id}` ||
    actions.busyAction === `resume-${room.id}`

  return (
    <EntityCard
      title={room.name}
      description={room.topic}
      badges={<Badge variant={statusTone(room.status)}>{statusLabel(room.status)}</Badge>}
      actions={
        <RowActions
          onEdit={onEdit}
          onDelete={() => actions.handleDelete("rooms", room.id, room.name)}
          extra={
            <>
              <Button size="icon-sm" variant="outline" asChild aria-label="查看">
                <a href={`/rooms/${room.id}`}>
                  <Eye className="h-3.5 w-3.5" />
                </a>
              </Button>
              <Button size="icon-sm" disabled={busyCtl} onClick={() => runCtl("tick")} aria-label="Tick">
                {actions.busyAction === `tick-${room.id}` ? (
                  <LoaderCircle className="h-3.5 w-3.5 animate-spin" />
                ) : (
                  <RefreshCcw className="h-3.5 w-3.5" />
                )}
              </Button>
              {room.status === "paused" ? (
                <Button size="icon-sm" variant="ink" disabled={busyCtl} onClick={() => runCtl("resume")} aria-label="恢复">
                  <PlayCircle className="h-3.5 w-3.5" />
                </Button>
              ) : (
                <Button size="icon-sm" variant="ink" disabled={busyCtl} onClick={() => runCtl("pause")} aria-label="暂停">
                  <PauseCircle className="h-3.5 w-3.5" />
                </Button>
              )}
            </>
          }
        />
      }
    >
      <div className="grid gap-1.5 sm:grid-cols-4">
        <StatText label="消息" value={room.message_count} />
        <StatText label="Token" value={room.tokens_today} />
        <StatText label="Tick" value={`${room.tick_min_seconds}-${room.tick_max_seconds}s`} />
        <StatText label="活动" value={relativeUnix(room.last_message_at_unix)} />
      </div>

      {members.length ? (
        <div className="mt-3 flex flex-wrap gap-1">
          {members.slice(0, 8).map((m) => (
            <Badge key={m.id} variant="outline">
              {m.persona_name}
            </Badge>
          ))}
          {members.length > 8 ? (
            <span className="px-1 text-xs text-muted-foreground">+{members.length - 8}</span>
          ) : null}
        </div>
      ) : null}
    </EntityCard>
  )
}
