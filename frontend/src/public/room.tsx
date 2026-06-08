import { Send } from "lucide-react"
import { type FormEvent, useCallback, useEffect, useRef, useState } from "react"
import { postJSON } from "../shared/lib/api"
import {
  countChars,
  formatDateTime,
  initials,
  messageKindLabel,
  messageSpeaker,
  statusLabel,
  statusTone,
} from "../shared/lib/format"
import { EmptyState, Shell } from "../shared/shell"
import type { Message, RoomMemberView, RoomPageData } from "../shared/types"
import { Badge } from "../shared/ui/badge"
import { Button } from "../shared/ui/button"
import { Textarea } from "../shared/ui/textarea"
import { cn } from "../shared/lib/utils"

export function RoomPage({ data }: { data: RoomPageData }) {
  const [messages, setMessages] = useState<Message[]>(data.messages)
  const [composer, setComposer] = useState("")
  const [sending, setSending] = useState(false)
  const [connected, setConnected] = useState(false)
  const [autoScroll, setAutoScroll] = useState(true)
  const renderedIDs = useRef(new Set(data.messages.map((m) => m.id)))
  const listRef = useRef<HTMLDivElement | null>(null)

  useEffect(() => {
    const source = new EventSource(`/api/rooms/${data.room.id}/events`)
    source.onopen = () => setConnected(true)
    source.onerror = () => setConnected(false)
    source.addEventListener("message", (event) => {
      setConnected(true)
      const message = JSON.parse(event.data) as Message
      if (renderedIDs.current.has(message.id)) return
      renderedIDs.current.add(message.id)
      setMessages((current) => [...current, message])
    })
    return () => source.close()
  }, [data.room.id])

  useEffect(() => {
    if (!autoScroll || !listRef.current) return
    listRef.current.scrollTop = listRef.current.scrollHeight
  }, [messages, autoScroll])

  const handleScroll = useCallback(() => {
    const el = listRef.current
    if (!el) return
    setAutoScroll(el.scrollHeight - el.scrollTop - el.clientHeight < 48)
  }, [])

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    const content = composer.trim()
    if (!content) return
    setSending(true)
    try {
      await postJSON(`/api/rooms/${data.room.id}/input`, { content })
      setComposer("")
    } catch (error) {
      console.error(error)
    } finally {
      setSending(false)
    }
  }

  function appendMention(name: string) {
    setComposer((current) => {
      const joiner = current && !current.endsWith(" ") ? " " : ""
      return `${current}${joiner}@${name} `
    })
  }

  return (
    <Shell
      title={data.room.name}
      actions={
        <>
          <StreamDot connected={connected} />
          <Badge variant={statusTone(data.room.status)}>{statusLabel(data.room.status)}</Badge>
          <Button asChild variant="outline">
            <a href="/">返回</a>
          </Button>
        </>
      }
    >
      <div className="grid gap-4 xl:grid-cols-[260px_minmax(0,1fr)_300px]">
        <aside className="space-y-3">
          <RoomMeta data={data} />
          {data.latestSummary ? (
            <section className="rounded-lg border border-border bg-card p-4">
              <h2 className="text-sm font-semibold">最近摘要</h2>
              <p className="mt-2 text-sm leading-relaxed text-foreground/85">{data.latestSummary.content}</p>
            </section>
          ) : null}
        </aside>

        <section className="flex min-h-[600px] flex-col overflow-hidden rounded-lg border border-border bg-card">
          <div className="flex items-center justify-between gap-3 border-b border-border px-4 py-2 text-xs text-muted-foreground">
            <span>
              {messages.length} / {data.messageCount} 条
            </span>
            <Button
              size="sm"
              variant={autoScroll ? "default" : "outline"}
              onClick={() => setAutoScroll((v) => !v)}
            >
              {autoScroll ? "自动滚动" : "跟随停止"}
            </Button>
          </div>

          <div ref={listRef} className="flex-1 space-y-2 overflow-y-auto p-4" onScroll={handleScroll}>
            {messages.length ? (
              messages.map((message) => <MessageItem key={message.id} message={message} />)
            ) : (
              <EmptyState title="暂无消息" />
            )}
          </div>
        </section>

        <aside className="space-y-3">
          <Composer
            composer={composer}
            sending={sending}
            members={data.members}
            onChange={setComposer}
            onMention={appendMention}
            onSubmit={handleSubmit}
          />
          <Members members={data.members} />
        </aside>
      </div>
    </Shell>
  )
}

function RoomMeta({ data }: { data: RoomPageData }) {
  const { room } = data
  return (
    <section className="space-y-3 rounded-lg border border-border bg-card p-4">
      {room.topic ? <p className="text-sm text-foreground/85">{room.topic}</p> : null}
      {room.description ? <p className="text-sm text-muted-foreground">{room.description}</p> : null}
      <div className="grid gap-1.5">
        <Meta label="成员" value={data.memberCount} />
        <Meta label="消息" value={data.messageCount} />
        <Meta label="今日 Token" value={data.tokensToday} />
        <Meta label="热度" value={room.heat} />
        <Meta label="冲突值" value={room.conflict_level} />
        <Meta label="Tick" value={`${room.tick_min_seconds}-${room.tick_max_seconds}s`} />
        <Meta label="日预算" value={room.daily_token_budget} />
      </div>
    </section>
  )
}

