import { AlertTriangle, CheckCircle2, Download, Loader2, RefreshCw, Rocket } from "lucide-react"
import { useEffect, useState } from "react"
import { fetchJSON, postJSON } from "../shared/lib/api"
import type { UpdateChannel, UpdateCheckResult, VersionInfo } from "../shared/types"
import { Alert, AlertDescription, AlertTitle } from "../shared/ui/alert"
import { Badge } from "../shared/ui/badge"
import { Button } from "../shared/ui/button"
import { Card, CardAction, CardContent, CardDescription, CardHeader, CardTitle } from "../shared/ui/card"
import { Label } from "../shared/ui/label"
import { selectClassName } from "./ui"

type Phase = "idle" | "checked" | "applied" | "restarting" | "restarted"

function formatBytes(n: number): string {
  if (!n) return "0 B"
  const units = ["B", "KiB", "MiB", "GiB"]
  let i = 0
  let v = n
  while (v >= 1024 && i < units.length - 1) {
    v /= 1024
    i++
  }
  return `${v.toFixed(i === 0 ? 0 : 1)} ${units[i]}`
}

function shortCommit(c: string): string {
  if (!c || c === "unknown") return c || "-"
  return c.length > 7 ? c.slice(0, 7) : c
}

export function UpdatePanel() {
  const [current, setCurrent] = useState<VersionInfo | null>(null)
  const [channel, setChannel] = useState<UpdateChannel>("stable")
  const [check, setCheck] = useState<UpdateCheckResult | null>(null)
  const [phase, setPhase] = useState<Phase>("idle")
  const [busy, setBusy] = useState<"check" | "apply" | "restart" | null>(null)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    fetchJSON<{ version: VersionInfo }>("/api/admin/version")
      .then((r) => {
        setCurrent(r.version)
        if (r.version.channel === "dev" || r.version.channel === "stable") {
          setChannel(r.version.channel as UpdateChannel)
        }
      })
      .catch((err) => setError(err.message || "读取版本失败"))
  }, [])

  const injected = current ? current.commit !== "unknown" : false

  async function runCheck() {
    setBusy("check")
    setError(null)
    setCheck(null)
    try {
      const r = await fetchJSON<{ result: UpdateCheckResult }>(`/api/admin/update/check?channel=${channel}`)
      setCheck(r.result)
      setPhase("checked")
    } catch (err) {
      setError(err instanceof Error ? err.message : "检查失败")
    } finally {
      setBusy(null)
    }
  }

  async function runApply() {
    if (!check) return
    setBusy("apply")
    setError(null)
    try {
      await postJSON("/api/admin/update/apply", { channel })
      setPhase("applied")
    } catch (err) {
      setError(err instanceof Error ? err.message : "升级失败")
    } finally {
      setBusy(null)
    }
  }

  async function runRestart() {
    if (!current) return
    setBusy("restart")
    setError(null)
    setPhase("restarting")
    try {
      await postJSON("/api/admin/update/restart", {})
    } catch {
      /* connection drops mid-restart, expected */
    }

    const before = current
    const deadline = Date.now() + 60_000
    const tick = async () => {
      if (Date.now() > deadline) {
        setError("重启超时,请手动刷新")
        setBusy(null)
        setPhase("applied")
        return
      }
      try {
        const r = await fetchJSON<{ version: VersionInfo }>("/api/admin/version")
        if (r.version.commit !== before.commit || r.version.buildDate !== before.buildDate) {
          setCurrent(r.version)
          setPhase("restarted")
          setBusy(null)
          return
        }
      } catch {
        /* still down, keep polling */
      }
      setTimeout(tick, 1000)
    }
    setTimeout(tick, 1500)
  }

  return (
    <section className="grid gap-4 xl:grid-cols-2">
      <Card className="overflow-hidden">
        <CardHeader className="p-4 pb-3">
          <CardTitle className="text-sm">当前版本</CardTitle>
          <CardDescription>正在运行的二进制信息</CardDescription>
          <CardAction>
            {current ? <Badge variant="outline">{current.channel}</Badge> : null}
          </CardAction>
        </CardHeader>
        <CardContent className="space-y-3 p-4 pt-0">
          {current ? (
            <dl className="grid grid-cols-[auto_1fr] gap-x-4 gap-y-1.5 text-sm">
              <dt className="text-muted-foreground">版本</dt>
              <dd className="font-mono">{current.version}</dd>
              <dt className="text-muted-foreground">Commit</dt>
              <dd className="font-mono text-xs">{shortCommit(current.commit)}</dd>
              <dt className="text-muted-foreground">构建</dt>
              <dd className="font-mono text-xs">{current.buildDate}</dd>
              <dt className="text-muted-foreground">平台</dt>
              <dd className="font-mono text-xs">
                {current.goos}/{current.goarch}
              </dd>
            </dl>
          ) : (
            <p className="text-sm text-muted-foreground">加载中</p>
          )}
          {!injected && current ? (
            <Alert variant="warning">
              <AlertTriangle className="h-4 w-4" />
              <AlertTitle>本地构建</AlertTitle>
            </Alert>
          ) : null}
        </CardContent>
      </Card>

      <Card className="overflow-hidden">
        <CardHeader className="p-4 pb-3">
          <CardTitle className="text-sm">更新</CardTitle>
          <CardDescription>检查、替换并重启到新版本</CardDescription>
        </CardHeader>
        <CardContent className="space-y-3 p-4 pt-0">

        <div className="space-y-1.5">
          <Label htmlFor="update-channel">通道</Label>
          <select
            id="update-channel"
            className={selectClassName}
            value={channel}
            onChange={(e) => {
              setChannel(e.target.value as UpdateChannel)
              setCheck(null)
              setPhase("idle")
            }}
            disabled={busy !== null}
          >
            <option value="stable">stable</option>
            <option value="dev">dev</option>
          </select>
        </div>

        <div className="flex flex-wrap gap-2">
          <Button onClick={runCheck} disabled={busy !== null} variant="outline">
            {busy === "check" ? <Loader2 className="h-4 w-4 animate-spin" /> : <RefreshCw className="h-4 w-4" />}
            检查
          </Button>
          {check && check.updateAvailable && phase !== "applied" && phase !== "restarting" && phase !== "restarted" ? (
            <Button onClick={runApply} disabled={busy !== null}>
              {busy === "apply" ? <Loader2 className="h-4 w-4 animate-spin" /> : <Download className="h-4 w-4" />}
              更新
            </Button>
          ) : null}
          {phase === "applied" ? (
            <Button onClick={runRestart} disabled={busy !== null} variant="destructive">
              {busy === "restart" ? <Loader2 className="h-4 w-4 animate-spin" /> : <Rocket className="h-4 w-4" />}
              重启
            </Button>
          ) : null}
        </div>

        {error ? (
          <Alert variant="error">
            <AlertTitle>错误</AlertTitle>
            <AlertDescription>{error}</AlertDescription>
          </Alert>
        ) : null}

        {check ? (
          <div className="space-y-3 rounded-md border bg-muted/25 p-3 text-sm">
            <div className="flex flex-wrap items-center gap-2">
              <span className="font-mono">{check.latestVersion || check.latestTag}</span>
              <Badge variant={check.updateAvailable ? "default" : "outline"}>
                {check.updateAvailable ? "有更新" : "已最新"}
              </Badge>
            </div>
            {check.publishedAt ? (
              <p className="text-xs text-muted-foreground">{new Date(check.publishedAt).toLocaleString()}</p>
            ) : null}
            <dl className="grid grid-cols-[auto_1fr] gap-x-3 gap-y-1 text-xs">
              <dt className="text-muted-foreground">Tag</dt>
              <dd className="font-mono">{check.latestTag}</dd>
              <dt className="text-muted-foreground">Asset</dt>
              <dd className="break-all font-mono">{check.assetName}</dd>
              <dt className="text-muted-foreground">大小</dt>
              <dd className="font-mono">{formatBytes(check.assetSize)}</dd>
              {check.latestBuildDate ? (
                <>
                  <dt className="text-muted-foreground">构建</dt>
                  <dd className="font-mono">{check.latestBuildDate}</dd>
                </>
              ) : null}
              {check.latestCommit ? (
                <>
                  <dt className="text-muted-foreground">Commit</dt>
                  <dd className="font-mono">{shortCommit(check.latestCommit)}</dd>
                </>
              ) : null}
            </dl>
            {check.notes ? (
              <pre className="max-h-48 overflow-auto whitespace-pre-wrap break-words rounded-md border bg-background p-2 text-xs">
                {check.notes}
              </pre>
            ) : null}
          </div>
        ) : null}

        {phase === "applied" ? (
          <Alert variant="success">
            <CheckCircle2 className="h-4 w-4" />
            <AlertTitle>已替换</AlertTitle>
          </Alert>
        ) : null}

        {phase === "restarting" ? (
          <Alert>
            <Loader2 className="h-4 w-4 animate-spin" />
            <AlertTitle>重启中…</AlertTitle>
          </Alert>
        ) : null}

        {phase === "restarted" ? (
          <Alert variant="success">
            <CheckCircle2 className="h-4 w-4" />
            <AlertTitle>完成</AlertTitle>
            <AlertDescription>
              当前 {current?.version} ({shortCommit(current?.commit ?? "")})
            </AlertDescription>
          </Alert>
        ) : null}
        </CardContent>
      </Card>
    </section>
  )
}
