import { Activity, ArrowUpRight, Bot, Flame, Radio, Sparkles, Users } from "lucide-react"
import type { ReactNode } from "react"
import type { Message, RoomMemberView, RoomOverview, RoomStatus } from "../types"
import { cn } from "../lib/utils"
import { formatDateTime, initials, messageKindLabel, messageSpeaker, relativeUnix, statusLabel, statusTone } from "../lib/format"
import { Badge } from "./ui/badge"
import { Button } from "./ui/button"

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
    <div className="relative z-10">
      <div className="container py-5 pb-12 md:py-8 lg:pb-14">
        <div className="mb-4 flex flex-wrap items-center justify-between gap-3">
          <div className="inline-flex items-center gap-3 rounded-full border border-border/70 bg-background/75 px-4 py-2 shadow-float backdrop-blur">
            <span className="signal-dot animate-pulse-soft" />
            <span className="font-mono text-[11px] uppercase tracking-[0.34em] text-muted-foreground">LLM Ensemble Theatre</span>
          </div>
          <div className="inline-flex items-center gap-2 rounded-full border border-border/70 bg-background/75 px-4 py-2 text-xs text-muted-foreground shadow-float backdrop-blur">
            <Radio className="h-3.5 w-3.5 text-primary" />
            单二进制部署 · SQLite · SSE · React
          </div>
        </div>

        <section className="grid gap-5 xl:grid-cols-[minmax(0,1.3fr)_minmax(320px,0.92fr)]">
          <header className="panel-surface-strong hero-shell animate-enter px-6 py-6 md:px-8 md:py-8">
            <div className="space-y-6">
              <div className="flex flex-wrap items-start justify-between gap-4">
                <div className="space-y-4">
                  <span className="eyebrow-chip">{eyebrow}</span>
                  <div className="space-y-4">
                    <h1 className="display-title max-w-4xl text-4xl leading-[0.92] sm:text-5xl lg:text-[4.4rem]">{title}</h1>
                    <div className="max-w-2xl text-sm leading-7 text-foreground/82 sm:text-base sm:leading-8">{description}</div>
                  </div>
                </div>
                <div className="hidden rounded-[1.45rem] border border-border/70 bg-background/68 px-4 py-3 text-right backdrop-blur xl:block">
                  <p className="data-kicker">Operating Mode</p>
                  <p className="mt-2 font-display text-xl font-semibold tracking-[-0.05em]">Readable by Default</p>
                  <p className="mt-1 max-w-[14rem] text-sm leading-6 text-muted-foreground">实时状态优先，装饰退后一步，操作入口始终靠前。</p>
                </div>
              </div>

              {actions ? <div className="flex flex-wrap gap-3">{actions}</div> : null}
            </div>
          </header>

          <aside className="panel-surface animate-enter animate-enter-delay-2 px-6 py-6">
            <div className="mb-5 flex items-start justify-between gap-4">
              <div>
                <p className="tiny-label">Signals</p>
                <h2 className="font-display text-[1.9rem] font-semibold tracking-[-0.05em]">{metricsTitle}</h2>
              </div>
              <div className="rounded-full border border-border/70 bg-background/70 px-3 py-1.5 text-xs text-muted-foreground">Live</div>
            </div>
            <div className="surface-grid">{metrics}</div>
          </aside>
        </section>

        {highlights.length ? (
          <div className="panel-surface mt-5 animate-enter animate-enter-delay-3 px-4 py-3">
            <div className="flex flex-wrap gap-2">
              {highlights.map((item, index) => (
                <span
                  key={`${item}-${index}`}
                  className="rounded-full border border-border/70 bg-background/65 px-3 py-1.5 text-[11px] uppercase tracking-[0.28em] text-muted-foreground"
                >
                  {item}
                </span>
              ))}
            </div>
          </div>
        ) : null}

        <main className="mt-6 space-y-6 animate-enter animate-enter-delay-4">{children}</main>
      </div>
    </div>
  )
}

export function MetricTile({ label, value, hint, icon }: { label: string; value: ReactNode; hint?: ReactNode; icon?: ReactNode }) {
  return (
    <div className="panel-surface-muted px-4 py-4">
      <div className="flex items-start justify-between gap-3">
        <span className="data-kicker">{label}</span>
        {icon ? <div className="rounded-full bg-background/70 p-2 text-primary">{icon}</div> : null}
      </div>
      <div className="mt-4 font-display text-3xl font-semibold tracking-[-0.06em]">{value}</div>
      {hint ? <p className="mt-2 text-sm leading-6 text-muted-foreground">{hint}</p> : null}
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
    <div className="panel-surface-muted px-4 py-4">
      <div className="mb-3 flex items-center justify-between gap-3">
        <span className="text-sm font-medium text-muted-foreground">{label}</span>
        <span className="font-display text-xl tracking-[-0.05em]">{value}</span>
      </div>
      <div className="h-2.5 overflow-hidden rounded-full bg-background/70">
        <div
          className={cn("h-full rounded-full transition-all", tone === "warm" ? "bg-primary" : "bg-signal-cool")}
          style={{ width: `${Math.max(6, Math.min(100, value))}%` }}
        />
      </div>
      {hint ? <p className="mt-3 text-sm leading-6 text-muted-foreground">{hint}</p> : null}
    </div>
  )
}

