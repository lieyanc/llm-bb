import {
  Cable,
  Flag,
  Gauge,
  GitBranch,
  Home,
  LockOpen,
  MessageSquareText,
  Settings,
  Theater,
  UsersRound,
} from "lucide-react"
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
      description="管理房间、角色、阵营、关系和模型接入"
      actions={
        <Button asChild variant="outline">
          <a href="/">
            <Home className="h-4 w-4" />
            返回
          </a>
        </Button>
      }
    >
      <MetricGrid>
        <MetricTile icon={<Theater className="h-4 w-4" />} label="房间" value={data.rooms.length} />
        <MetricTile icon={<Gauge className="h-4 w-4" />} label="运行中" value={data.runningRooms} />
        <MetricTile icon={<UsersRound className="h-4 w-4" />} label="角色" value={data.personas.length} />
        <MetricTile icon={<MessageSquareText className="h-4 w-4" />} label="累计消息" value={data.totalMessages} />
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
        <TabsList className="w-full justify-start overflow-x-auto rounded-lg">
          <TabsTrigger value="rooms">
            <Theater className="h-4 w-4" />
            房间 ({data.rooms.length})
          </TabsTrigger>
          <TabsTrigger value="personas">
            <UsersRound className="h-4 w-4" />
            角色 ({data.personas.length})
          </TabsTrigger>
          <TabsTrigger value="factions">
            <Flag className="h-4 w-4" />
            阵营 ({data.factions.length})
          </TabsTrigger>
          <TabsTrigger value="relationships">
            <GitBranch className="h-4 w-4" />
            关系 ({data.relationships?.length ?? 0})
          </TabsTrigger>
          <TabsTrigger value="providers">
            <Cable className="h-4 w-4" />
            接入 ({data.providers.length})
          </TabsTrigger>
          <TabsTrigger value="system">
            <Settings className="h-4 w-4" />
            系统
          </TabsTrigger>
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
