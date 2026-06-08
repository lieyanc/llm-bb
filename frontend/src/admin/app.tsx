import { LockOpen } from "lucide-react"
import { MetricGrid, MetricTile, Shell } from "../shared/shell"
import type { AdminPageData } from "../shared/types"
import { Alert, AlertDescription, AlertTitle } from "../shared/ui/alert"
import { Button } from "../shared/ui/button"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "../shared/ui/tabs"
import { FactionsSection } from "./sections/factions"
import { PersonasSection } from "./sections/personas"
import { ProvidersSection } from "./sections/providers"
import { RelationshipsSection } from "./sections/relationships"
import { RoomsSection } from "./sections/rooms"
import { SystemSection } from "./sections/system"
import { useAdminActions } from "./use-admin-actions"

export function AdminApp({ data }: { data: AdminPageData }) {
  const actions = useAdminActions()
  const notice = actions.notice

  return (
    <Shell
      title="导演台"
      actions={
        <Button asChild variant="outline">
          <a href="/">返回</a>
        </Button>
      }
    >
      <MetricGrid>
        <MetricTile label="房间" value={data.rooms.length} />
        <MetricTile label="运行中" value={data.runningRooms} />
        <MetricTile label="角色" value={data.personas.length} />
        <MetricTile label="累计消息" value={data.totalMessages} />
      </MetricGrid>

      {data.adminOpen ? (
        <Alert variant="warning">
          <AlertTitle className="flex items-center gap-2">
            <LockOpen className="h-4 w-4" />
            未设置管理员口令
          </AlertTitle>
        </Alert>
      ) : null}

      {notice ? (
        <Alert
          variant={
            notice.tone === "success" ? "success" : notice.tone === "warning" ? "warning" : "error"
          }
        >
          <AlertTitle>{notice.title}</AlertTitle>
          <AlertDescription>{notice.message}</AlertDescription>
        </Alert>
      ) : null}

      <Tabs defaultValue="rooms">
        <TabsList className="w-full justify-start">
          <TabsTrigger value="rooms">房间 ({data.rooms.length})</TabsTrigger>
          <TabsTrigger value="personas">角色 ({data.personas.length})</TabsTrigger>
          <TabsTrigger value="factions">阵营 ({data.factions.length})</TabsTrigger>
          <TabsTrigger value="relationships">关系 ({data.relationships?.length ?? 0})</TabsTrigger>
          <TabsTrigger value="providers">接入 ({data.providers.length})</TabsTrigger>
          <TabsTrigger value="system">系统</TabsTrigger>
        </TabsList>

        <TabsContent value="rooms">
          <RoomsSection
            rooms={data.rooms}
            personas={data.personas}
            roomMembers={data.roomMembers}
            actions={actions}
          />
        </TabsContent>
        <TabsContent value="personas">
          <PersonasSection
            personas={data.personas}
            factions={data.factions}
            providers={data.providers}
            actions={actions}
          />
        </TabsContent>
        <TabsContent value="factions">
          <FactionsSection factions={data.factions} actions={actions} />
        </TabsContent>
        <TabsContent value="relationships">
          <RelationshipsSection
            relationships={data.relationships ?? []}
            personas={data.personas}
            actions={actions}
          />
        </TabsContent>
        <TabsContent value="providers">
          <ProvidersSection providers={data.providers} actions={actions} />
        </TabsContent>
        <TabsContent value="system">
          <SystemSection />
        </TabsContent>
      </Tabs>
    </Shell>
  )
}
