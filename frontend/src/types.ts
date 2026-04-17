export type RoomStatus = "running" | "paused" | "degraded"
export type MessageKind = "chat" | "user" | "system" | "summary"

export interface ProviderConfig {
  id: number
  name: string
  base_url: string
  api_key: string
  default_model: string
  timeout_ms: number
  enabled: boolean
  created_at: string
  updated_at: string
}

export interface Persona {
  id: number
  name: string
  avatar: string
  public_identity: string
  speaking_style: string
  stance: string
  goal: string
  taboo: string
  aggression: number
  activity_level: number
  faction_id: number
  provider_config_id: number
  model_name: string
  temperature: number
  max_tokens: number
  cooldown_seconds: number
  enabled: boolean
  created_at: string
  updated_at: string
}

export interface Faction {
  id: number
  name: string
  description: string
  shared_values: string
  shared_style: string
  default_bias: string
  created_at: string
  updated_at: string
}

export interface Room {
  id: number
  name: string
  topic: string
  description: string
  status: RoomStatus
  heat: number
  conflict_level: number
  tick_min_seconds: number
  tick_max_seconds: number
  daily_token_budget: number
  summary_trigger_count: number
  message_retention_count: number
  created_at: string
  updated_at: string
}

export interface RoomOverview extends Room {
  message_count: number
  tokens_today: number
  members_count: number
  last_message_at_unix: number
}

export interface RoomMemberView {
  id: number
  room_id: number
  persona_id: number
  role_weight: number
  can_initiate: boolean
  can_reply: boolean
  persona_name: string
  avatar: string
  public_identity: string
  speaking_style: string
  stance: string
  goal: string
  taboo: string
  aggression: number
  activity_level: number
  model_name: string
  temperature: number
  max_tokens: number
  cooldown_seconds: number
  persona_enabled: boolean
  faction_id: number
  faction_name: string
  faction_description: string
  provider_config_id: number
  provider_name: string
  provider_base_url: string
  provider_api_key: string
  provider_model: string
  provider_timeout_ms: number
  provider_enabled: boolean
}

export interface Message {
  id: number
  room_id: number
  persona_id: number
  persona_name: string
  persona_avatar: string
  kind: MessageKind
  content: string
  reply_to_message_id: number
  source: string
  prompt_tokens: number
  completion_tokens: number
  created_at: string
}

export interface Summary {
  id: number
  room_id: number
  from_message_id: number
  to_message_id: number
  content: string
  created_at: string
}

export interface Relationship {
  id: number
  source_persona_id: number
  target_persona_id: number
  affinity: number
  hostility: number
  respect: number
  focus_weight: number
  notes: string
  updated_at: string
}

export interface HomePageData {
  rooms: RoomOverview[]
  totalRooms: number
  runningRooms: number
  totalMessages: number
  totalTokens: number
}

export interface RoomPageData {
  room: Room
  members: RoomMemberView[]
  messages: Message[]
  latestSummary: Summary | null
  tokensToday: number
  messageCount: number
  memberCount: number
}

export interface AdminPageData {
  rooms: RoomOverview[]
  personas: Persona[]
  factions: Faction[]
  providers: ProviderConfig[]
  relationships: Relationship[]
  adminOpen: boolean
  roomMembers: Record<string, RoomMemberView[]>
  runningRooms: number
  totalMessages: number
  totalTokens: number
}

export type Bootstrap =
  | { page: "home"; title: string; data: HomePageData }
  | { page: "room"; title: string; data: RoomPageData }
  | { page: "admin"; title: string; data: AdminPageData }
