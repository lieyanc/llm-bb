import type { Message, MessageKind, RoomStatus } from "../types"

const dateTimeFormatter = new Intl.DateTimeFormat("zh-CN", {
  year: "numeric",
  month: "2-digit",
  day: "2-digit",
  hour: "2-digit",
  minute: "2-digit",
  second: "2-digit",
})

export function formatDateTime(value: string) {
  if (!value) {
    return "-"
  }

  return dateTimeFormatter.format(new Date(value))
}

export function relativeUnix(value: number) {
  if (!value) {
    return "暂无活动"
  }

  const diff = Date.now() - value * 1000
  const minutes = Math.floor(diff / 60000)
  if (minutes < 1) {
    return "刚刚活跃"
  }
  if (minutes < 60) {
    return `${minutes} 分钟前`
  }
  const hours = Math.floor(minutes / 60)
  if (hours < 24) {
    return `${hours} 小时前`
  }
  return `${Math.floor(hours / 24)} 天前`
}

export function statusLabel(status: RoomStatus) {
  switch (status) {
    case "running":
      return "运行中"
    case "paused":
      return "已暂停"
    default:
      return "降频中"
  }
}

export function statusTone(status: RoomStatus) {
  switch (status) {
    case "running":
      return "success" as const
    case "paused":
      return "warning" as const
    default:
      return "destructive" as const
  }
}

export function initials(value: string) {
  const raw = value.trim()
  if (!raw) {
    return "?"
  }
  return Array.from(raw).slice(0, 2).join("")
}

export function messageKindLabel(kind: MessageKind) {
  switch (kind) {
    case "user":
      return "插话"
    case "system":
      return "事件"
    case "summary":
      return "摘要"
    default:
      return "台词"
  }
}

export function messageSpeaker(message: Message) {
  if (message.persona_name) {
    return message.persona_name
  }
  if (message.kind === "user") {
    return "观众"
  }
  return "系统"
}

export function maskKey(value: string) {
  const raw = value.trim()
  if (!raw) {
    return "未配置"
  }
  if (raw.length <= 8) {
    return "*".repeat(raw.length)
  }
  return `${raw.slice(0, 4)}${"*".repeat(raw.length - 8)}${raw.slice(-4)}`
}

export function countChars(value: string) {
  return Array.from(value).length
}
