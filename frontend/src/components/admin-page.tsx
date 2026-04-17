import { LoaderCircle, LockOpen, PauseCircle, Pencil, PlayCircle, RefreshCcw, Trash2, X } from "lucide-react"
import { type FormEvent, type ReactNode, useState } from "react"
import { deleteJSON, patchJSON, postJSON } from "../lib/api"
import { maskKey, relativeUnix } from "../lib/format"
import type { AdminPageData, Faction, Persona, ProviderConfig, Relationship, RoomMemberView, RoomOverview } from "../types"
import { AdminMetrics, AppFrame, EmptyState, SectionLead, StatusBadge } from "./shared"
import { Alert, AlertDescription, AlertTitle } from "./ui/alert"
import { Badge } from "./ui/badge"
import { Button } from "./ui/button"
import { Checkbox } from "./ui/checkbox"
import { Input } from "./ui/input"
import { Label } from "./ui/label"
import { ScrollArea } from "./ui/scroll-area"
import { Switch } from "./ui/switch"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "./ui/tabs"
import { Textarea } from "./ui/textarea"

type Notice = {
  tone: "success" | "error" | "warning"
  title: string
  message: string
} | null

const selectClassName =
  "flex h-9 w-full rounded-lg border border-border bg-card px-3 py-1.5 text-sm text-foreground transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/30 focus-visible:border-primary"