export function SectionLead({ eyebrow, title, description }: { eyebrow: string; title: string; description: string }) {
  return (
    <div className="flex flex-col gap-3 lg:flex-row lg:items-end lg:justify-between">
      <div className="space-y-2">
        <p className="tiny-label">{eyebrow}</p>
        <h2 className="display-title text-3xl sm:text-[2.65rem]">{title}</h2>
      </div>
      <p className="max-w-2xl text-sm leading-7 text-muted-foreground">{description}</p>
    </div>
  )
}

export function RoomCard({ room, ctaLabel = "打开房间" }: { room: RoomOverview; ctaLabel?: string }) {
  return (
    <article className="panel-surface hover-rise group flex h-full flex-col gap-5 p-5">
      <div className="flex items-start justify-between gap-4">
        <div className="space-y-2">
          <p className="tiny-label">Room #{room.id}</p>
          <h3 className="font-display text-2xl font-semibold tracking-[-0.05em]">{room.name}</h3>
          <p className="text-sm leading-7 text-foreground/84">{room.topic || "未填写房间主题"}</p>
        </div>
        <StatusBadge className="shrink-0" status={room.status} />
      </div>

      <p className="text-sm leading-7 text-muted-foreground">{room.description || "未填写房间描述。"}</p>

      <div className="space-y-3">
        <SignalBar label="热度" tone="warm" value={room.heat} />
        <SignalBar label="冲突值" tone="cool" value={room.conflict_level} />
      </div>

      <div className="grid gap-2 sm:grid-cols-2">
        <InfoChip icon={<Bot className="h-4 w-4 text-primary" />} label="消息" value={room.message_count} />
        <InfoChip icon={<Users className="h-4 w-4 text-primary" />} label="成员" value={room.members_count} />
        <InfoChip icon={<Activity className="h-4 w-4 text-primary" />} label="今日 Token" value={room.tokens_today} />
        <InfoChip icon={<Sparkles className="h-4 w-4 text-primary" />} label="最近活动" value={relativeUnix(room.last_message_at_unix)} />
      </div>

      <div className="mt-auto flex items-center justify-between gap-3 border-t border-border/60 pt-4">
        <p className="text-xs uppercase tracking-[0.2em] text-muted-foreground">
          Tick {room.tick_min_seconds}-{room.tick_max_seconds}s
        </p>
        <Button asChild size="sm">
          <a href={`/rooms/${room.id}`}>
            {ctaLabel}
            <ArrowUpRight className="h-4 w-4" />
          </a>
        </Button>
      </div>
    </article>
  )
}

