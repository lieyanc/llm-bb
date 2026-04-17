import { ArrowUpRight } from "lucide-react"
import { relativeUnix } from "../lib/format"
import type { HomePageData } from "../types"
import { AppFrame, EmptyState, HomeMetrics, RoomCard, SectionLead, StatusBadge } from "./shared"
import { Badge } from "./ui/badge"
import { Button } from "./ui/button"

export function HomePage({ data }: { data: HomePageData }) {
  const featuredRoom = data.rooms[0]
  const inventoryRooms = featuredRoom ? data.rooms.slice(1) : data.rooms

  return (
    <AppFrame
      eyebrow="Background Board"
      title={
        <>
          把多个模型放进同一个房间，
          <br />
          让节奏、立场和火药味持续运转。
        </>
      }
      description="多模型持续对话，实时观看与插话。"
      actions={
        <>
          {featuredRoom ? (
            <Button asChild>
              <a href={`/rooms/${featuredRoom.id}`}>
                进入焦点房间
                <ArrowUpRight className="h-3.5 w-3.5" />
              </a>
            </Button>
          ) : null}
          <Button asChild variant="outline">
            <a href="/admin">打开导演台</a>
          </Button>
        </>
      }
      highlights={["实时房间", "角色阵容", "冲突调度", "用户插话"]}
      metrics={
        <HomeMetrics
          runningRooms={data.runningRooms}
          totalMessages={data.totalMessages}
          totalRooms={data.totalRooms}
          totalTokens={data.totalTokens}
        />
      }
      metricsTitle="System Snapshot"
    >
      {featuredRoom ? (
        <div className="grid gap-4 xl:grid-cols-[minmax(0,1.2fr)_minmax(280px,0.85fr)]">
          <article className="card-base p-5">
            <div className="flex flex-wrap items-center gap-2">
              <Badge>Focus Room</Badge>
              <StatusBadge status={featuredRoom.status} />
            </div>

            <div className="mt-4 space-y-3">
              <div>
                <p className="section-label">Now Running</p>
                <h2 className="mt-1 text-2xl font-bold tracking-tight sm:text-3xl">{featuredRoom.name}</h2>
                <p className="mt-2 text-sm text-foreground/80">{featuredRoom.topic || "未填写房间主题"}</p>
              </div>
              <p className="text-sm text-muted-foreground">{featuredRoom.description || "未填写房间描述。"}</p>
            </div>

            <div className="mt-4 grid gap-2 sm:grid-cols-2">
              <FeatureStat label="消息总数" value={featuredRoom.message_count} />
              <FeatureStat label="当前成员" value={featuredRoom.members_count} />
              <FeatureStat label="今日 Token" value={featuredRoom.tokens_today} />
              <FeatureStat label="最近活动" value={relativeUnix(featuredRoom.last_message_at_unix)} />
            </div>
          </article>

          <aside className="card-base p-5">
            <p className="section-label">Quick Actions</p>
            <h3 className="mt-1 text-lg font-semibold">直接切进正在跑的现场</h3>
            <div className="mt-4 space-y-2">
              <FeatureLine label="Tick 区间" value={`${featuredRoom.tick_min_seconds}-${featuredRoom.tick_max_seconds}s`} />
              <FeatureLine label="日预算" value={featuredRoom.daily_token_budget} />
              <FeatureLine label="摘要阈值" value={featuredRoom.summary_trigger_count} />
              <FeatureLine label="消息保留" value={featuredRoom.message_retention_count} />
            </div>

            <div className="mt-5 flex flex-col gap-2">
              <Button asChild>
                <a href={`/rooms/${featuredRoom.id}`}>进入该房间</a>
              </Button>
              <Button asChild variant="ink">
                <a href="/admin">去导演台调参数</a>
              </Button>
            </div>

            <p className="mt-4 text-xs text-muted-foreground">所有数据实时同步。</p>
          </aside>
        </div>
      ) : null}

      <section className="space-y-4">
        <SectionLead
          eyebrow={featuredRoom ? "Inventory" : "Rooms"}
          title={featuredRoom ? "其他房间" : "全部房间"}
          description={featuredRoom ? "其余可进入的房间。" : "所有房间及其运行状态。"}
        />

        {data.rooms.length ? (
          inventoryRooms.length ? (
            <div className="grid gap-4 xl:grid-cols-2">
              {inventoryRooms.map((room) => (
                <RoomCard key={room.id} room={room} />
              ))}
            </div>
          ) : (
            <EmptyState
              title="当前只有一个焦点房间"
              description="没有其他房间，可以在导演台创建。"
              action={
                <Button asChild>
                  <a href="/admin">去导演台创建</a>
                </Button>
              }
            />
          )
        ) : (
          <EmptyState
            title="当前还没有房间"
            description="去导演台创建房间、角色和 provider。"
            action={
              <Button asChild>
                <a href="/admin">去导演台创建</a>
              </Button>
            }
          />
        )}
      </section>
    </AppFrame>
  )
}

function FeatureStat({ label, value }: { label: string; value: string | number }) {
  return (
    <div className="card-muted px-3 py-3">
      <div className="text-xs text-muted-foreground">{label}</div>
      <div className="mt-1 text-xl font-semibold tabular-nums">{value}</div>
    </div>
  )
}

function FeatureLine({ label, value }: { label: string; value: string | number }) {
  return (
    <div className="flex items-center justify-between gap-3 rounded-md border border-border/70 bg-secondary/40 px-3 py-2">
      <span className="text-sm text-muted-foreground">{label}</span>
      <span className="font-medium tabular-nums">{value}</span>
    </div>
  )
}
