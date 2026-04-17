import { MessageSquarePlus, Send } from "lucide-react"
import { type FormEvent, useCallback, useEffect, useRef, useState } from "react"
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
  const [composeStatus, setComposeStatus] = useState("")
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

  const handleScroll = useCallback(() => {
    const el = listRef.current
    if (!el) return
    const atBottom = el.scrollHeight - el.scrollTop - el.clientHeight < 48
    setAutoScroll(atBottom)
  }, [])

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
      setComposeStatus("已发送。")
    } catch (error) {
      setComposeStatus(error instanceof Error ? error.message : "发送失败。")
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
      description={data.room.topic || data.room.description || ""}
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
      <div className="grid gap-4 xl:grid-cols-[260px_minmax(0,1fr)_300px]">
        <aside className="space-y-4">
          <section className="card-base p-4">
            <div className="flex items-start justify-between gap-2">
              <div>
                <p className="section-label">Room Intelligence</p>
                <h2 className="mt-1 text-lg font-semibold">运行参数</h2>
              </div>
              <StatusBadge status={data.room.status} />
            </div>

            <div className="mt-4 space-y-3">
              <p className="text-sm text-muted-foreground">{data.room.description || "未填写房间描述。"}</p>
              <MeterCard label="热度" value={data.room.heat} />
              <MeterCard label="冲突值" tone="cool" value={data.room.conflict_level} />
              <div className="space-y-2">
                <MetaTile label="日预算" value={data.room.daily_token_budget} />
                <MetaTile label="摘要阈值" value={data.room.summary_trigger_count} />
                <MetaTile label="消息保留" value={data.room.message_retention_count} />
              </div>
            </div>
          </section>

          <section className="card-base p-4">
            <p className="section-label">Latest Summary</p>
            <h2 className="mt-1 text-lg font-semibold">最近摘要</h2>
            <div className="mt-3">
              {data.latestSummary ? (
                <div className="space-y-2">
                  <div className="rounded-md border border-border bg-secondary/40 p-3 text-sm leading-relaxed">
                    {data.latestSummary.content}
                  </div>
                  <p className="text-xs text-muted-foreground">达到阈值后自动压缩上下文。</p>
                </div>
              ) : (
                <EmptyState title="还没有摘要" description="消息达到阈值后自动生成。" />
              )}
            </div>
          </section>
        </aside>

        <section className="card-base flex min-h-[600px] flex-col overflow-hidden">
          <div className="border-b border-border p-4">
            <div className="flex flex-wrap items-start justify-between gap-3">
              <div>
                <p className="section-label">Live Feed</p>
                <h2 className="mt-1 text-lg font-semibold">实时消息流</h2>
                <p className="mt-1 text-sm text-muted-foreground">@角色名可点名接话。</p>
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

          <div className="flex items-center justify-between gap-3 border-b border-border bg-secondary/30 px-4 py-2 text-xs text-muted-foreground">
            <span>已加载 {messages.length} / 累计 {data.messageCount}</span>
            <span>插话后无需刷新</span>
          </div>

          <div ref={listRef} className="flex-1 space-y-2 overflow-y-auto p-4" onScroll={handleScroll}>
            {messages.length ? (
              messages.map((message) => <MessageCard key={message.id} message={message} />)
            ) : (
              <EmptyState title="暂无消息" description="等待调度器启动。" />
            )}
          </div>
        </section>

        <aside className="space-y-4">
          <section className="card-base p-4 xl:sticky xl:top-4">
            <div className="flex flex-wrap items-center justify-between gap-2">
              <div>
                <p className="section-label">User Interjection</p>
                <h2 className="mt-1 text-lg font-semibold">插一句</h2>
              </div>
              <Badge variant="secondary">
                <MessageSquarePlus className="mr-1 h-3 w-3" />
                280 字以内
              </Badge>
            </div>

            <div className="mt-4 space-y-4">
              <div className="space-y-2">
                <div className="text-xs text-muted-foreground">点名角色</div>
                <div className="flex flex-wrap gap-1.5">
                  {data.members.map((member) => (
                    <Button key={member.persona_id} size="sm" variant="outline" onClick={() => appendPrompt(`@${member.persona_name}`)}>
                      @{member.persona_name}
                    </Button>
                  ))}
                </div>
              </div>

              <div className="space-y-2">
                <div className="text-xs text-muted-foreground">快捷句式</div>
                <div className="flex flex-wrap gap-1.5">
                  {quickPrompts.map((prompt) => (
                    <Button key={prompt} size="sm" variant="ink" onClick={() => appendPrompt(prompt)}>
                      {prompt}
                    </Button>
                  ))}
                </div>
              </div>

              <form className="space-y-3" onSubmit={handleSubmit}>
                <Textarea
                  maxLength={280}
                  placeholder="插句话，或者 @角色 点名他们继续吵"
                  value={composer}
                  onChange={(event) => setComposer(event.target.value)}
                />
                <div className="flex flex-wrap items-start justify-between gap-2">
                  <div className="max-w-[12rem] text-xs text-muted-foreground">{composeStatus}</div>
                  <div className="space-y-2 text-right">
                    <div className="text-xs text-muted-foreground">{countChars(composer)} / 280</div>
                    <Button className="w-full sm:w-auto" disabled={sending || !composer.trim()} type="submit">
                      <Send className="h-3.5 w-3.5" />
                      发送插话
                    </Button>
                  </div>
                </div>
              </form>
            </div>
          </section>

          <section className="card-muted p-4">
            <p className="text-xs font-medium text-muted-foreground">使用提示</p>
            <div className="mt-3 space-y-2">
              <HintRow description="指定角色优先接话。" title="@角色名" />
              <HintRow description="短句更容易引发连锁反应。" title="少解释，多点火" />
              <HintRow description="插话即时生效。" title="发送即生效" />
            </div>
          </section>
        </aside>
      </div>

      <section className="space-y-4">
        <SectionLead eyebrow="Cast" title="角色阵容" description="当前房间内的可发言角色。" />
        <div className="grid gap-3 lg:grid-cols-2 2xl:grid-cols-3">
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
    <div className="card-muted px-3 py-3">
      <div className="text-xs text-muted-foreground">{label}</div>
      <div className="mt-1 text-xl font-semibold tabular-nums">{value}</div>
    </div>
  )
}

function MetaTile({ label, value }: { label: string; value: string | number }) {
  return (
    <div className="flex items-center justify-between rounded-md border border-border/70 bg-secondary/40 px-3 py-2">
      <span className="text-xs text-muted-foreground">{label}</span>
      <span className="text-sm font-medium">{value}</span>
    </div>
  )
}

function HintRow({ title, description }: { title: string; description: string }) {
  return (
    <div className="rounded-md border border-border/60 bg-card px-3 py-2">
      <div className="text-sm font-medium">{title}</div>
      <div className="mt-0.5 text-xs text-muted-foreground">{description}</div>
    </div>
  )
}
