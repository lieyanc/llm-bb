import { MessageSquarePlus, Send } from "lucide-react"
import { type FormEvent, useEffect, useRef, useState } from "react"
import { postJSON } from "../lib/api"
import { countChars } from "../lib/format"
import type { Message, RoomPageData } from "../types"
import { AppFrame, EmptyState, HeroActionGroup, MessageCard, MeterCard, PersonaSpotlight, SectionLead, StatusBadge, StreamStatus } from "./shared"
import { Badge } from "./ui/badge"
import { Button } from "./ui/button"
import { Textarea } from "./ui/textarea"

const quickPrompts = ["继续吵，别停。", "挑一个最装客观的人正面回。", "站队，别端水。"]

export function RoomPage({ data }: { data: RoomPageData }) {
  const [messages, setMessages] = useState<Message[]>(data.messages)
  const [composer, setComposer] = useState("")
  const [composeStatus, setComposeStatus] = useState("输入后会直接广播到房间，并推动下一轮调度。")
  const [sending, setSending] = useState(false)
  const [connected, setConnected] = useState(false)
  const [autoScroll, setAutoScroll] = useState(true)
  const renderedIDs = useRef(new Set(data.messages.map((message) => message.id)))
  const listRef = useRef<HTMLDivElement | null>(null)

  useEffect(() => {
    const source = new EventSource(`/api/rooms/${data.room.id}/events`)
    source.onopen = () => setConnected(true)
    source.onerror = () => setConnected(false)
    source.addEventListener("message", (event) => {
      setConnected(true)
      const message = JSON.parse(event.data) as Message
      if (renderedIDs.current.has(message.id)) {
        return
      }
      renderedIDs.current.add(message.id)
      setMessages((current) => [...current, message])
    })

    return () => {
      source.close()
    }
  }, [data.room.id])

  useEffect(() => {
    if (!autoScroll || !listRef.current) {
      return
    }
    listRef.current.scrollTop = listRef.current.scrollHeight
  }, [messages, autoScroll])

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    const content = composer.trim()
    if (!content) {
      return
    }

    setSending(true)
    setComposeStatus("发送中…")

    try {
      await postJSON(`/api/rooms/${data.room.id}/input`, { content })
      setComposer("")
      setComposeStatus("插话已送出，正在推动下一轮调度。")
    } catch (error) {
      setComposeStatus(error instanceof Error ? error.message : "网络异常，发送失败。")
    } finally {
      setSending(false)
    }
  }

  function appendPrompt(fragment: string) {
    setComposer((current) => {
      const joiner = current && !current.endsWith(" ") ? " " : ""
      return `${current}${joiner}${fragment}`
    })
  }

  function scrollToBottom() {
    if (!listRef.current) {
      return
    }
    listRef.current.scrollTo({ top: listRef.current.scrollHeight, behavior: "smooth" })
  }

  return (
    <AppFrame
      eyebrow={`Room #${data.room.id}`}
      title={data.room.name}
      description={data.room.topic || data.room.description || "这个房间正在持续接收调度、生成台词，并接受观众实时插话。"}
      actions={<HeroActionGroup />}
      highlights={["实时消息流", "插话控制", "摘要压缩", "角色阵容"]}
      metrics={
        <div className="grid gap-3 sm:grid-cols-2">
          <HeaderMetric label="当前成员" value={data.memberCount} />
          <HeaderMetric label="总消息数" value={data.messageCount} />
          <HeaderMetric label="今日 Token" value={data.tokensToday} />
          <HeaderMetric label="Tick 区间" value={`${data.room.tick_min_seconds}-${data.room.tick_max_seconds}s`} />
        </div>
      }
      metricsTitle="Room Snapshot"
    >
      <div className="grid gap-6 xl:grid-cols-[280px_minmax(0,1fr)_320px]">
        <aside className="space-y-5">
          <section className="panel-surface px-5 py-5">
            <div className="flex items-start justify-between gap-3">
              <div>
                <p className="tiny-label">Room Intelligence</p>
                <h2 className="mt-2 font-display text-2xl font-semibold tracking-[-0.05em]">运行参数</h2>
              </div>
              <StatusBadge status={data.room.status} />
            </div>

            <div className="mt-5 space-y-4">
              <p className="text-sm leading-7 text-muted-foreground">{data.room.description || "未填写房间描述。"}</p>
              <MeterCard hint="越高越容易连续接话和抢节奏。" label="热度" value={data.room.heat} />
              <MeterCard hint="越高越倾向出现反驳、阴阳和站队。" label="冲突值" tone="cool" value={data.room.conflict_level} />
              <div className="grid gap-3">
                <MetaTile label="日预算" value={data.room.daily_token_budget} />
                <MetaTile label="摘要阈值" value={data.room.summary_trigger_count} />
                <MetaTile label="消息保留" value={data.room.message_retention_count} />
              </div>
            </div>
          </section>

          <section className="panel-surface px-5 py-5">
            <p className="tiny-label">Latest Summary</p>
            <h2 className="mt-2 font-display text-2xl font-semibold tracking-[-0.05em]">最近摘要</h2>
            <div className="mt-4">
              {data.latestSummary ? (
                <div className="space-y-3">
                  <div className="rounded-[1.35rem] border border-border/65 bg-background/68 p-4 text-sm leading-7 text-foreground/88">
                    {data.latestSummary.content}
                  </div>
                  <p className="text-sm leading-7 text-muted-foreground">达到摘要阈值后，系统会压缩上下文，帮助房间长期稳定运行。</p>
                </div>
              ) : (
                <EmptyState title="还没有摘要" description="等累计消息达到阈值后，系统会自动压缩历史上下文。" />
              )}
            </div>
          </section>
        </aside>

        <section className="panel-surface flex min-h-[680px] flex-col overflow-hidden">
          <div className="border-b border-border/60 px-5 py-5">
            <div className="flex flex-wrap items-start justify-between gap-4">
              <div className="space-y-2">
                <p className="tiny-label">Live Feed</p>
                <h2 className="font-display text-[2rem] font-semibold tracking-[-0.05em]">实时消息流</h2>
                <p className="max-w-2xl text-sm leading-7 text-muted-foreground">点名角色会优先接话；如果 provider 未配置，系统会回退到本地生成器。</p>
              </div>
              <div className="flex flex-wrap items-center gap-2">
                <StreamStatus connected={connected} />
                <Button size="sm" variant={autoScroll ? "default" : "outline"} onClick={() => setAutoScroll((value) => !value)}>
                  {autoScroll ? "自动滚动" : "手动查看"}
                </Button>
                {!autoScroll ? (
                  <Button size="sm" variant="ghost" onClick={scrollToBottom}>
                    回到底部
                  </Button>
                ) : null}
              </div>
            </div>
          </div>

          <div className="flex flex-wrap items-center justify-between gap-3 border-b border-border/60 bg-secondary/32 px-5 py-3 text-xs uppercase tracking-[0.24em] text-muted-foreground">
            <span>已加载 {messages.length} / 累计 {data.messageCount}</span>
            <span>插话后无需刷新</span>
          </div>

          <div ref={listRef} className="flex-1 space-y-3 overflow-y-auto px-5 py-5">
            {messages.length ? (
              messages.map((message) => <MessageCard key={message.id} message={message} />)
            ) : (
              <EmptyState title="房间还没有消息" description="调度器启动后，角色会逐步开始接话和拱火。" />
            )}
          </div>
        </section>

        <aside className="space-y-5">
          <section className="panel-surface px-5 py-5 xl:sticky xl:top-6">
            <div className="flex flex-wrap items-center justify-between gap-3">
              <div>
                <p className="tiny-label">User Interjection</p>
                <h2 className="mt-2 font-display text-2xl font-semibold tracking-[-0.05em]">插一句</h2>
              </div>
              <Badge variant="secondary">
                <MessageSquarePlus className="mr-1 h-3.5 w-3.5" />
                280 字以内
              </Badge>
            </div>

            <div className="mt-5 space-y-5">
              <div className="space-y-3">
                <div className="data-kicker">点名角色</div>
                <div className="flex flex-wrap gap-2">
                  {data.members.map((member) => (
                    <Button key={member.persona_id} size="sm" variant="outline" onClick={() => appendPrompt(`@${member.persona_name}`)}>
                      @{member.persona_name}
                    </Button>
                  ))}
                </div>
              </div>

              <div className="space-y-3">
                <div className="data-kicker">快捷句式</div>
                <div className="flex flex-wrap gap-2">
                  {quickPrompts.map((prompt) => (
                    <Button key={prompt} size="sm" variant="ink" onClick={() => appendPrompt(prompt)}>
                      {prompt}
                    </Button>
                  ))}
                </div>
              </div>

              <form className="space-y-4" onSubmit={handleSubmit}>
                <Textarea
                  maxLength={280}
                  placeholder="插句话，或者 @角色 点名他们继续吵"
                  value={composer}
                  onChange={(event) => setComposer(event.target.value)}
                />
                <div className="flex flex-wrap items-start justify-between gap-3">
                  <div className="max-w-[14rem] text-sm leading-7 text-muted-foreground">{composeStatus}</div>
                  <div className="space-y-3 text-right">
                    <div className="text-xs text-muted-foreground">{countChars(composer)} / 280</div>
                    <Button className="w-full sm:w-auto" disabled={sending || !composer.trim()} type="submit">
                      <Send className="h-4 w-4" />
                      发送插话
                    </Button>
                  </div>
                </div>
              </form>
            </div>
          </section>

          <section className="panel-surface-muted px-5 py-5">
            <p className="tiny-label">Input Tips</p>
            <div className="mt-4 space-y-3">
              <HintRow description="优先让指定角色接住这轮话题。" title="@角色名" />
              <HintRow description="短句更容易触发快速连锁反应。" title="少解释，多点火" />
              <HintRow description="每次插话都会立即广播并推动下一轮调度。" title="发送即生效" />
            </div>
          </section>
        </aside>
      </div>

      <section className="space-y-5">
        <SectionLead eyebrow="Cast" title="角色阵容" description="按身份、阵营和行为参数查看当前可发言角色。" />
        <div className="grid gap-4 lg:grid-cols-2 2xl:grid-cols-3">
          {data.members.map((member) => (
            <PersonaSpotlight key={member.persona_id} member={member} />
          ))}
        </div>
      </section>
    </AppFrame>
  )
}

function HeaderMetric({ label, value }: { label: string; value: string | number }) {
  return (
    <div className="panel-surface-muted px-4 py-4">
      <div className="data-kicker">{label}</div>
      <div className="mt-2 font-display text-3xl font-semibold tracking-[-0.06em]">{value}</div>
    </div>
  )
}

function MetaTile({ label, value }: { label: string; value: string | number }) {
  return (
    <div className="rounded-[1.2rem] border border-border/65 bg-background/68 px-4 py-3">
      <div className="data-kicker">{label}</div>
      <div className="mt-1 text-sm font-medium text-foreground">{value}</div>
    </div>
  )
}

function HintRow({ title, description }: { title: string; description: string }) {
  return (
    <div className="rounded-[1.15rem] border border-border/55 bg-background/55 px-4 py-3">
      <div className="text-sm font-medium text-foreground">{title}</div>
      <div className="mt-1 text-sm leading-6 text-muted-foreground">{description}</div>
    </div>
  )
}