function Meta({ label, value }: { label: string; value: string | number }) {
  return (
    <div className="flex items-center justify-between text-sm">
      <span className="text-muted-foreground">{label}</span>
      <span className="font-medium tabular-nums">{value}</span>
    </div>
  )
}

function Composer({
  composer,
  sending,
  members,
  onChange,
  onMention,
  onSubmit,
}: {
  composer: string
  sending: boolean
  members: RoomMemberView[]
  onChange: (value: string) => void
  onMention: (name: string) => void
  onSubmit: (event: FormEvent<HTMLFormElement>) => void
}) {
  return (
    <section className="space-y-3 rounded-lg border border-border bg-card p-4">
      <div className="flex items-center justify-between">
        <h2 className="text-sm font-semibold">插一句</h2>
        <span className="text-xs text-muted-foreground tabular-nums">{countChars(composer)} / 280</span>
      </div>
      {members.length ? (
        <div className="flex flex-wrap gap-1.5">
          {members.map((member) => (
            <Button
              key={member.persona_id}
              size="sm"
              variant="outline"
              onClick={() => onMention(member.persona_name)}
            >
              @{member.persona_name}
            </Button>
          ))}
        </div>
      ) : null}
      <form className="space-y-2" onSubmit={onSubmit}>
        <Textarea
          maxLength={280}
          placeholder="直接输入，或 @ 点名角色"
          value={composer}
          onChange={(event) => onChange(event.target.value)}
        />
        <Button className="w-full" disabled={sending || !composer.trim()} type="submit">
          <Send className="h-3.5 w-3.5" />
          发送
        </Button>
      </form>
    </section>
  )
}

function Members({ members }: { members: RoomMemberView[] }) {
  if (!members.length) return null
  return (
    <section className="space-y-2 rounded-lg border border-border bg-card p-4">
      <h2 className="text-sm font-semibold">角色阵容</h2>
      <div className="space-y-2">
        {members.map((member) => (
          <div key={member.persona_id} className="flex items-start gap-2">
            <span className="flex h-8 w-8 shrink-0 items-center justify-center rounded-md bg-primary/10 text-xs font-semibold text-primary">
              {initials(member.persona_name)}
            </span>
            <div className="min-w-0 flex-1">
              <div className="flex items-center gap-1.5">
                <span className="truncate text-sm font-medium">{member.persona_name}</span>
                {member.faction_name ? (
                  <Badge variant="secondary" className="shrink-0">
                    {member.faction_name}
                  </Badge>
                ) : null}
              </div>
              {member.public_identity ? (
                <p className="truncate text-xs text-muted-foreground">{member.public_identity}</p>
              ) : null}
            </div>
          </div>
        ))}
      </div>
    </section>
  )
}

function MessageItem({ message }: { message: Message }) {
  const tone =
    message.kind === "user"
      ? "border-signal-cool/20 bg-signal-cool/5"
      : message.kind === "system" || message.kind === "summary"
        ? "border-border bg-secondary/40"
        : "border-primary/15 bg-primary/5"

  const avatarTone =
    message.kind === "user"
      ? "bg-signal-cool/10 text-signal-cool"
      : message.kind === "system" || message.kind === "summary"
        ? "bg-secondary text-secondary-foreground"
        : "bg-primary/10 text-primary"

  return (
    <article className={cn("rounded-lg border p-3", tone)}>
      <div className="flex gap-3">
        <div
          className={cn(
            "flex h-9 w-9 shrink-0 items-center justify-center rounded-lg text-xs font-semibold",
            avatarTone,
          )}
        >
          {initials(messageSpeaker(message))}
        </div>
        <div className="min-w-0 flex-1">
          <div className="flex flex-wrap items-center justify-between gap-2">
            <div className="flex items-center gap-2">
              <strong className="text-sm font-semibold">{messageSpeaker(message)}</strong>
              {message.kind !== "chat" ? (
                <Badge variant={message.kind === "user" ? "outline" : "secondary"}>
                  {messageKindLabel(message.kind)}
                </Badge>
              ) : null}
            </div>
            <time className="text-xs text-muted-foreground">{formatDateTime(message.created_at)}</time>
          </div>
          <p className="mt-1 whitespace-pre-wrap text-sm leading-relaxed text-foreground/90">{message.content}</p>
        </div>
      </div>
    </article>
  )
}

function StreamDot({ connected }: { connected: boolean }) {
  return (
    <span className="inline-flex items-center gap-1.5 text-xs text-muted-foreground">
      <span className={cn("h-1.5 w-1.5 rounded-full", connected ? "bg-success" : "bg-warning animate-pulse")} />
      {connected ? "已连接" : "重连中"}
    </span>
  )
}
