import { Activity, ArrowUpRight, Bot, Flame, Radio, Sparkles, Users } from "lucide-react"
import type { ReactNode } from "react"
import type { Message, RoomMemberView, RoomOverview, RoomStatus } from "../types"
import { cn } from "../lib/utils"
import { formatDateTime, initials, messageKindLabel, messageSpeaker, relativeUnix, statusLabel, statusTone } from "../lib/format"
import { Badge } from "./ui/badge"
import { Button } from "./ui/button"
import { ThemeToggle } from "./theme-toggle"

export function AppFrame({
  eyebrow,
  title,
  description,
  actions,
  metrics,
  metricsTitle = "Live Snapshot",
  highlights = [],
  children,
}: {
  eyebrow: string
  title: ReactNode
  description: ReactNode
  actions?: ReactNode
  metrics?: ReactNode
  metricsTitle?: string
  highlights?: string[]
  children: ReactNode
}) {
  return (
    <div className="container py-6 pb-12">
      <div className="mb-4 flex items-center gap-2 text-sm text-muted-foreground">
        <span className="status-dot" />
        <span className="font-mono text-xs">LLM Background Board</span>
        <div className="ml-auto">
          <ThemeToggle />
        </div>
      </div>

      <div className="grid gap-4 xl:grid-cols-[minmax(0,1.4fr)_minmax(300px,0.8fr)]">
        <header className="card-base p-6">
          <div className="space-y-4">
            <p className="section-label">{eyebrow}</p>
            <h1 className="text-2xl font-bold tracking-tight sm:text-3xl">{title}</h1>
            <div className="max-w-2xl text-sm leading-relaxed text-muted-foreground">{description}</div>
            {actions ? <div className="flex flex-wrap gap-2 pt-1">{actions}</div> : null}
            {highlights.length ? (
              <div className="flex flex-wrap gap-1.5 pt-1">
                {highlights.map((item, index) => (
                  <span
                    key={`${item}-${index}`}
                    className="rounded-md border border-border bg-secondary/60 px-2 py-0.5 text-xs text-muted-foreground"
                  >
                    {item}
                  </span>
                ))}
              </div>
            ) : null}
          </div>
        </header>

        <aside className="card-base p-5">
          <div className="mb-4 flex items-center justify-between gap-3">
            <h2 className="text-sm font-semibold">{metricsTitle}</h2>
            <span className="rounded-md bg-primary/10 px-2 py-0.5 text-xs font-medium text-primary">Live</span>
          </div>
          <div className="grid gap-3">{metrics}</div>
        </aside>
      </div>

      <main className="mt-6 space-y-6">{children}</main>
    </div>
  )
}

export function MetricTile({ label, value, hint, icon }: { label: string; value: ReactNode; hint?: ReactNode; icon?: ReactNode }) {
  return (
    <div className="card-muted px-3 py-3">
      <div className="flex items-center justify-between gap-2">
        <span className="text-xs text-muted-foreground">{label}</span>
        {icon ? <div className="text-primary">{icon}</div> : null}
      </div>
      <div className="mt-1 text-xl font-semibold tabular-nums">{value}</div>
      {hint ? <p className="mt-1 text-xs text-muted-foreground">{hint}</p> : null}
    </div>
  )
}

export function StatusBadge({ status, className }: { status: RoomStatus; className?: string }) {
  return (
    <Badge className={className} variant={statusTone(status)}>
      {statusLabel(status)}
    </Badge>
  )
}

export function MeterCard({
  label,
  value,
  tone = "warm",
  hint,
}: {
  label: string
  value: number
  tone?: "warm" | "cool"
  hint?: string
}) {
  return (
    <div className="card-muted px-3 py-3">
      <div className="mb-2 flex items-center justify-between gap-2">
        <span className="text-sm text-muted-foreground">{label}</span>
        <span className="text-lg font-semibold tabular-nums">{value}</span>
      </div>
      <div className="h-1.5 overflow-hidden rounded-full bg-secondary">
        <div
          className={cn("h-full rounded-full transition-all", tone === "warm" ? "bg-primary" : "bg-signal-cool")}
          style={{ width: `${Math.max(6, Math.min(100, value))}%` }}
        />
      </div>
      {hint ? <p className="mt-2 text-xs text-muted-foreground">{hint}</p> : null}
    </div>
  )
}

export function SectionLead({ eyebrow, title, description }: { eyebrow: string; title: string; description?: string }) {
  return (
    <div className="flex flex-col gap-2 lg:flex-row lg:items-end lg:justify-between">
      <div>
        <p className="section-label">{eyebrow}</p>
        <h2 className="mt-1 text-xl font-bold tracking-tight sm:text-2xl">{title}</h2>
      </div>
      {description ? <p className="max-w-xl text-sm text-muted-foreground">{description}</p> : null}
    </div>
  )
}

