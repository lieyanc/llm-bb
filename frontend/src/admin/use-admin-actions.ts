import { useCallback, useState } from "react"
import { deleteJSON } from "../shared/lib/api"

export type NoticeTone = "success" | "error" | "warning"
export interface Notice {
  tone: NoticeTone
  title: string
  message: string
}

export interface AdminActions {
  notice: Notice | null
  setNotice: (next: Notice | null) => void
  busyAction: string | null
  runAction: (key: string, label: string, fn: () => Promise<unknown>) => Promise<void>
  handleDelete: (entityType: string, id: number, label: string) => Promise<void>
}

export function useAdminActions(): AdminActions {
  const [notice, setNotice] = useState<Notice | null>(null)
  const [busyAction, setBusyAction] = useState<string | null>(null)

  const runAction = useCallback(async (key: string, label: string, fn: () => Promise<unknown>) => {
    setBusyAction(key)
    try {
      await fn()
      setNotice({ tone: "success", title: label, message: "即将刷新。" })
      window.setTimeout(() => window.location.reload(), 500)
    } catch (error) {
      setNotice({
        tone: "error",
        title: `${label}失败`,
        message: error instanceof Error ? error.message : "请求失败",
      })
    } finally {
      setBusyAction(null)
    }
  }, [])

  const handleDelete = useCallback(
    async (entityType: string, id: number, label: string) => {
      if (!window.confirm(`删除「${label}」？不可撤销。`)) return
      await runAction(`delete-${entityType}-${id}`, "删除", () =>
        deleteJSON(`/api/admin/${entityType}/${id}`),
      )
    },
    [runAction],
  )

  return { notice, setNotice, busyAction, runAction, handleDelete }
}