export function AdminPage({ data }: { data: AdminPageData }) {
  const [notice, setNotice] = useState<Notice>(null)
  const [busyAction, setBusyAction] = useState<string | null>(null)

  const [selectedPersonas, setSelectedPersonas] = useState<number[]>([])
  const [editingRoom, setEditingRoom] = useState<RoomOverview | null>(null)
  const [roomDraft, setRoomDraft] = useState({
    name: "",
    topic: "",
    description: "",
    heat: "60",
    conflict_level: "55",
    tick_min_seconds: "25",
    tick_max_seconds: "55",
    daily_token_budget: "40000",
    summary_trigger_count: "24",
    message_retention_count: "120",
  })

  const [editingPersona, setEditingPersona] = useState<Persona | null>(null)
  const [personaDraft, setPersonaDraft] = useState({
    name: "",
    public_identity: "",
    speaking_style: "",
    stance: "",
    goal: "",
    taboo: "",
    faction_id: data.factions[0]?.id ? String(data.factions[0].id) : "",
    provider_config_id: data.providers[0]?.id ? String(data.providers[0].id) : "",
    model_name: "",
    temperature: "0.9",
    max_tokens: "220",
    cooldown_seconds: "120",
    aggression: "50",
    activity_level: "50",
    enabled: true,
  })

  const [editingFaction, setEditingFaction] = useState<Faction | null>(null)
  const [factionDraft, setFactionDraft] = useState({
    name: "",
    default_bias: "",
    description: "",
    shared_values: "",
    shared_style: "",
  })

  const [editingProvider, setEditingProvider] = useState<ProviderConfig | null>(null)
  const [providerDraft, setProviderDraft] = useState({
    name: "",
    base_url: "",
    api_key: "",
    default_model: "",
    timeout_ms: "20000",
    enabled: true,
  })

  const [relDraft, setRelDraft] = useState({
    source_persona_id: "",
    target_persona_id: "",
    affinity: "0",
    hostility: "0",
    respect: "0",
    focus_weight: "0",
    notes: "",
  })

  function startEditRoom(room: RoomOverview) {
    setEditingRoom(room)
    const members = data.roomMembers[String(room.id)] || []
    setSelectedPersonas(members.map((m) => m.persona_id))
    setRoomDraft({
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

  function cancelEditRoom() {
    setEditingRoom(null)
    setSelectedPersonas([])
    setRoomDraft({ name: "", topic: "", description: "", heat: "60", conflict_level: "55", tick_min_seconds: "25", tick_max_seconds: "55", daily_token_budget: "40000", summary_trigger_count: "24", message_retention_count: "120" })
  }

  function startEditPersona(persona: Persona) {
    setEditingPersona(persona)
    setPersonaDraft({
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

  function cancelEditPersona() {
    setEditingPersona(null)
    setPersonaDraft({ name: "", public_identity: "", speaking_style: "", stance: "", goal: "", taboo: "", faction_id: data.factions[0]?.id ? String(data.factions[0].id) : "", provider_config_id: data.providers[0]?.id ? String(data.providers[0].id) : "", model_name: "", temperature: "0.9", max_tokens: "220", cooldown_seconds: "120", aggression: "50", activity_level: "50", enabled: true })
  }

  function startEditFaction(faction: Faction) {
    setEditingFaction(faction)
    setFactionDraft({ name: faction.name, default_bias: faction.default_bias, description: faction.description, shared_values: faction.shared_values, shared_style: faction.shared_style })
  }

  function cancelEditFaction() {
    setEditingFaction(null)
    setFactionDraft({ name: "", default_bias: "", description: "", shared_values: "", shared_style: "" })
  }

  function startEditProvider(provider: ProviderConfig) {
    setEditingProvider(provider)
    setProviderDraft({ name: provider.name, base_url: provider.base_url, api_key: provider.api_key, default_model: provider.default_model, timeout_ms: String(provider.timeout_ms), enabled: provider.enabled })
  }

  function cancelEditProvider() {
    setEditingProvider(null)
    setProviderDraft({ name: "", base_url: "", api_key: "", default_model: "", timeout_ms: "20000", enabled: true })
  }

  async function handleRoomSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    const payload = {
      ...roomDraft,
      heat: Number(roomDraft.heat),
      conflict_level: Number(roomDraft.conflict_level),
      tick_min_seconds: Number(roomDraft.tick_min_seconds),
      tick_max_seconds: Number(roomDraft.tick_max_seconds),
      daily_token_budget: Number(roomDraft.daily_token_budget),
      summary_trigger_count: Number(roomDraft.summary_trigger_count),
      message_retention_count: Number(roomDraft.message_retention_count),
      persona_ids: selectedPersonas,
    }
    const action = editingRoom ? "edit-room" : "create-room"
    setBusyAction(action)
    try {
      if (editingRoom) {
        await patchJSON(`/api/admin/rooms/${editingRoom.id}`, payload)
        setNotice({ tone: "success", title: "房间已更新", message: "页面即将刷新。" })
      } else {
        await postJSON("/api/admin/rooms", payload)
        setNotice({ tone: "success", title: "房间已创建", message: "页面即将刷新。" })
      }
      window.setTimeout(() => window.location.reload(), 500)
    } catch (error) {
      setNotice({ tone: "error", title: editingRoom ? "更新房间失败" : "创建房间失败", message: error instanceof Error ? error.message : "请求失败" })
    } finally {
      setBusyAction(null)
    }
  }

  async function handlePersonaSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    const payload = {
      ...personaDraft,
      faction_id: Number(personaDraft.faction_id || 0),
      provider_config_id: Number(personaDraft.provider_config_id || 0),
      temperature: Number(personaDraft.temperature),
      max_tokens: Number(personaDraft.max_tokens),
      cooldown_seconds: Number(personaDraft.cooldown_seconds),
      aggression: Number(personaDraft.aggression),
      activity_level: Number(personaDraft.activity_level),
    }
    const action = editingPersona ? "edit-persona" : "create-persona"
    setBusyAction(action)
    try {
      if (editingPersona) {
        await patchJSON(`/api/admin/personas/${editingPersona.id}`, payload)
        setNotice({ tone: "success", title: "角色已更新", message: "页面即将刷新。" })
      } else {
        await postJSON("/api/admin/personas", payload)
        setNotice({ tone: "success", title: "角色已创建", message: "页面即将刷新。" })
      }
      window.setTimeout(() => window.location.reload(), 500)
    } catch (error) {
      setNotice({ tone: "error", title: editingPersona ? "更新角色失败" : "创建角色失败", message: error instanceof Error ? error.message : "请求失败" })
    } finally {
      setBusyAction(null)
    }
  }

  async function handleFactionSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    const action = editingFaction ? "edit-faction" : "create-faction"
    setBusyAction(action)
    try {
      if (editingFaction) {
        await patchJSON(`/api/admin/factions/${editingFaction.id}`, factionDraft)
        setNotice({ tone: "success", title: "阵营已更新", message: "页面即将刷新。" })
      } else {
        await postJSON("/api/admin/factions", factionDraft)
        setNotice({ tone: "success", title: "阵营已创建", message: "页面即将刷新。" })
      }
      window.setTimeout(() => window.location.reload(), 500)
    } catch (error) {
      setNotice({ tone: "error", title: editingFaction ? "更新阵营失败" : "创建阵营失败", message: error instanceof Error ? error.message : "请求失败" })
    } finally {
      setBusyAction(null)
    }
  }

  async function handleProviderSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    const action = editingProvider ? "edit-provider" : "create-provider"
    setBusyAction(action)
    try {
      if (editingProvider) {
        await patchJSON(`/api/admin/providers/${editingProvider.id}`, { ...providerDraft, timeout_ms: Number(providerDraft.timeout_ms) })
        setNotice({ tone: "success", title: "Provider 已更新", message: "页面即将刷新。" })
      } else {
        await postJSON("/api/admin/providers", { ...providerDraft, timeout_ms: Number(providerDraft.timeout_ms) })
        setNotice({ tone: "success", title: "Provider 已创建", message: "页面即将刷新。" })
      }
      window.setTimeout(() => window.location.reload(), 500)
    } catch (error) {
      setNotice({ tone: "error", title: editingProvider ? "更新 Provider 失败" : "创建 Provider 失败", message: error instanceof Error ? error.message : "请求失败" })
    } finally {
      setBusyAction(null)
    }
  }

  async function handleRelSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setBusyAction("create-rel")
    try {
      await postJSON("/api/admin/relationships", {
        source_persona_id: Number(relDraft.source_persona_id),
        target_persona_id: Number(relDraft.target_persona_id),
        affinity: Number(relDraft.affinity),
        hostility: Number(relDraft.hostility),
        respect: Number(relDraft.respect),
        focus_weight: Number(relDraft.focus_weight),
        notes: relDraft.notes,
      })
      setNotice({ tone: "success", title: "关系已保存", message: "页面即将刷新。" })
      window.setTimeout(() => window.location.reload(), 500)
    } catch (error) {
      setNotice({ tone: "error", title: "保存关系失败", message: error instanceof Error ? error.message : "请求失败" })
    } finally {
      setBusyAction(null)
    }
  }

  async function handleDelete(entityType: string, id: number, label: string) {
    if (!window.confirm(`确定要删除「${label}」吗？此操作不可撤销。`)) return
    setBusyAction(`delete-${entityType}-${id}`)
    try {
      await deleteJSON(`/api/admin/${entityType}/${id}`)
      setNotice({ tone: "success", title: "删除成功", message: "页面即将刷新。" })
      window.setTimeout(() => window.location.reload(), 500)
    } catch (error) {
      setNotice({ tone: "error", title: "删除失败", message: error instanceof Error ? error.message : "请求失败" })
    } finally {
      setBusyAction(null)
    }
  }

  async function runRoomAction(roomID: number, action: "tick" | "pause" | "resume") {
    setBusyAction(`${action}-${roomID}`)
    try {
      await postJSON(`/api/admin/rooms/${roomID}/${action}`)
      setNotice({ tone: "success", title: "操作已执行", message: "页面即将刷新。" })
      window.setTimeout(() => window.location.reload(), 500)
    } catch (error) {
      setNotice({ tone: "error", title: "操作失败", message: error instanceof Error ? error.message : "请求失败" })
    } finally {
      setBusyAction(null)
    }
  }

  const personaMap = new Map(data.personas.map((p) => [p.id, p.name]))

  return (
    <AppFrame
      eyebrow="Director Console"
      title="导演台"
      description="管理房间、角色、阵营、关系和 provider。"
      actions={
        <Button asChild variant="outline">
          <a href="/">返回前台</a>
        </Button>
      }
      highlights={["房间控制", "角色工作室", "阵营设定", "角色关系", "模型接入"]}
      metrics={
        <AdminMetrics
          personas={data.personas.length}
          providers={data.providers.length}
          rooms={data.rooms.length}
          runningRooms={data.runningRooms}
          totalMessages={data.totalMessages}
          totalTokens={data.totalTokens}
        />
      }
      metricsTitle="Console Snapshot"
    >
      <div className="space-y-4">
        {data.adminOpen ? (
          <Alert variant="warning">
            <AlertTitle className="flex items-center gap-2">
              <LockOpen className="h-4 w-4" />
              当前管理后台未设置口令
            </AlertTitle>
            <AlertDescription>适合本地或内网使用。生产环境请在配置文件中设置 `admin_password`。</AlertDescription>
          </Alert>
        ) : null}

        {notice ? (
          <Alert variant={notice.tone === "success" ? "success" : notice.tone === "warning" ? "warning" : "error"}>
            <AlertTitle>{notice.title}</AlertTitle>
            <AlertDescription>{notice.message}</AlertDescription>
          </Alert>
        ) : null}

        <Tabs defaultValue="rooms">
          <TabsList className="w-full justify-start">
            <TabsTrigger value="rooms">
              <TabLabel count={data.rooms.length} label="房间控制" />
            </TabsTrigger>
            <TabsTrigger value="personas">
              <TabLabel count={data.personas.length} label="角色工作室" />
            </TabsTrigger>
            <TabsTrigger value="factions">
              <TabLabel count={data.factions.length} label="阵营设定" />
            </TabsTrigger>
            <TabsTrigger value="relationships">
              <TabLabel count={data.relationships?.length ?? 0} label="角色关系" />
            </TabsTrigger>
            <TabsTrigger value="providers">
              <TabLabel count={data.providers.length} label="模型接入" />
            </TabsTrigger>
          </TabsList>

          {/* ── Rooms ── */}
          <TabsContent className="space-y-5" value="rooms">
            <SectionLead eyebrow="Rooms" title="房间控制" description="" />
            <div className="grid gap-4 xl:grid-cols-[minmax(0,1.3fr)_360px]">
              <section className="card-base p-4">
                {data.rooms.length ? (
                  <div className="divide-y divide-border">
                    {data.rooms.map((room) => (
                      <RoomControlRow
                        busyAction={busyAction}
                        key={room.id}
                        members={data.roomMembers[String(room.id)] || []}
                        room={room}
                        runRoomAction={runRoomAction}
                        onEdit={() => startEditRoom(room)}
                        onDelete={() => handleDelete("rooms", room.id, room.name)}
                      />
                    ))}
                  </div>
                ) : (
                  <EmptyState title="当前还没有房间" description="在右侧创建。" />
                )}
              </section>

              <section className="card-base p-4 xl:sticky xl:top-4 xl:self-start">
                <div className="flex items-center justify-between gap-2">
                  <div>
                    <p className="section-label">{editingRoom ? "Edit Room" : "Create Room"}</p>
                    <h2 className="mt-1 text-lg font-semibold">{editingRoom ? `编辑 #${editingRoom.id}` : "创建房间"}</h2>
                  </div>
                  {editingRoom ? (
                    <Button size="sm" variant="ghost" onClick={cancelEditRoom}>
                      <X className="h-3.5 w-3.5" /> 取消
                    </Button>
                  ) : null}
                </div>
                <p className="mt-1 text-sm text-muted-foreground">{editingRoom ? "修改后保存即生效。" : "创建后立即可用。"}</p>
                <form className="mt-4 space-y-4" onSubmit={handleRoomSubmit}>
                  <FormBlock title="基础信息">
                    <div className="grid gap-3 sm:grid-cols-2">
                      <Field label="房间名">
                        <Input required value={roomDraft.name} onChange={(e) => setRoomDraft((c) => ({ ...c, name: e.target.value }))} />
                      </Field>
                      <Field label="主题">
                        <Input value={roomDraft.topic} onChange={(e) => setRoomDraft((c) => ({ ...c, topic: e.target.value }))} />
                      </Field>
                    </div>
                    <Field label="描述">
                      <Textarea value={roomDraft.description} onChange={(e) => setRoomDraft((c) => ({ ...c, description: e.target.value }))} />
                    </Field>
                  </FormBlock>

                  <FormBlock title="节奏参数">
                    <div className="grid gap-3 sm:grid-cols-2">
                      <Field label="热度"><Input type="number" value={roomDraft.heat} onChange={(e) => setRoomDraft((c) => ({ ...c, heat: e.target.value }))} /></Field>
                      <Field label="冲突值"><Input type="number" value={roomDraft.conflict_level} onChange={(e) => setRoomDraft((c) => ({ ...c, conflict_level: e.target.value }))} /></Field>
                      <Field label="最小 Tick"><Input type="number" value={roomDraft.tick_min_seconds} onChange={(e) => setRoomDraft((c) => ({ ...c, tick_min_seconds: e.target.value }))} /></Field>
                      <Field label="最大 Tick"><Input type="number" value={roomDraft.tick_max_seconds} onChange={(e) => setRoomDraft((c) => ({ ...c, tick_max_seconds: e.target.value }))} /></Field>
                    </div>
                  </FormBlock>

                  <FormBlock title="预算与上下文">
                    <div className="grid gap-3 sm:grid-cols-2">
                      <Field label="日预算"><Input type="number" value={roomDraft.daily_token_budget} onChange={(e) => setRoomDraft((c) => ({ ...c, daily_token_budget: e.target.value }))} /></Field>
                      <Field label="摘要阈值"><Input type="number" value={roomDraft.summary_trigger_count} onChange={(e) => setRoomDraft((c) => ({ ...c, summary_trigger_count: e.target.value }))} /></Field>
                    </div>
                    <Field label="消息保留"><Input type="number" value={roomDraft.message_retention_count} onChange={(e) => setRoomDraft((c) => ({ ...c, message_retention_count: e.target.value }))} /></Field>
                  </FormBlock>

                  <FormBlock title="成员选择">
                    <div className="flex items-center justify-between gap-2">
                      <Label>已选成员</Label>
                      <Badge variant="outline">{selectedPersonas.length} 个</Badge>
                    </div>
                    <ScrollArea className="h-[220px] rounded-lg border border-border bg-card p-3">
                      <div className="space-y-2 pr-3">
                        {data.personas.map((persona) => {
                          const checked = selectedPersonas.includes(persona.id)
                          return (
                            <label className="flex cursor-pointer items-start gap-2.5 rounded-md border border-border/60 bg-secondary/30 p-2.5 transition hover:bg-secondary/60" key={persona.id}>
                              <Checkbox checked={checked} onCheckedChange={(v) => setSelectedPersonas((c) => (v ? [...c, persona.id] : c.filter((i) => i !== persona.id)))} />
                              <div className="min-w-0">
                                <div className="text-sm font-medium">#{persona.id} {persona.name}</div>
                                <div className="text-xs text-muted-foreground">{persona.public_identity || "未填写公开身份"}</div>
                              </div>
                            </label>
                          )
                        })}
                      </div>
                    </ScrollArea>
                  </FormBlock>

                  <Button className="w-full" disabled={busyAction === "create-room" || busyAction === "edit-room"} type="submit">
                    {(busyAction === "create-room" || busyAction === "edit-room") ? <LoaderCircle className="h-4 w-4 animate-spin" /> : null}
                    {editingRoom ? "保存修改" : "创建房间"}
                  </Button>
                </form>
              </section>
            </div>
          </TabsContent>

          {/* ── Personas ── */}
          <TabsContent className="space-y-5" value="personas">
            <SectionLead eyebrow="Personas" title="角色工作室" description="" />
            <div className="grid gap-4 xl:grid-cols-[minmax(0,1.15fr)_360px]">
              <div className="grid gap-3 lg:grid-cols-2">
                {data.personas.length ? (
                  data.personas.map((persona) => (
                    <PersonaRecord
                      key={persona.id}
                      persona={persona}
                      onEdit={() => startEditPersona(persona)}
                      onDelete={() => handleDelete("personas", persona.id, persona.name)}
                    />
                  ))
                ) : (
                  <EmptyState title="还没有角色" description="先创建角色，再把它们加入房间。" />
                )}
              </div>

              <section className="card-base p-4 xl:sticky xl:top-4 xl:self-start">
                <div className="flex items-center justify-between gap-2">
                  <div>
                    <p className="section-label">{editingPersona ? "Edit Persona" : "Create Persona"}</p>
                    <h2 className="mt-1 text-lg font-semibold">{editingPersona ? `编辑 #${editingPersona.id}` : "创建角色"}</h2>
                  </div>
                  {editingPersona ? (
                    <Button size="sm" variant="ghost" onClick={cancelEditPersona}><X className="h-3.5 w-3.5" /> 取消</Button>
                  ) : null}
                </div>
                <p className="mt-1 text-sm text-muted-foreground">{editingPersona ? "修改后保存即生效。" : "填写身份、动机和模型设置。"}</p>
                <form className="mt-4 space-y-4" onSubmit={handlePersonaSubmit}>
                  <FormBlock title="身份外观">
                    <div className="grid gap-3 sm:grid-cols-2">
                      <Field label="角色名"><Input required value={personaDraft.name} onChange={(e) => setPersonaDraft((c) => ({ ...c, name: e.target.value }))} /></Field>
                      <Field label="公开身份"><Input value={personaDraft.public_identity} onChange={(e) => setPersonaDraft((c) => ({ ...c, public_identity: e.target.value }))} /></Field>
                    </div>
                    <Field label="说话风格"><Textarea value={personaDraft.speaking_style} onChange={(e) => setPersonaDraft((c) => ({ ...c, speaking_style: e.target.value }))} /></Field>
                  </FormBlock>

                  <FormBlock title="动机设定">
                    <Field label="立场"><Textarea value={personaDraft.stance} onChange={(e) => setPersonaDraft((c) => ({ ...c, stance: e.target.value }))} /></Field>
                    <Field label="目标"><Textarea value={personaDraft.goal} onChange={(e) => setPersonaDraft((c) => ({ ...c, goal: e.target.value }))} /></Field>
                    <Field label="禁区"><Textarea value={personaDraft.taboo} onChange={(e) => setPersonaDraft((c) => ({ ...c, taboo: e.target.value }))} /></Field>
                  </FormBlock>

                  <FormBlock title="接入路由">
                    <div className="grid gap-3 sm:grid-cols-2">
                      <Field label="阵营">
                        <select className={selectClassName} value={personaDraft.faction_id} onChange={(e) => setPersonaDraft((c) => ({ ...c, faction_id: e.target.value }))}>
                          <option value="">未绑定</option>
                          {data.factions.map((f) => <option key={f.id} value={f.id}>{f.name}</option>)}
                        </select>
                      </Field>
                      <Field label="Provider">
                        <select className={selectClassName} value={personaDraft.provider_config_id} onChange={(e) => setPersonaDraft((c) => ({ ...c, provider_config_id: e.target.value }))}>
                          <option value="">未绑定</option>
                          {data.providers.map((p) => <option key={p.id} value={p.id}>{p.name}</option>)}
                        </select>
                      </Field>
                    </div>
                    <Field label="模型名"><Input value={personaDraft.model_name} onChange={(e) => setPersonaDraft((c) => ({ ...c, model_name: e.target.value }))} /></Field>
                  </FormBlock>

                  <FormBlock title="运行参数">
                    <div className="grid gap-3 sm:grid-cols-2">
                      <Field label="temperature"><Input step="0.1" type="number" value={personaDraft.temperature} onChange={(e) => setPersonaDraft((c) => ({ ...c, temperature: e.target.value }))} /></Field>
                      <Field label="max_tokens"><Input type="number" value={personaDraft.max_tokens} onChange={(e) => setPersonaDraft((c) => ({ ...c, max_tokens: e.target.value }))} /></Field>
                      <Field label="冷却秒数"><Input type="number" value={personaDraft.cooldown_seconds} onChange={(e) => setPersonaDraft((c) => ({ ...c, cooldown_seconds: e.target.value }))} /></Field>
                      <Field label="攻击性"><Input type="number" value={personaDraft.aggression} onChange={(e) => setPersonaDraft((c) => ({ ...c, aggression: e.target.value }))} /></Field>
                      <Field label="活跃度"><Input type="number" value={personaDraft.activity_level} onChange={(e) => setPersonaDraft((c) => ({ ...c, activity_level: e.target.value }))} /></Field>
                    </div>
                  </FormBlock>

                  <div className="flex items-center justify-between rounded-lg border border-border bg-secondary/30 px-3 py-2.5">
                    <div>
                      <Label htmlFor="persona-enabled">启用角色</Label>
                      <p className="text-xs text-muted-foreground">关闭后仍保留配置，但不会参与调度。</p>
                    </div>
                    <Switch checked={personaDraft.enabled} id="persona-enabled" onCheckedChange={(checked) => setPersonaDraft((c) => ({ ...c, enabled: checked }))} />
                  </div>

                  <Button className="w-full" disabled={busyAction === "create-persona" || busyAction === "edit-persona"} type="submit">
                    {(busyAction === "create-persona" || busyAction === "edit-persona") ? <LoaderCircle className="h-4 w-4 animate-spin" /> : null}
                    {editingPersona ? "保存修改" : "创建角色"}
                  </Button>
                </form>
              </section>
            </div>
          </TabsContent>

          {/* ── Factions ── */}
          <TabsContent className="space-y-5" value="factions">
            <SectionLead eyebrow="Factions" title="阵营设定" description="" />
            <div className="grid gap-4 xl:grid-cols-[minmax(0,1.05fr)_360px]">
              <div className="grid gap-3">
                {data.factions.length ? (
                  data.factions.map((faction) => (
                    <FactionRecord
                      faction={faction}
                      key={faction.id}
                      onEdit={() => startEditFaction(faction)}
                      onDelete={() => handleDelete("factions", faction.id, faction.name)}
                    />
                  ))
                ) : (
                  <EmptyState title="还没有阵营" description="先创建阵营，再把角色挂到不同阵营下。" />
                )}
              </div>

              <section className="card-base p-4 xl:sticky xl:top-4 xl:self-start">
                <div className="flex items-center justify-between gap-2">
                  <div>
                    <p className="section-label">{editingFaction ? "Edit Faction" : "Create Faction"}</p>
                    <h2 className="mt-1 text-lg font-semibold">{editingFaction ? `编辑 #${editingFaction.id}` : "创建阵营"}</h2>
                  </div>
                  {editingFaction ? (
                    <Button size="sm" variant="ghost" onClick={cancelEditFaction}><X className="h-3.5 w-3.5" /> 取消</Button>
                  ) : null}
                </div>
                <p className="mt-1 text-sm text-muted-foreground">{editingFaction ? "修改后保存即生效。" : "阵营名要短。"}</p>
                <form className="mt-4 space-y-4" onSubmit={handleFactionSubmit}>
                  <FormBlock title="基础信息">
                    <div className="grid gap-3 sm:grid-cols-2">
                      <Field label="阵营名"><Input required value={factionDraft.name} onChange={(e) => setFactionDraft((c) => ({ ...c, name: e.target.value }))} /></Field>
                      <Field label="默认偏向"><Input value={factionDraft.default_bias} onChange={(e) => setFactionDraft((c) => ({ ...c, default_bias: e.target.value }))} /></Field>
                    </div>
                    <Field label="阵营描述"><Textarea value={factionDraft.description} onChange={(e) => setFactionDraft((c) => ({ ...c, description: e.target.value }))} /></Field>
                  </FormBlock>

                  <FormBlock title="群体行为">
                    <Field label="共同价值"><Textarea value={factionDraft.shared_values} onChange={(e) => setFactionDraft((c) => ({ ...c, shared_values: e.target.value }))} /></Field>
                    <Field label="共同话风"><Textarea value={factionDraft.shared_style} onChange={(e) => setFactionDraft((c) => ({ ...c, shared_style: e.target.value }))} /></Field>
                  </FormBlock>

                  <Button className="w-full" disabled={busyAction === "create-faction" || busyAction === "edit-faction"} type="submit">
                    {(busyAction === "create-faction" || busyAction === "edit-faction") ? <LoaderCircle className="h-4 w-4 animate-spin" /> : null}
                    {editingFaction ? "保存修改" : "创建阵营"}
                  </Button>
                </form>
              </section>
            </div>
          </TabsContent>

          {/* ── Relationships ── */}
          <TabsContent className="space-y-5" value="relationships">
            <SectionLead eyebrow="Relationships" title="角色关系" description="配置角色间的亲近、敌意、尊重和关注权重，直接影响调度和接话概率。" />
            <div className="grid gap-4 xl:grid-cols-[minmax(0,1.15fr)_360px]">
              <div className="grid gap-3">
                {(data.relationships?.length ?? 0) > 0 ? (
                  data.relationships.map((rel) => (
                    <RelationshipRecord
                      key={rel.id}
                      rel={rel}
                      personaMap={personaMap}
                      onDelete={() => handleDelete("relationships", rel.id, `${personaMap.get(rel.source_persona_id) ?? rel.source_persona_id} -> ${personaMap.get(rel.target_persona_id) ?? rel.target_persona_id}`)}
                    />
                  ))
                ) : (
                  <EmptyState title="还没有角色关系" description="在右侧配置角色间的关系数值。" />
                )}
              </div>

              <section className="card-base p-4 xl:sticky xl:top-4 xl:self-start">
                <p className="section-label">Upsert Relationship</p>
                <h2 className="mt-1 text-lg font-semibold">配置关系</h2>
                <p className="mt-1 text-sm text-muted-foreground">相同来源-目标对会自动覆盖。</p>
                <form className="mt-4 space-y-4" onSubmit={handleRelSubmit}>
                  <FormBlock title="角色对">
                    <div className="grid gap-3 sm:grid-cols-2">
                      <Field label="来源角色">
                        <select className={selectClassName} required value={relDraft.source_persona_id} onChange={(e) => setRelDraft((c) => ({ ...c, source_persona_id: e.target.value }))}>
                          <option value="">选择角色</option>
                          {data.personas.map((p) => <option key={p.id} value={p.id}>{p.name}</option>)}
                        </select>
                      </Field>
                      <Field label="目标角色">
                        <select className={selectClassName} required value={relDraft.target_persona_id} onChange={(e) => setRelDraft((c) => ({ ...c, target_persona_id: e.target.value }))}>
                          <option value="">选择角色</option>
                          {data.personas.map((p) => <option key={p.id} value={p.id}>{p.name}</option>)}
                        </select>
                      </Field>
                    </div>
                  </FormBlock>

                  <FormBlock title="关系数值">
                    <div className="grid gap-3 sm:grid-cols-2">
                      <Field label="亲近"><Input type="number" value={relDraft.affinity} onChange={(e) => setRelDraft((c) => ({ ...c, affinity: e.target.value }))} /></Field>
                      <Field label="敌意"><Input type="number" value={relDraft.hostility} onChange={(e) => setRelDraft((c) => ({ ...c, hostility: e.target.value }))} /></Field>
                      <Field label="尊重"><Input type="number" value={relDraft.respect} onChange={(e) => setRelDraft((c) => ({ ...c, respect: e.target.value }))} /></Field>
                      <Field label="关注权重"><Input type="number" value={relDraft.focus_weight} onChange={(e) => setRelDraft((c) => ({ ...c, focus_weight: e.target.value }))} /></Field>
                    </div>
                    <Field label="备注"><Textarea value={relDraft.notes} onChange={(e) => setRelDraft((c) => ({ ...c, notes: e.target.value }))} /></Field>
                  </FormBlock>

                  <Button className="w-full" disabled={busyAction === "create-rel"} type="submit">
                    {busyAction === "create-rel" ? <LoaderCircle className="h-4 w-4 animate-spin" /> : null}
                    保存关系
                  </Button>
                </form>
              </section>
            </div>
          </TabsContent>

          {/* ── Providers ── */}
          <TabsContent className="space-y-5" value="providers">
            <SectionLead eyebrow="Providers" title="模型接入" description="OpenAI-compatible chat completions 接口。" />
            <div className="grid gap-4 xl:grid-cols-[minmax(0,1.05fr)_360px]">
              <div className="grid gap-3">
                {data.providers.length ? (
                  data.providers.map((provider) => (
                    <ProviderRecord
                      key={provider.id}
                      provider={provider}
                      onEdit={() => startEditProvider(provider)}
                      onDelete={() => handleDelete("providers", provider.id, provider.name)}
                    />
                  ))
                ) : (
                  <EmptyState title="还没有 provider" description="未配置时使用本地退化生成器。" />
                )}
              </div>

              <section className="card-base p-4 xl:sticky xl:top-4 xl:self-start">
                <div className="flex items-center justify-between gap-2">
                  <div>
                    <p className="section-label">{editingProvider ? "Edit Provider" : "Create Provider"}</p>
                    <h2 className="mt-1 text-lg font-semibold">{editingProvider ? `编辑 #${editingProvider.id}` : "创建 Provider"}</h2>
                  </div>
                  {editingProvider ? (
                    <Button size="sm" variant="ghost" onClick={cancelEditProvider}><X className="h-3.5 w-3.5" /> 取消</Button>
                  ) : null}
                </div>
                <p className="mt-1 text-sm text-muted-foreground">{editingProvider ? "修改后保存即生效。" : "例如 `https://example.com/v1`。"}</p>
                <form className="mt-4 space-y-4" onSubmit={handleProviderSubmit}>
                  <FormBlock title="连接信息">
                    <Field label="名称"><Input required value={providerDraft.name} onChange={(e) => setProviderDraft((c) => ({ ...c, name: e.target.value }))} /></Field>
                    <Field label="Base URL"><Input value={providerDraft.base_url} onChange={(e) => setProviderDraft((c) => ({ ...c, base_url: e.target.value }))} /></Field>
                    <Field label="API Key"><Input value={providerDraft.api_key} onChange={(e) => setProviderDraft((c) => ({ ...c, api_key: e.target.value }))} /></Field>
                  </FormBlock>

                  <FormBlock title="默认参数">
                    <div className="grid gap-3 sm:grid-cols-2">
                      <Field label="默认模型"><Input value={providerDraft.default_model} onChange={(e) => setProviderDraft((c) => ({ ...c, default_model: e.target.value }))} /></Field>
                      <Field label="超时毫秒"><Input type="number" value={providerDraft.timeout_ms} onChange={(e) => setProviderDraft((c) => ({ ...c, timeout_ms: e.target.value }))} /></Field>
                    </div>
                  </FormBlock>

                  <div className="flex items-center justify-between rounded-lg border border-border bg-secondary/30 px-3 py-2.5">
                    <div>
                      <Label htmlFor="provider-enabled">启用 Provider</Label>
                      <p className="text-xs text-muted-foreground">关闭后不会被角色选中。</p>
                    </div>
                    <Switch checked={providerDraft.enabled} id="provider-enabled" onCheckedChange={(checked) => setProviderDraft((c) => ({ ...c, enabled: checked }))} />
                  </div>

                  <Button className="w-full" disabled={busyAction === "create-provider" || busyAction === "edit-provider"} type="submit">
                    {(busyAction === "create-provider" || busyAction === "edit-provider") ? <LoaderCircle className="h-4 w-4 animate-spin" /> : null}
                    {editingProvider ? "保存修改" : "创建 Provider"}
                  </Button>
                </form>
              </section>
            </div>
          </TabsContent>
        </Tabs>
      </div>
    </AppFrame>
  )
}

/* ── Entity Cards ── */

function RoomControlRow({
  room,
  members,
  busyAction,
  runRoomAction,
  onEdit,
  onDelete,
}: {
  room: RoomOverview
  members: RoomMemberView[]
  busyAction: string | null
  runRoomAction: (roomID: number, action: "tick" | "pause" | "resume") => Promise<void>
  onEdit: () => void
  onDelete: () => void
}) {
  const roomActionBusy = busyAction === `pause-${room.id}` || busyAction === `resume-${room.id}` || busyAction === `tick-${room.id}`

  return (
    <article className="grid gap-4 py-4 first:pt-0 last:pb-0 lg:grid-cols-[minmax(0,1fr)_minmax(300px,0.9fr)]">
      <div className="space-y-3">
        <div className="flex items-start justify-between gap-2">
          <div>
            <p className="section-label">Room #{room.id}</p>
            <h3 className="mt-1 text-lg font-semibold">{room.name}</h3>
            <p className="mt-1 text-sm text-foreground/80">{room.topic || "未填写房间主题"}</p>
          </div>
          <StatusBadge status={room.status} />
        </div>
        <p className="text-sm text-muted-foreground">{room.description || "未填写房间描述。"}</p>
        <div className="flex flex-wrap gap-1.5">
          {members.length ? (
            members.slice(0, 6).map((member) => (
              <Badge key={member.id} variant="outline">{member.persona_name}</Badge>
            ))
          ) : (
            <Badge variant="outline">暂未配置成员</Badge>
          )}
          {members.length > 6 ? <span className="px-1 text-xs text-muted-foreground">+{members.length - 6} 个</span> : null}
        </div>
      </div>

      <div className="rounded-lg border border-border bg-secondary/30 p-3">
        <div className="grid gap-2 sm:grid-cols-2">
          <StatText label="消息" value={room.message_count} />
          <StatText label="今日 Token" value={room.tokens_today} />
          <StatText label="最近活动" value={relativeUnix(room.last_message_at_unix)} />
          <StatText label="Tick" value={`${room.tick_min_seconds}-${room.tick_max_seconds}s`} />
        </div>
        <div className="mt-3 flex flex-wrap gap-2">
          <Button asChild size="sm" variant="outline"><a href={`/rooms/${room.id}`}>查看前台</a></Button>
          <Button disabled={roomActionBusy} onClick={() => runRoomAction(room.id, "tick")} size="sm">
            {busyAction === `tick-${room.id}` ? <LoaderCircle className="h-3.5 w-3.5 animate-spin" /> : <RefreshCcw className="h-3.5 w-3.5" />}
            立即 Tick
          </Button>
          {room.status === "paused" ? (
            <Button disabled={roomActionBusy} onClick={() => runRoomAction(room.id, "resume")} size="sm" variant="ink">
              <PlayCircle className="h-3.5 w-3.5" /> 恢复运行
            </Button>
          ) : (
            <Button disabled={roomActionBusy} onClick={() => runRoomAction(room.id, "pause")} size="sm" variant="ink">
              <PauseCircle className="h-3.5 w-3.5" /> 暂停房间
            </Button>
          )}
          <Button size="sm" variant="outline" onClick={onEdit}><Pencil className="h-3.5 w-3.5" /> 编辑</Button>
          <Button size="sm" variant="outline" onClick={onDelete}><Trash2 className="h-3.5 w-3.5" /> 删除</Button>
        </div>
      </div>
    </article>
  )
}

function PersonaRecord({ persona, onEdit, onDelete }: { persona: Persona; onEdit: () => void; onDelete: () => void }) {
  return (
    <article className="card-muted p-4 transition-colors hover:border-primary/30">
      <div className="flex items-start justify-between gap-2">
        <div>
          <p className="section-label">Persona #{persona.id}</p>
          <h3 className="mt-1 text-base font-semibold">{persona.name}</h3>
          <p className="mt-0.5 text-sm text-muted-foreground">{persona.public_identity || "未填写公开身份"}</p>
        </div>
        <Badge variant={persona.enabled ? "success" : "outline"}>{persona.enabled ? "启用中" : "已停用"}</Badge>
      </div>
      <p className="mt-3 text-sm text-foreground/85">{persona.speaking_style || "未填写说话风格"}</p>
      {persona.goal ? <p className="mt-1 line-clamp-2 text-sm text-muted-foreground">目标：{persona.goal}</p> : null}
      <div className="mt-3 grid gap-1.5 sm:grid-cols-2">
        <StatText label="阵营 ID" value={persona.faction_id || "-"} />
        <StatText label="Provider ID" value={persona.provider_config_id || "-"} />
        <StatText label="temperature" value={persona.temperature} />
        <StatText label="max_tokens" value={persona.max_tokens} />
      </div>
      <div className="mt-3 flex gap-2">
        <Button size="sm" variant="outline" onClick={onEdit}><Pencil className="h-3.5 w-3.5" /> 编辑</Button>
        <Button size="sm" variant="outline" onClick={onDelete}><Trash2 className="h-3.5 w-3.5" /> 删除</Button>
      </div>
    </article>
  )
}

function FactionRecord({ faction, onEdit, onDelete }: { faction: Faction; onEdit: () => void; onDelete: () => void }) {
  return (
    <article className="card-muted p-4 transition-colors hover:border-primary/30">
      <div className="flex items-center justify-between gap-2">
        <div>
          <p className="section-label">Faction #{faction.id}</p>
          <h3 className="mt-1 text-base font-semibold">{faction.name}</h3>
        </div>
        {faction.default_bias ? <Badge variant="secondary">{faction.default_bias}</Badge> : null}
      </div>
      <p className="mt-3 text-sm text-foreground/85">{faction.description || "未填写阵营描述。"}</p>
      <div className="mt-3 grid gap-1.5 sm:grid-cols-2">
        <StatText label="共同价值" value={faction.shared_values || "-"} />
        <StatText label="共同话风" value={faction.shared_style || "-"} />
      </div>
      <div className="mt-3 flex gap-2">
        <Button size="sm" variant="outline" onClick={onEdit}><Pencil className="h-3.5 w-3.5" /> 编辑</Button>
        <Button size="sm" variant="outline" onClick={onDelete}><Trash2 className="h-3.5 w-3.5" /> 删除</Button>
      </div>
    </article>
  )
}

function ProviderRecord({ provider, onEdit, onDelete }: { provider: ProviderConfig; onEdit: () => void; onDelete: () => void }) {
  return (
    <article className="card-muted p-4 transition-colors hover:border-primary/30">
      <div className="flex items-center justify-between gap-2">
        <div>
          <p className="section-label">Provider #{provider.id}</p>
          <h3 className="mt-1 text-base font-semibold">{provider.name}</h3>
        </div>
        <Badge variant={provider.enabled ? "success" : "outline"}>{provider.enabled ? "已启用" : "已停用"}</Badge>
      </div>
      <p className="mt-3 text-sm text-foreground/85">{provider.base_url || "未填写 Base URL"}</p>
      <div className="mt-3 grid gap-1.5 sm:grid-cols-2">
        <StatText label="默认模型" value={provider.default_model || "-"} />
        <StatText label="超时毫秒" value={provider.timeout_ms} />
        <StatText label="API Key" value={maskKey(provider.api_key)} />
        <StatText label="最后更新" value={relativeUnix(provider.updated_at ? Math.floor(new Date(provider.updated_at).getTime() / 1000) : 0)} />
      </div>
      <div className="mt-3 flex gap-2">
        <Button size="sm" variant="outline" onClick={onEdit}><Pencil className="h-3.5 w-3.5" /> 编辑</Button>
        <Button size="sm" variant="outline" onClick={onDelete}><Trash2 className="h-3.5 w-3.5" /> 删除</Button>
      </div>
    </article>
  )
}

function RelationshipRecord({ rel, personaMap, onDelete }: { rel: Relationship; personaMap: Map<number, string>; onDelete: () => void }) {
  const sourceName = personaMap.get(rel.source_persona_id) ?? `#${rel.source_persona_id}`
  const targetName = personaMap.get(rel.target_persona_id) ?? `#${rel.target_persona_id}`

  return (
    <article className="card-muted p-4 transition-colors hover:border-primary/30">
      <div className="flex items-center justify-between gap-2">
        <div>
          <p className="section-label">Relationship #{rel.id}</p>
          <h3 className="mt-1 text-base font-semibold">{sourceName} → {targetName}</h3>
        </div>
        <Button size="sm" variant="outline" onClick={onDelete}><Trash2 className="h-3.5 w-3.5" /></Button>
      </div>
      {rel.notes ? <p className="mt-2 text-sm text-muted-foreground">{rel.notes}</p> : null}
      <div className="mt-3 grid gap-1.5 sm:grid-cols-2">
        <StatText label="亲近" value={rel.affinity} />
        <StatText label="敌意" value={rel.hostility} />
        <StatText label="尊重" value={rel.respect} />
        <StatText label="关注权重" value={rel.focus_weight} />
      </div>
    </article>
  )
}

/* ── Shared Form Components ── */

function FormBlock({ title, children }: { title: string; children: ReactNode }) {
  return (
    <section className="card-muted space-y-3 p-3">
      <h3 className="text-sm font-semibold">{title}</h3>
      <div className="space-y-3">{children}</div>
    </section>
  )
}

function Field({ label, children }: { label: string; children: ReactNode }) {
  return (
    <div className="space-y-1.5">
      <Label>{label}</Label>
      {children}
    </div>
  )
}

function StatText({ label, value }: { label: string; value: string | number }) {
  return (
    <div className="flex items-center justify-between rounded-md border border-border/60 bg-card px-2.5 py-2">
      <span className="text-xs text-muted-foreground">{label}</span>
      <span className="text-sm font-medium">{value}</span>
    </div>
  )
}

function TabLabel({ label, count }: { label: string; count: number }) {
  return (
    <span className="inline-flex items-center gap-1.5">
      <span>{label}</span>
      <span className="rounded bg-secondary px-1.5 py-0.5 text-[10px] text-muted-foreground">{count}</span>
    </span>
  )
}
