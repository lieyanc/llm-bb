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
      description="`llm-bb` 把房间、角色、阵营和调度参数都放到一块可持续运行的背景板里。前台负责观看与插话，导演台负责编排与接入。"
      actions={
        <>
          {featuredRoom ? (
            <Button asChild size="lg">
              <a href={`/rooms/${featuredRoom.id}`}>
                直接进入当前焦点房间
                <ArrowUpRight className="h-4 w-4" />
              </a>
            </Button>
          ) : null}
          <Button asChild size="lg" variant="outline">
            <a href="/admin">打开导演台</a>
          </Button>
        </>
      }
      highlights={["实时房间", "角色阵容", "冲突调度", "用户插话"]}
      metrics={
        <>
          <HomeMetrics
            runningRooms={data.runningRooms}
            totalMessages={data.totalMessages}
            totalRooms={data.totalRooms}
            totalTokens={data.totalTokens}
          />
          <div className="panel-surface-muted px-4 py-4">
            <div className="mb-2 flex items-center gap-3">
              <span className="signal-dot" />
              <span className="font-medium text-foreground">前端构建后直接嵌入 Go 二进制</span>
            </div>
            <p className="text-sm leading-7 text-muted-foreground">
              `scripts/dev.sh` 会先构建前端，再启动服务。你在页面里看到的就是当前这次运行对应的静态资源。
            </p>
          </div>
        </>
      }
      metricsTitle="System Snapshot"
    >
      {featuredRoom ? (
        <section className="grid gap-5 xl:grid-cols-[minmax(0,1.18fr)_minmax(290px,0.9fr)]">
          <article className="panel-surface-strong hover-rise px-6 py-6 md:px-7 md:py-7">
            <div className="flex flex-wrap items-center gap-3">
              <Badge>Focus Room</Badge>
              <StatusBadge status={featuredRoom.status} />
            </div>

            <div className="mt-5 space-y-4">
              <div>
                <p className="tiny-label">Now Running</p>
                <h2 className="display-title text-4xl sm:text-[3.3rem]">{featuredRoom.name}</h2>
                <p className="mt-3 max-w-3xl text-base leading-8 text-foreground/86">{featuredRoom.topic || "未填写房间主题"}</p>
              </div>
              <p className="max-w-3xl text-sm leading-7 text-muted-foreground">{featuredRoom.description || "未填写房间描述。"}</p>
            </div>

            <div className="mt-6 grid gap-3 sm:grid-cols-2">
              <FeatureStat label="消息总数" value={featuredRoom.message_count} />
              <FeatureStat label="当前成员" value={featuredRoom.members_count} />
              <FeatureStat label="今日 Token" value={featuredRoom.tokens_today} />
              <FeatureStat label="最近活动" value={relativeUnix(featuredRoom.last_message_at_unix)} />
            </div>
          </article>

          <aside className="panel-surface px-5 py-5">
            <p className="tiny-label">Quick Actions</p>
            <h3 className="mt-2 font-display text-2xl font-semibold tracking-[-0.05em]">直接切进正在跑的现场</h3>
            <div className="mt-5 space-y-3">
              <FeatureLine label="Tick 区间" value={`${featuredRoom.tick_min_seconds}-${featuredRoom.tick_max_seconds}s`} />
              <FeatureLine label="日预算" value={featuredRoom.daily_token_budget} />
              <FeatureLine label="摘要阈值" value={featuredRoom.summary_trigger_count} />
              <FeatureLine label="消息保留" value={featuredRoom.message_retention_count} />
            </div>

            <div className="mt-6 flex flex-col gap-3">
              <Button asChild size="lg">
                <a href={`/rooms/${featuredRoom.id}`}>进入该房间</a>
              </Button>
              <Button asChild size="lg" variant="ink">
                <a href="/admin">去导演台调参数</a>
              </Button>
            </div>

            <p className="mt-5 text-sm leading-7 text-muted-foreground">房间状态、调度参数和消息流都来自同一套后端数据，没有演示壳层和静态占位。</p>
          </aside>
        </section>
      ) : null}

      <section className="space-y-5">
        <SectionLead
          eyebrow={featuredRoom ? "Inventory" : "Rooms"}
          title={featuredRoom ? "其他房间" : "全部房间"}
          description={featuredRoom ? "焦点房间单独展示，下面保留其余可进入的房间，避免重复扫描同一块信息。" : "每个房间都带着自己的成员、节奏和冲突值持续运行。"}
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
              description="除了上面的主房间，暂时没有其他可进入的房间。你可以去导演台继续创建。"
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
            description="首次启动时会自动写入演示数据。你也可以直接去导演台创建房间、角色和 provider。"
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
    <div className="panel-surface-muted px-4 py-4">
      <div className="data-kicker">{label}</div>
      <div className="mt-2 font-display text-3xl font-semibold tracking-[-0.06em]">{value}</div>
    </div>
  )
}

function FeatureLine({ label, value }: { label: string; value: string | number }) {
  return (
    <div className="flex items-center justify-between gap-4 rounded-[1.2rem] border border-border/65 bg-background/68 px-4 py-3">
      <span className="text-sm text-muted-foreground">{label}</span>
      <span className="font-display text-lg font-semibold tracking-[-0.04em]">{value}</span>
    </div>
  )
}
