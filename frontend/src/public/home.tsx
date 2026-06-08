import { ArrowUpRight } from "lucide-react"
import { relativeUnix, statusLabel, statusTone } from "../shared/lib/format"
import { EmptyState, MetricGrid, MetricTile, Shell } from "../shared/shell"
import type { HomePageData, RoomOverview } from "../shared/types"
import { Badge } from "../shared/ui/badge"
import { Button } from "../shared/ui/button"
import { Card, CardContent, CardFooter, CardHeader, CardTitle } from "../shared/ui/card"

export function HomePage({ data }: { data: HomePageData }) {
  return (
    <Shell
      title="llm-bb"
      actions={
        <Button asChild variant="outline">
          <a href="/admin">导演台</a>
        </Button>
      }
    >
      <MetricGrid>
        <MetricTile label="房间" value={data.totalRooms} />
        <MetricTile label="运行中" value={data.runningRooms} />
        <MetricTile label="累计消息" value={data.totalMessages} />
        <MetricTile label="今日 Token" value={data.totalTokens} />
      </MetricGrid>

      {data.rooms.length ? (
        <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-3">
          {data.rooms.map((room) => (
            <RoomCard key={room.id} room={room} />
          ))}
        </div>
      ) : (
        <EmptyState
          title="还没有房间"
          action={
            <Button asChild>
              <a href="/admin">去创建</a>
            </Button>
          }
        />
      )}
    </Shell>
  )
}

function RoomCard({ room }: { room: RoomOverview }) {
  return (
    <a
      className="group block h-full"
      href={`/rooms/${room.id}`}
    >
      <Card className="flex h-full flex-col transition-colors hover:border-primary/50">
        <CardHeader className="p-4 pb-3">
          <div className="flex items-start justify-between gap-2">
            <div className="min-w-0">
              <CardTitle className="truncate">{room.name}</CardTitle>
              {room.topic ? <p className="mt-1 truncate text-sm text-muted-foreground">{room.topic}</p> : null}
            </div>
            <Badge variant={statusTone(room.status)}>{statusLabel(room.status)}</Badge>
          </div>
        </CardHeader>

        <CardContent className="grid grid-cols-3 gap-2 p-4 pt-0 text-xs">
          <Stat label="消息" value={room.message_count} />
          <Stat label="成员" value={room.members_count} />
          <Stat label="Token" value={room.tokens_today} />
        </CardContent>

        <CardFooter className="mt-auto justify-between border-t p-4 text-xs text-muted-foreground">
          <span>{relativeUnix(room.last_message_at_unix)}</span>
          <span className="inline-flex items-center gap-1 text-primary">
            打开
            <ArrowUpRight className="h-3 w-3" />
          </span>
        </CardFooter>
      </Card>
    </a>
  )
}

function Stat({ label, value }: { label: string; value: number }) {
  return (
    <div className="rounded-md border bg-muted/30 px-2 py-1.5">
      <div className="text-[10px] text-muted-foreground">{label}</div>
      <div className="mt-0.5 text-sm font-semibold tabular-nums">{value}</div>
    </div>
  )
}