export function PersonaSpotlight({ member }: { member: RoomMemberView }) {
  return (
    <article className="panel-surface-muted hover-rise px-4 py-4">
      <div className="flex items-start gap-4">
        <div className="flex h-12 w-12 shrink-0 items-center justify-center rounded-2xl bg-primary/12 font-display text-lg font-semibold text-primary">
          {initials(member.persona_name)}
        </div>
        <div className="min-w-0 flex-1 space-y-3">
          <div className="flex flex-wrap items-center justify-between gap-2">
            <div className="space-y-1">
              <h3 className="font-display text-lg font-semibold tracking-[-0.05em]">{member.persona_name}</h3>
              <p className="text-sm text-foreground/84">{member.public_identity || "未填写公开身份"}</p>
            </div>
            <div className="flex flex-wrap gap-2">
              {member.faction_name ? <Badge variant="secondary">{member.faction_name}</Badge> : null}
              {member.provider_name ? <Badge variant="outline">{member.provider_name}</Badge> : null}
            </div>
          </div>

          <p className="text-sm leading-7 text-muted-foreground">{member.speaking_style || "未填写说话风格"}</p>

          <div className="grid gap-2 sm:grid-cols-2">
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
      ? "border-signal-cool/20 bg-accent/45"
      : message.kind === "system" || message.kind === "summary"
        ? "border-border/75 bg-secondary/62"
        : "border-primary/15 bg-background/82"

  const dotTone =
    message.kind === "user"
      ? "bg-signal-cool"
      : message.kind === "system" || message.kind === "summary"
        ? "bg-muted-foreground"
        : "bg-primary"

  return (
    <article className={cn("rounded-[1.5rem] border p-4 shadow-sm transition hover:border-primary/25 hover:shadow-glow", tone, className)}>
      <div className="flex gap-4">
        <span className={cn("mt-1.5 h-10 w-1 shrink-0 rounded-full", dotTone)} />
        <div
          className={cn(
            "flex h-11 w-11 shrink-0 items-center justify-center rounded-2xl font-display text-sm font-semibold",
            message.kind === "chat" && "bg-primary/12 text-primary",
            message.kind === "user" && "bg-signal-cool/12 text-signal-cool",
            (message.kind === "system" || message.kind === "summary") && "bg-background/72 text-secondary-foreground",
          )}
        >
          {initials(messageSpeaker(message))}
        </div>
        <div className="min-w-0 flex-1">
          <div className="flex flex-wrap items-center justify-between gap-3">
            <div className="flex flex-wrap items-center gap-2">
              <strong className="font-display text-base font-semibold tracking-[-0.05em]">{messageSpeaker(message)}</strong>
              <Badge variant={message.kind === "chat" ? "default" : message.kind === "user" ? "outline" : "secondary"}>
                {messageKindLabel(message.kind)}
              </Badge>
            </div>
            <time className="text-xs text-muted-foreground">{formatDateTime(message.created_at)}</time>
          </div>
          <p className="mt-3 whitespace-pre-wrap text-sm leading-7 text-foreground/92">{message.content}</p>
        </div>
      </div>
    </article>
  )
}

export function StreamStatus({ connected }: { connected: boolean }) {
  return (
    <div className="inline-flex items-center gap-2 rounded-full border border-border/70 bg-background/72 px-3 py-1.5 text-xs text-muted-foreground">
      <span className={cn("h-2.5 w-2.5 rounded-full", connected ? "bg-success animate-pulse-soft" : "bg-warning animate-pulse-soft")} />
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
    <div className="panel-surface-muted px-6 py-8">
      <div className="space-y-4">
        <Badge variant="outline">Empty</Badge>
        <div>
          <h3 className="font-display text-2xl font-semibold tracking-[-0.05em]">{title}</h3>
          <p className="mt-3 max-w-2xl text-sm leading-7 text-muted-foreground">{description}</p>
        </div>
        {action ? <div>{action}</div> : null}
      </div>
    </div>
  )
}

export function HeroActionGroup() {
  return (
    <>
      <Button asChild>
        <a href="/">
          返回总览
          <ArrowUpRight className="h-4 w-4" />
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
      <MetricTile label="房间总数" value={totalRooms} icon={<Users className="h-4 w-4" />} />
      <MetricTile label="运行中" value={runningRooms} icon={<Flame className="h-4 w-4" />} />
      <MetricTile label="累计消息" value={totalMessages} icon={<Bot className="h-4 w-4" />} />
      <MetricTile label="今日 Token" value={totalTokens} icon={<Activity className="h-4 w-4" />} />
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
      <MetricTile label="房间" value={rooms} icon={<Users className="h-4 w-4" />} />
      <MetricTile label="运行中" value={runningRooms} icon={<Radio className="h-4 w-4" />} />
      <MetricTile label="角色" value={personas} icon={<Bot className="h-4 w-4" />} />
      <MetricTile label="Provider" value={providers} icon={<Sparkles className="h-4 w-4" />} />
      <MetricTile label="累计消息" value={totalMessages} icon={<Activity className="h-4 w-4" />} />
      <MetricTile label="今日 Token" value={totalTokens} icon={<Flame className="h-4 w-4" />} />
    </div>
  )
}

function SignalBar({ label, value, tone }: { label: string; value: number; tone: "warm" | "cool" }) {
  return (
    <div className="space-y-2">
      <div className="flex items-center justify-between gap-3 text-sm">
        <span className="text-muted-foreground">{label}</span>
        <span className="font-display text-lg tracking-[-0.04em]">{value}</span>
      </div>
      <div className="h-2 overflow-hidden rounded-full bg-secondary">
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
    <div className="rounded-[1.2rem] border border-border/65 bg-background/68 px-3 py-3">
      <div className="mb-2 flex items-center gap-2">
        <div className="flex h-8 w-8 items-center justify-center rounded-full bg-primary/10">{icon}</div>
        <span className="data-kicker">{label}</span>
      </div>
      <div className="truncate text-sm font-medium text-foreground">{value}</div>
    </div>
  )
}

function MiniMeta({ label, value }: { label: string; value: ReactNode }) {
  return (
    <div className="rounded-[1rem] border border-border/55 bg-background/60 px-3 py-2">
      <div className="data-kicker">{label}</div>
      <div className="mt-1 text-sm font-medium text-foreground">{value}</div>
    </div>
  )
}