export function RoomCard({ room, ctaLabel = "打开房间" }: { room: RoomOverview; ctaLabel?: string }) {
  return (
    <article className="card-base flex h-full flex-col gap-4 p-5 transition-colors hover:border-primary/30">
      <div className="flex items-start justify-between gap-3">
        <div>
          <p className="section-label">Room #{room.id}</p>
          <h3 className="mt-1 text-lg font-semibold">{room.name}</h3>
          <p className="mt-1 text-sm text-foreground/80">{room.topic || "未填写房间主题"}</p>
        </div>
        <StatusBadge className="shrink-0" status={room.status} />
      </div>

      <p className="text-sm text-muted-foreground">{room.description || "未填写房间描述。"}</p>

      <div className="space-y-2">
        <SignalBar label="热度" tone="warm" value={room.heat} />
        <SignalBar label="冲突值" tone="cool" value={room.conflict_level} />
      </div>

      <div className="grid gap-2 sm:grid-cols-2">
        <InfoChip icon={<Bot className="h-3.5 w-3.5" />} label="消息" value={room.message_count} />
        <InfoChip icon={<Users className="h-3.5 w-3.5" />} label="成员" value={room.members_count} />
        <InfoChip icon={<Activity className="h-3.5 w-3.5" />} label="今日 Token" value={room.tokens_today} />
        <InfoChip icon={<Sparkles className="h-3.5 w-3.5" />} label="最近活动" value={relativeUnix(room.last_message_at_unix)} />
      </div>

      <div className="mt-auto flex items-center justify-between gap-3 border-t border-border pt-3">
        <p className="text-xs text-muted-foreground">
          Tick {room.tick_min_seconds}-{room.tick_max_seconds}s
        </p>
        <Button asChild size="sm">
          <a href={`/rooms/${room.id}`}>
            {ctaLabel}
            <ArrowUpRight className="h-3.5 w-3.5" />
          </a>
        </Button>
      </div>
    </article>
  )
}

export function PersonaSpotlight({ member }: { member: RoomMemberView }) {
  return (
    <article className="card-muted p-4 transition-colors hover:border-primary/30">
      <div className="flex items-start gap-3">
        <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg bg-primary/10 text-sm font-semibold text-primary">
          {initials(member.persona_name)}
        </div>
        <div className="min-w-0 flex-1 space-y-2">
          <div className="flex flex-wrap items-center justify-between gap-2">
            <div>
              <h3 className="text-sm font-semibold">{member.persona_name}</h3>
              <p className="text-xs text-foreground/70">{member.public_identity || "未填写公开身份"}</p>
            </div>
            <div className="flex flex-wrap gap-1">
              {member.faction_name ? <Badge variant="secondary">{member.faction_name}</Badge> : null}
              {member.provider_name ? <Badge variant="outline">{member.provider_name}</Badge> : null}
            </div>
          </div>

          <p className="text-sm text-muted-foreground">{member.speaking_style || "未填写说话风格"}</p>

          <div className="grid gap-1.5 sm:grid-cols-2">
            <MiniMeta label="攻击性" value={member.aggression} />
            <MiniMeta label="活跃度" value={member.activity_level} />
            <MiniMeta label="冷却" value={`${member.cooldown_seconds}s`} />
            <MiniMeta label="台词上限" value={member.max_tokens} />
          </div>
        </div>
      </div>
    </article>
  )
}

export function MessageCard({ message, className }: { message: Message; className?: string }) {
  const tone =
    message.kind === "user"
      ? "border-signal-cool/20 bg-signal-cool/5"
      : message.kind === "system" || message.kind === "summary"
        ? "border-border bg-secondary/40"
        : "border-primary/15 bg-primary/5"

  const dotColor =
    message.kind === "user"
      ? "bg-signal-cool"
      : message.kind === "system" || message.kind === "summary"
        ? "bg-muted-foreground"
        : "bg-primary"

  return (
    <article className={cn("rounded-lg border p-3 transition-colors hover:border-primary/20", tone, className)}>
      <div className="flex gap-3">
        <span className={cn("mt-1 h-8 w-0.5 shrink-0 rounded-full", dotColor)} />
        <div
          className={cn(
            "flex h-9 w-9 shrink-0 items-center justify-center rounded-lg text-xs font-semibold",
            message.kind === "chat" && "bg-primary/10 text-primary",
            message.kind === "user" && "bg-signal-cool/10 text-signal-cool",
            (message.kind === "system" || message.kind === "summary") && "bg-secondary text-secondary-foreground",
          )}
        >
          {initials(messageSpeaker(message))}
        </div>
        <div className="min-w-0 flex-1">
          <div className="flex flex-wrap items-center justify-between gap-2">
            <div className="flex flex-wrap items-center gap-2">
              <strong className="text-sm font-semibold">{messageSpeaker(message)}</strong>
              <Badge variant={message.kind === "chat" ? "default" : message.kind === "user" ? "outline" : "secondary"}>
                {messageKindLabel(message.kind)}
              </Badge>
            </div>
            <time className="text-xs text-muted-foreground">{formatDateTime(message.created_at)}</time>
          </div>
          <p className="mt-2 whitespace-pre-wrap text-sm leading-relaxed text-foreground/90">{message.content}</p>
        </div>
      </div>
    </article>
  )
}

