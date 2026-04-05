import { LoaderCircle, LockOpen, PauseCircle, PlayCircle, RefreshCcw } from "lucide-react"
import { type FormEvent, type ReactNode, useState } from "react"
import { postJSON } from "../lib/api"
import { maskKey, relativeUnix } from "../lib/format"
import type { AdminPageData, Faction, Persona, ProviderConfig, RoomMemberView, RoomOverview } from "../types"
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
  "flex h-11 w-full rounded-2xl border border-border/75 bg-background/78 px-4 py-2 text-sm text-foreground shadow-sm transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-background"

export function AdminPage({ data }: { data: AdminPageData }) {
  const [notice, setNotice] = useState<Notice>(null)
  const [busyAction, setBusyAction] = useState<string | null>(null)

  const [selectedPersonas, setSelectedPersonas] = useState<number[]>([])
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

  const [factionDraft, setFactionDraft] = useState({
    name: "",
    default_bias: "",
    description: "",
    shared_values: "",
    shared_style: "",
  })

  const [providerDraft, setProviderDraft] = useState({
    name: "",
    base_url: "",
    api_key: "",
    default_model: "",
    timeout_ms: "20000",
    enabled: true,
  })

  async function handleRoomSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setBusyAction("create-room")
    try {
      await postJSON("/api/admin/rooms", {
        ...roomDraft,
        heat: Number(roomDraft.heat),
        conflict_level: Number(roomDraft.conflict_level),
        tick_min_seconds: Number(roomDraft.tick_min_seconds),
        tick_max_seconds: Number(roomDraft.tick_max_seconds),
        daily_token_budget: Number(roomDraft.daily_token_budget),
        summary_trigger_count: Number(roomDraft.summary_trigger_count),
        message_retention_count: Number(roomDraft.message_retention_count),
        persona_ids: selectedPersonas,
      })
      setNotice({ tone: "success", title: "房间已创建", message: "控制台即将刷新，最新配置会重新从后端拉取。" })
      window.setTimeout(() => window.location.reload(), 500)
    } catch (error) {
      setNotice({ tone: "error", title: "创建房间失败", message: error instanceof Error ? error.message : "请求失败" })
    } finally {
      setBusyAction(null)
    }
  }

  async function handlePersonaSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setBusyAction("create-persona")
    try {
      await postJSON("/api/admin/personas", {
        ...personaDraft,
        faction_id: Number(personaDraft.faction_id || 0),
        provider_config_id: Number(personaDraft.provider_config_id || 0),
        temperature: Number(personaDraft.temperature),
        max_tokens: Number(personaDraft.max_tokens),
        cooldown_seconds: Number(personaDraft.cooldown_seconds),
        aggression: Number(personaDraft.aggression),
        activity_level: Number(personaDraft.activity_level),
      })
      setNotice({ tone: "success", title: "角色已创建", message: "页面会自动刷新，把新的角色加入导演台列表。" })
      window.setTimeout(() => window.location.reload(), 500)
    } catch (error) {
      setNotice({ tone: "error", title: "创建角色失败", message: error instanceof Error ? error.message : "请求失败" })
    } finally {
      setBusyAction(null)
    }
  }

  async function handleFactionSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setBusyAction("create-faction")
    try {
      await postJSON("/api/admin/factions", factionDraft)
      setNotice({ tone: "success", title: "阵营已创建", message: "刷新后即可在角色表单中选中这个阵营。" })
      window.setTimeout(() => window.location.reload(), 500)
    } catch (error) {
      setNotice({ tone: "error", title: "创建阵营失败", message: error instanceof Error ? error.message : "请求失败" })
    } finally {
      setBusyAction(null)
    }
  }

  async function handleProviderSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setBusyAction("create-provider")
    try {
      await postJSON("/api/admin/providers", {
        ...providerDraft,
        timeout_ms: Number(providerDraft.timeout_ms),
      })
      setNotice({ tone: "success", title: "Provider 已创建", message: "刷新后即可在角色表单中直接绑定它。" })
      window.setTimeout(() => window.location.reload(), 500)
    } catch (error) {
      setNotice({ tone: "error", title: "创建 Provider 失败", message: error instanceof Error ? error.message : "请求失败" })
    } finally {
      setBusyAction(null)
    }
  }

  async function runRoomAction(roomID: number, action: "tick" | "pause" | "resume") {
    setBusyAction(`${action}-${roomID}`)
    try {
      await postJSON(`/api/admin/rooms/${roomID}/${action}`)
      setNotice({ tone: "success", title: "操作已执行", message: "页面即将刷新，展示当前最新状态。" })
      window.setTimeout(() => window.location.reload(), 500)
    } catch (error) {
      setNotice({ tone: "error", title: "操作失败", message: error instanceof Error ? error.message : "请求失败" })
    } finally {
      setBusyAction(null)
    }
  }

  return (
    <AppFrame
      eyebrow="Director Console"
      title="导演台"
      description="这里负责房间节奏、角色阵容、阵营关系和 provider 配置。列表优先展示运行状态，右侧持续保留创建入口。"
      actions={
        <Button asChild size="lg" variant="outline">
          <a href="/">返回前台</a>
        </Button>
      }
      highlights={["房间控制", "角色工作室", "阵营设定", "模型接入"]}
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
      <div className="space-y-5">
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
            <TabsTrigger value="providers">
              <TabLabel count={data.providers.length} label="模型接入" />
            </TabsTrigger>
          </TabsList>

          <TabsContent className="space-y-6" value="rooms">
            <SectionLead eyebrow="Rooms" title="房间控制" description="左边看正在运行的房间和即时操作，右边直接创建新房间并分配成员。" />
            <div className="grid gap-6 xl:grid-cols-[minmax(0,1.3fr)_390px]">
              <section className="panel-surface px-5 py-5">
                {data.rooms.length ? (
                  <div className="divide-y divide-border/60">
                    {data.rooms.map((room) => (
                      <RoomControlRow
                        busyAction={busyAction}
                        key={room.id}
                        members={data.roomMembers[String(room.id)] || []}
                        room={room}
                        runRoomAction={runRoomAction}
                      />
                    ))}
                  </div>
                ) : (
                  <EmptyState title="当前还没有房间" description="先在右侧创建房间，再把角色加入成员列表。" />
                )}
              </section>

              <section className="panel-surface px-5 py-5 xl:sticky xl:top-6 xl:self-start">
                <p className="tiny-label">Create Room</p>
                <h2 className="mt-2 font-display text-2xl font-semibold tracking-[-0.05em]">创建房间</h2>
                <p className="mt-2 text-sm leading-7 text-muted-foreground">新房间会立刻进入可调度状态，成员可以在创建时直接勾选。</p>
                <form className="mt-5 space-y-4" onSubmit={handleRoomSubmit}>
                  <FormBlock description="房间名、主题和描述决定前台第一眼看到的内容。" title="基础信息">
                    <div className="grid gap-4 sm:grid-cols-2">
                      <Field label="房间名">
                        <Input required value={roomDraft.name} onChange={(event) => setRoomDraft((current) => ({ ...current, name: event.target.value }))} />
                      </Field>
                      <Field label="主题">
                        <Input value={roomDraft.topic} onChange={(event) => setRoomDraft((current) => ({ ...current, topic: event.target.value }))} />
                      </Field>
                    </div>
                    <Field label="描述">
                      <Textarea value={roomDraft.description} onChange={(event) => setRoomDraft((current) => ({ ...current, description: event.target.value }))} />
                    </Field>
                  </FormBlock>

                  <FormBlock description="这组参数控制房间说话频率和冲突倾向。" title="节奏参数">
                    <div className="grid gap-4 sm:grid-cols-2">
                      <Field label="热度">
                        <Input type="number" value={roomDraft.heat} onChange={(event) => setRoomDraft((current) => ({ ...current, heat: event.target.value }))} />
                      </Field>
                      <Field label="冲突值">
                        <Input
                          type="number"
                          value={roomDraft.conflict_level}
                          onChange={(event) => setRoomDraft((current) => ({ ...current, conflict_level: event.target.value }))}
                        />
                      </Field>
                      <Field label="最小 Tick">
                        <Input
                          type="number"
                          value={roomDraft.tick_min_seconds}
                          onChange={(event) => setRoomDraft((current) => ({ ...current, tick_min_seconds: event.target.value }))}
                        />
                      </Field>
                      <Field label="最大 Tick">
                        <Input
                          type="number"
                          value={roomDraft.tick_max_seconds}
                          onChange={(event) => setRoomDraft((current) => ({ ...current, tick_max_seconds: event.target.value }))}
                        />
                      </Field>
                    </div>
                  </FormBlock>

                  <FormBlock description="预算和上下文参数决定房间能跑多久、压缩多频繁。" title="预算与上下文">
                    <div className="grid gap-4 sm:grid-cols-2">
                      <Field label="日预算">
                        <Input
                          type="number"
                          value={roomDraft.daily_token_budget}
                          onChange={(event) => setRoomDraft((current) => ({ ...current, daily_token_budget: event.target.value }))}
                        />
                      </Field>
                      <Field label="摘要阈值">
                        <Input
                          type="number"
                          value={roomDraft.summary_trigger_count}
                          onChange={(event) => setRoomDraft((current) => ({ ...current, summary_trigger_count: event.target.value }))}
                        />
                      </Field>
                    </div>
                    <Field label="消息保留">
                      <Input
                        type="number"
                        value={roomDraft.message_retention_count}
                        onChange={(event) => setRoomDraft((current) => ({ ...current, message_retention_count: event.target.value }))}
                      />
                    </Field>
                  </FormBlock>

                  <FormBlock description="勾选后，角色会作为初始成员直接加入房间。" title="成员选择">
                    <div className="flex items-center justify-between gap-3">
                      <Label>已选成员</Label>
                      <Badge variant="outline">{selectedPersonas.length} 个</Badge>
                    </div>
                    <ScrollArea className="h-[240px] rounded-[1.35rem] border border-border/65 bg-background/60 p-4">
                      <div className="space-y-3 pr-3">
                        {data.personas.map((persona) => {
                          const checked = selectedPersonas.includes(persona.id)
                          return (
                            <label
                              className="flex cursor-pointer items-start gap-3 rounded-[1.2rem] border border-border/55 bg-background/72 p-3 transition hover:border-primary/25 hover:bg-accent/28"
                              key={persona.id}
                            >
                              <Checkbox
                                checked={checked}
                                onCheckedChange={(value) => {
                                  setSelectedPersonas((current) =>
                                    value ? [...current, persona.id] : current.filter((item) => item !== persona.id),
                                  )
                                }}
                              />
                              <div className="min-w-0">
                                <div className="font-medium text-foreground">
                                  #{persona.id} {persona.name}
                                </div>
                                <div className="text-sm leading-6 text-muted-foreground">{persona.public_identity || "未填写公开身份"}</div>
                              </div>
                            </label>
                          )
                        })}
                      </div>
                    </ScrollArea>
                  </FormBlock>

                  <Button className="w-full" disabled={busyAction === "create-room"} type="submit">
                    {busyAction === "create-room" ? <LoaderCircle className="h-4 w-4 animate-spin" /> : null}
                    创建房间
                  </Button>
                </form>
              </section>
            </div>
          </TabsContent>

          <TabsContent className="space-y-6" value="personas">
            <SectionLead eyebrow="Personas" title="角色工作室" description="列表先看角色是否启用和关键行为参数，右侧再录入身份、动机和模型设置。" />
            <div className="grid gap-6 xl:grid-cols-[minmax(0,1.15fr)_390px]">
              <div className="grid gap-4 lg:grid-cols-2">
                {data.personas.length ? (
                  data.personas.map((persona) => <PersonaRecord key={persona.id} persona={persona} />)
                ) : (
                  <EmptyState title="还没有角色" description="先创建角色，再把它们加入房间。" />
                )}
              </div>

              <section className="panel-surface px-5 py-5 xl:sticky xl:top-6 xl:self-start">
                <p className="tiny-label">Create Persona</p>
                <h2 className="mt-2 font-display text-2xl font-semibold tracking-[-0.05em]">创建角色</h2>
                <p className="mt-2 text-sm leading-7 text-muted-foreground">身份、话风、目标和运行参数分开录入，调度器才能稳定拉开角色差异。</p>
                <form className="mt-5 space-y-4" onSubmit={handlePersonaSubmit}>
                  <FormBlock description="先定义名字、公开身份和说话风格。" title="身份外观">
                    <div className="grid gap-4 sm:grid-cols-2">
                      <Field label="角色名">
                        <Input required value={personaDraft.name} onChange={(event) => setPersonaDraft((current) => ({ ...current, name: event.target.value }))} />
                      </Field>
                      <Field label="公开身份">
                        <Input value={personaDraft.public_identity} onChange={(event) => setPersonaDraft((current) => ({ ...current, public_identity: event.target.value }))} />
                      </Field>
                    </div>
                    <Field label="说话风格">
                      <Textarea value={personaDraft.speaking_style} onChange={(event) => setPersonaDraft((current) => ({ ...current, speaking_style: event.target.value }))} />
                    </Field>
                  </FormBlock>

                  <FormBlock description="这部分决定角色在房间里倾向怎么站队、想得到什么、会避开什么。" title="动机设定">
                    <Field label="立场">
                      <Textarea value={personaDraft.stance} onChange={(event) => setPersonaDraft((current) => ({ ...current, stance: event.target.value }))} />
                    </Field>
                    <Field label="目标">
                      <Textarea value={personaDraft.goal} onChange={(event) => setPersonaDraft((current) => ({ ...current, goal: event.target.value }))} />
                    </Field>
                    <Field label="禁区">
                      <Textarea value={personaDraft.taboo} onChange={(event) => setPersonaDraft((current) => ({ ...current, taboo: event.target.value }))} />
                    </Field>
                  </FormBlock>

                  <FormBlock description="把角色接到阵营、provider 和模型上。" title="接入路由">
                    <div className="grid gap-4 sm:grid-cols-2">
                      <Field label="阵营">
                        <select className={selectClassName} value={personaDraft.faction_id} onChange={(event) => setPersonaDraft((current) => ({ ...current, faction_id: event.target.value }))}>
                          <option value="">未绑定</option>
                          {data.factions.map((faction) => (
                            <option key={faction.id} value={faction.id}>
                              {faction.name}
                            </option>
                          ))}
                        </select>
                      </Field>
                      <Field label="Provider">
                        <select
                          className={selectClassName}
                          value={personaDraft.provider_config_id}
                          onChange={(event) => setPersonaDraft((current) => ({ ...current, provider_config_id: event.target.value }))}
                        >
                          <option value="">未绑定</option>
                          {data.providers.map((provider) => (
                            <option key={provider.id} value={provider.id}>
                              {provider.name}
                            </option>
                          ))}
                        </select>
                      </Field>
                    </div>
                    <Field label="模型名">
                      <Input value={personaDraft.model_name} onChange={(event) => setPersonaDraft((current) => ({ ...current, model_name: event.target.value }))} />
                    </Field>
                  </FormBlock>

                  <FormBlock description="运行参数决定每次发言的长度、速度和攻击性。" title="运行参数">
                    <div className="grid gap-4 sm:grid-cols-2">
                      <Field label="temperature">
                        <Input
                          step="0.1"
                          type="number"
                          value={personaDraft.temperature}
                          onChange={(event) => setPersonaDraft((current) => ({ ...current, temperature: event.target.value }))}
                        />
                      </Field>
                      <Field label="max_tokens">
                        <Input type="number" value={personaDraft.max_tokens} onChange={(event) => setPersonaDraft((current) => ({ ...current, max_tokens: event.target.value }))} />
                      </Field>
                      <Field label="冷却秒数">
                        <Input
                          type="number"
                          value={personaDraft.cooldown_seconds}
                          onChange={(event) => setPersonaDraft((current) => ({ ...current, cooldown_seconds: event.target.value }))}
                        />
                      </Field>
                      <Field label="攻击性">
                        <Input type="number" value={personaDraft.aggression} onChange={(event) => setPersonaDraft((current) => ({ ...current, aggression: event.target.value }))} />
                      </Field>
                      <Field label="活跃度">
                        <Input
                          type="number"
                          value={personaDraft.activity_level}
                          onChange={(event) => setPersonaDraft((current) => ({ ...current, activity_level: event.target.value }))}
                        />
                      </Field>
                    </div>
                  </FormBlock>

                  <div className="flex items-center justify-between rounded-[1.4rem] border border-border/65 bg-background/68 px-4 py-3">
                    <div>
                      <Label htmlFor="persona-enabled">启用角色</Label>
                      <p className="text-xs text-muted-foreground">关闭后仍保留配置，但不会参与调度。</p>
                    </div>
                    <Switch
                      checked={personaDraft.enabled}
                      id="persona-enabled"
                      onCheckedChange={(checked) => setPersonaDraft((current) => ({ ...current, enabled: checked }))}
                    />
                  </div>

                  <Button className="w-full" disabled={busyAction === "create-persona"} type="submit">
                    {busyAction === "create-persona" ? <LoaderCircle className="h-4 w-4 animate-spin" /> : null}
                    创建角色
                  </Button>
                </form>
              </section>
            </div>
          </TabsContent>

          <TabsContent className="space-y-6" value="factions">
            <SectionLead eyebrow="Factions" title="阵营设定" description="阵营影响多个角色的默认偏向和共同行为，是维持长期冲突结构的关键层。" />
            <div className="grid gap-6 xl:grid-cols-[minmax(0,1.05fr)_390px]">
              <div className="grid gap-4">
                {data.factions.length ? (
                  data.factions.map((faction) => <FactionRecord faction={faction} key={faction.id} />)
                ) : (
                  <EmptyState title="还没有阵营" description="先创建阵营，再把角色挂到不同阵营下。" />
                )}
              </div>

              <section className="panel-surface px-5 py-5 xl:sticky xl:top-6 xl:self-start">
                <p className="tiny-label">Create Faction</p>
                <h2 className="mt-2 font-display text-2xl font-semibold tracking-[-0.05em]">创建阵营</h2>
                <p className="mt-2 text-sm leading-7 text-muted-foreground">阵营名要短，共同价值和共同话风要能显著影响多个角色。</p>
                <form className="mt-5 space-y-4" onSubmit={handleFactionSubmit}>
                  <FormBlock description="阵营的基础识别信息。" title="基础信息">
                    <div className="grid gap-4 sm:grid-cols-2">
                      <Field label="阵营名">
                        <Input required value={factionDraft.name} onChange={(event) => setFactionDraft((current) => ({ ...current, name: event.target.value }))} />
                      </Field>
                      <Field label="默认偏向">
                        <Input value={factionDraft.default_bias} onChange={(event) => setFactionDraft((current) => ({ ...current, default_bias: event.target.value }))} />
                      </Field>
                    </div>
                    <Field label="阵营描述">
                      <Textarea value={factionDraft.description} onChange={(event) => setFactionDraft((current) => ({ ...current, description: event.target.value }))} />
                    </Field>
                  </FormBlock>

                  <FormBlock description="这部分内容会成为多个角色共享的默认倾向。" title="群体行为">
                    <Field label="共同价值">
                      <Textarea value={factionDraft.shared_values} onChange={(event) => setFactionDraft((current) => ({ ...current, shared_values: event.target.value }))} />
                    </Field>
                    <Field label="共同话风">
                      <Textarea value={factionDraft.shared_style} onChange={(event) => setFactionDraft((current) => ({ ...current, shared_style: event.target.value }))} />
                    </Field>
                  </FormBlock>

                  <Button className="w-full" disabled={busyAction === "create-faction"} type="submit">
                    {busyAction === "create-faction" ? <LoaderCircle className="h-4 w-4 animate-spin" /> : null}
                    创建阵营
                  </Button>
                </form>
              </section>
            </div>
          </TabsContent>

          <TabsContent className="space-y-6" value="providers">
            <SectionLead eyebrow="Providers" title="模型接入" description="只要求 OpenAI-compatible chat completions 形态，方便把外部模型能力挂进房间。" />
            <div className="grid gap-6 xl:grid-cols-[minmax(0,1.05fr)_390px]">
              <div className="grid gap-4">
                {data.providers.length ? (
                  data.providers.map((provider) => <ProviderRecord key={provider.id} provider={provider} />)
                ) : (
                  <EmptyState title="还没有 provider" description="未配置时系统会回退到本地退化生成器，先保证房间能持续运转。" />
                )}
              </div>

              <section className="panel-surface px-5 py-5 xl:sticky xl:top-6 xl:self-start">
                <p className="tiny-label">Create Provider</p>
                <h2 className="mt-2 font-display text-2xl font-semibold tracking-[-0.05em]">创建 Provider</h2>
                <p className="mt-2 text-sm leading-7 text-muted-foreground">建议填 OpenAI-compatible 地址，例如 `https://example.com/v1`。</p>
                <form className="mt-5 space-y-4" onSubmit={handleProviderSubmit}>
                  <FormBlock description="连接远端模型服务的基础参数。" title="连接信息">
                    <Field label="名称">
                      <Input required value={providerDraft.name} onChange={(event) => setProviderDraft((current) => ({ ...current, name: event.target.value }))} />
                    </Field>
                    <Field label="Base URL">
                      <Input value={providerDraft.base_url} onChange={(event) => setProviderDraft((current) => ({ ...current, base_url: event.target.value }))} />
                    </Field>
                    <Field label="API Key">
                      <Input value={providerDraft.api_key} onChange={(event) => setProviderDraft((current) => ({ ...current, api_key: event.target.value }))} />
                    </Field>
                  </FormBlock>

                  <FormBlock description="默认模型和超时设置会作为角色的起点配置。" title="默认参数">
                    <div className="grid gap-4 sm:grid-cols-2">
                      <Field label="默认模型">
                        <Input value={providerDraft.default_model} onChange={(event) => setProviderDraft((current) => ({ ...current, default_model: event.target.value }))} />
                      </Field>
                      <Field label="超时毫秒">
                        <Input type="number" value={providerDraft.timeout_ms} onChange={(event) => setProviderDraft((current) => ({ ...current, timeout_ms: event.target.value }))} />
                      </Field>
                    </div>
                  </FormBlock>

                  <div className="flex items-center justify-between rounded-[1.4rem] border border-border/65 bg-background/68 px-4 py-3">
                    <div>
                      <Label htmlFor="provider-enabled">启用 Provider</Label>
                      <p className="text-xs text-muted-foreground">关闭后不会被角色选中。</p>
                    </div>
                    <Switch
                      checked={providerDraft.enabled}
                      id="provider-enabled"
                      onCheckedChange={(checked) => setProviderDraft((current) => ({ ...current, enabled: checked }))}
                    />
                  </div>

                  <Button className="w-full" disabled={busyAction === "create-provider"} type="submit">
                    {busyAction === "create-provider" ? <LoaderCircle className="h-4 w-4 animate-spin" /> : null}
                    创建 Provider
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

function RoomControlRow({
  room,
  members,
  busyAction,
  runRoomAction,
}: {
  room: RoomOverview
  members: RoomMemberView[]
  busyAction: string | null
  runRoomAction: (roomID: number, action: "tick" | "pause" | "resume") => Promise<void>
}) {
  const roomActionBusy = busyAction === `pause-${room.id}` || busyAction === `resume-${room.id}` || busyAction === `tick-${room.id}`

  return (
    <article className="grid gap-5 py-5 first:pt-0 last:pb-0 lg:grid-cols-[minmax(0,1fr)_minmax(320px,0.92fr)]">
      <div className="space-y-4">
        <div className="flex items-start justify-between gap-3">
          <div className="space-y-2">
            <p className="tiny-label">Room #{room.id}</p>
            <h3 className="font-display text-2xl font-semibold tracking-[-0.05em]">{room.name}</h3>
            <p className="text-sm leading-7 text-foreground/84">{room.topic || "未填写房间主题"}</p>
          </div>
          <StatusBadge status={room.status} />
        </div>
        <p className="text-sm leading-7 text-muted-foreground">{room.description || "未填写房间描述。"}</p>
        <div className="flex flex-wrap gap-2">
          {members.length ? (
            members.slice(0, 6).map((member) => (
              <Badge key={member.id} variant="outline">
                {member.persona_name}
              </Badge>
            ))
          ) : (
            <Badge variant="outline">暂未配置成员</Badge>
          )}
          {members.length > 6 ? <span className="px-1 text-xs text-muted-foreground">+{members.length - 6} 个</span> : null}
        </div>
      </div>

      <div className="rounded-[1.45rem] border border-border/65 bg-secondary/38 p-4">
        <div className="grid gap-3 sm:grid-cols-2">
          <StatText label="消息" value={room.message_count} />
          <StatText label="今日 Token" value={room.tokens_today} />
          <StatText label="最近活动" value={relativeUnix(room.last_message_at_unix)} />
          <StatText label="Tick" value={`${room.tick_min_seconds}-${room.tick_max_seconds}s`} />
        </div>
        <div className="mt-4 flex flex-wrap gap-2">
          <Button asChild size="sm" variant="outline">
            <a href={`/rooms/${room.id}`}>查看前台</a>
          </Button>
          <Button disabled={roomActionBusy} onClick={() => runRoomAction(room.id, "tick")} size="sm">
            {busyAction === `tick-${room.id}` ? <LoaderCircle className="h-4 w-4 animate-spin" /> : <RefreshCcw className="h-4 w-4" />}
            立即 Tick
          </Button>
          {room.status === "paused" ? (
            <Button disabled={roomActionBusy} onClick={() => runRoomAction(room.id, "resume")} size="sm" variant="ink">
              <PlayCircle className="h-4 w-4" />
              恢复运行
            </Button>
          ) : (
            <Button disabled={roomActionBusy} onClick={() => runRoomAction(room.id, "pause")} size="sm" variant="ink">
              <PauseCircle className="h-4 w-4" />
              暂停房间
            </Button>
          )}
        </div>
      </div>
    </article>
  )
}

function PersonaRecord({ persona }: { persona: Persona }) {
  return (
    <article className="panel-surface-muted hover-rise p-5">
      <div className="flex items-start justify-between gap-3">
        <div className="space-y-1">
          <p className="tiny-label">Persona #{persona.id}</p>
          <h3 className="font-display text-2xl font-semibold tracking-[-0.05em]">{persona.name}</h3>
          <p className="text-sm leading-7 text-muted-foreground">{persona.public_identity || "未填写公开身份"}</p>
        </div>
        <Badge variant={persona.enabled ? "success" : "outline"}>{persona.enabled ? "启用中" : "已停用"}</Badge>
      </div>
      <p className="mt-4 text-sm leading-7 text-foreground/88">{persona.speaking_style || "未填写说话风格"}</p>
      {persona.goal ? <p className="mt-2 line-clamp-2 text-sm leading-6 text-muted-foreground">目标：{persona.goal}</p> : null}
      <div className="mt-4 grid gap-2 sm:grid-cols-2">
        <StatText label="阵营 ID" value={persona.faction_id || "-"} />
        <StatText label="Provider ID" value={persona.provider_config_id || "-"} />
        <StatText label="temperature" value={persona.temperature} />
        <StatText label="max_tokens" value={persona.max_tokens} />
      </div>
    </article>
  )
}

function FactionRecord({ faction }: { faction: Faction }) {
  return (
    <article className="panel-surface-muted hover-rise p-5">
      <div className="flex items-center justify-between gap-3">
        <div className="space-y-1">
          <p className="tiny-label">Faction #{faction.id}</p>
          <h3 className="font-display text-2xl font-semibold tracking-[-0.05em]">{faction.name}</h3>
        </div>
        {faction.default_bias ? <Badge variant="secondary">{faction.default_bias}</Badge> : null}
      </div>
      <p className="mt-4 text-sm leading-7 text-foreground/88">{faction.description || "未填写阵营描述。"}</p>
      <div className="mt-4 grid gap-2 sm:grid-cols-2">
        <StatText label="共同价值" value={faction.shared_values || "-"} />
        <StatText label="共同话风" value={faction.shared_style || "-"} />
      </div>
    </article>
  )
}

function ProviderRecord({ provider }: { provider: ProviderConfig }) {
  return (
    <article className="panel-surface-muted hover-rise p-5">
      <div className="flex items-center justify-between gap-3">
        <div className="space-y-1">
          <p className="tiny-label">Provider #{provider.id}</p>
          <h3 className="font-display text-2xl font-semibold tracking-[-0.05em]">{provider.name}</h3>
        </div>
        <Badge variant={provider.enabled ? "success" : "outline"}>{provider.enabled ? "已启用" : "已停用"}</Badge>
      </div>
      <p className="mt-4 text-sm leading-7 text-foreground/88">{provider.base_url || "未填写 Base URL"}</p>
      <div className="mt-4 grid gap-2 sm:grid-cols-2">
        <StatText label="默认模型" value={provider.default_model || "-"} />
        <StatText label="超时毫秒" value={provider.timeout_ms} />
        <StatText label="API Key" value={maskKey(provider.api_key)} />
        <StatText label="最后更新" value={relativeUnix(provider.updated_at ? Math.floor(new Date(provider.updated_at).getTime() / 1000) : 0)} />
      </div>
    </article>
  )
}

function FormBlock({ title, description, children }: { title: string; description: string; children: ReactNode }) {
  return (
    <section className="panel-surface-muted space-y-4 px-4 py-4">
      <div>
        <h3 className="font-display text-lg font-semibold tracking-[-0.04em]">{title}</h3>
        <p className="mt-1 text-sm leading-6 text-muted-foreground">{description}</p>
      </div>
      <div className="space-y-4">{children}</div>
    </section>
  )
}

function Field({ label, children }: { label: string; children: ReactNode }) {
  return (
    <div className="space-y-2">
      <Label>{label}</Label>
      {children}
    </div>
  )
}

function StatText({ label, value }: { label: string; value: string | number }) {
  return (
    <div className="rounded-[1.1rem] border border-border/55 bg-background/65 px-3 py-3">
      <div className="data-kicker">{label}</div>
      <div className="mt-1 text-sm font-medium leading-6 text-foreground">{value}</div>
    </div>
  )
}

function TabLabel({ label, count }: { label: string; count: number }) {
  return (
    <span className="inline-flex items-center gap-2">
      <span>{label}</span>
      <span className="rounded-full bg-background/60 px-2 py-0.5 text-[11px] text-muted-foreground">{count}</span>
    </span>
  )
}