export function StreamStatus({ connected }: { connected: boolean }) {
  return (
    <div className="inline-flex items-center gap-1.5 rounded-md border border-border bg-card px-2 py-1 text-xs text-muted-foreground">
      <span className={cn("h-1.5 w-1.5 rounded-full", connected ? "bg-success" : "bg-warning animate-pulse")} />
      {connected ? "SSE 已连接" : "SSE 重连中"}
    </div>
  )
}

export function EmptyState({
  title,
  description,
  action,
}: {
  title: string
  description: string
  action?: ReactNode
}) {
  return (
    <div className="card-muted px-5 py-6 text-center">
      <h3 className="text-sm font-semibold">{title}</h3>
      <p className="mt-1 text-sm text-muted-foreground">{description}</p>
      {action ? <div className="mt-3">{action}</div> : null}
    </div>
  )
}

export function HeroActionGroup() {
  return (
    <>
      <Button asChild>
        <a href="/">
          返回总览
          <ArrowUpRight className="h-3.5 w-3.5" />
        </a>
      </Button>
      <Button asChild variant="outline">
        <a href="/admin">打开导演台</a>
      </Button>
    </>
  )
}

export function HomeMetrics({ totalRooms, runningRooms, totalMessages, totalTokens }: { totalRooms: number; runningRooms: number; totalMessages: number; totalTokens: number }) {
  return (
    <div className="grid gap-3 sm:grid-cols-2">
      <MetricTile label="房间总数" value={totalRooms} icon={<Users className="h-3.5 w-3.5" />} />
      <MetricTile label="运行中" value={runningRooms} icon={<Flame className="h-3.5 w-3.5" />} />
      <MetricTile label="累计消息" value={totalMessages} icon={<Bot className="h-3.5 w-3.5" />} />
      <MetricTile label="今日 Token" value={totalTokens} icon={<Activity className="h-3.5 w-3.5" />} />
    </div>
  )
}

export function AdminMetrics({
  rooms,
  runningRooms,
  personas,
  providers,
  totalMessages,
  totalTokens,
}: {
  rooms: number
  runningRooms: number
  personas: number
  providers: number
  totalMessages: number
  totalTokens: number
}) {
  return (
    <div className="grid gap-3 sm:grid-cols-2">
      <MetricTile label="房间" value={rooms} icon={<Users className="h-3.5 w-3.5" />} />
      <MetricTile label="运行中" value={runningRooms} icon={<Radio className="h-3.5 w-3.5" />} />
      <MetricTile label="角色" value={personas} icon={<Bot className="h-3.5 w-3.5" />} />
      <MetricTile label="Provider" value={providers} icon={<Sparkles className="h-3.5 w-3.5" />} />
      <MetricTile label="累计消息" value={totalMessages} icon={<Activity className="h-3.5 w-3.5" />} />
      <MetricTile label="今日 Token" value={totalTokens} icon={<Flame className="h-3.5 w-3.5" />} />
    </div>
  )
}

function SignalBar({ label, value, tone }: { label: string; value: number; tone: "warm" | "cool" }) {
  return (
    <div className="space-y-1">
      <div className="flex items-center justify-between gap-2 text-sm">
        <span className="text-muted-foreground">{label}</span>
        <span className="font-medium tabular-nums">{value}</span>
      </div>
      <div className="h-1.5 overflow-hidden rounded-full bg-secondary">
        <div
          className={cn("h-full rounded-full", tone === "warm" ? "bg-primary" : "bg-signal-cool")}
          style={{ width: `${Math.max(8, Math.min(100, value))}%` }}
        />
      </div>
    </div>
  )
}

function InfoChip({ icon, label, value }: { icon: ReactNode; label: string; value: ReactNode }) {
  return (
    <div className="flex items-center gap-2 rounded-md border border-border/70 bg-secondary/40 px-2.5 py-2">
      <div className="text-primary">{icon}</div>
      <span className="text-xs text-muted-foreground">{label}</span>
      <span className="ml-auto text-sm font-medium tabular-nums">{value}</span>
    </div>
  )
}

function MiniMeta({ label, value }: { label: string; value: ReactNode }) {
  return (
    <div className="flex items-center justify-between rounded-md border border-border/60 bg-secondary/30 px-2.5 py-1.5">
      <span className="text-xs text-muted-foreground">{label}</span>
      <span className="text-xs font-medium">{value}</span>
    </div>
  )
}
