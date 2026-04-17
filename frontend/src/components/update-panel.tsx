import { AlertTriangle, CheckCircle2, Download, Loader2, RefreshCw, Rocket } from "lucide-react"
import { useEffect, useState } from "react"
import { fetchJSON, postJSON } from "../lib/api"
import type { UpdateChannel, UpdateCheckResult, VersionInfo } from "../types"
import { Alert, AlertDescription, AlertTitle } from "./ui/alert"
import { Badge } from "./ui/badge"
import { Button } from "./ui/button"
import { Label } from "./ui/label"

type Phase = "idle" | "checked" | "applied" | "restarting" | "restarted"

const selectClassName =
  "flex h-9 w-full rounded-lg border border-border bg-card px-3 py-1.5 text-sm text-foreground transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/30 focus-visible:border-primary"

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
    } catch (err) {
      // Ignore: connection will drop mid-restart, that's expected.
    }

    const before = current
    const deadline = Date.now() + 60_000
    const tick = async () => {
      if (Date.now() > deadline) {
        setError("等待服务重启超时，请手动刷新页面")
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
        // still down, keep polling
      }
      setTimeout(tick, 1000)
    }
    setTimeout(tick, 1500)
  }

  return (
    <section className="grid gap-4 xl:grid-cols-[minmax(0,1fr)_minmax(0,1fr)]">
      <div className="rounded-xl border border-border bg-card p-5 space-y-4">
        <header className="space-y-1">
          <p className="text-xs uppercase tracking-[0.2em] text-muted-foreground">Current Build</p>
          <h3 className="text-lg font-semibold">当前版本</h3>
        </header>
        {current ? (
          <dl className="grid grid-cols-[auto_1fr] gap-x-4 gap-y-2 text-sm">
            <dt className="text-muted-foreground">版本</dt>
            <dd className="font-mono">
              {current.version}
              <Badge className="ml-2" variant="outline">
                {current.channel}
              </Badge>
            </dd>
            <dt className="text-muted-foreground">Commit</dt>
            <dd className="font-mono text-xs">{shortCommit(current.commit)}</dd>
            <dt className="text-muted-foreground">构建时间</dt>
            <dd className="font-mono text-xs">{current.buildDate}</dd>
            <dt className="text-muted-foreground">平台</dt>
            <dd className="font-mono text-xs">{current.goos}/{current.goarch}</dd>
          </dl>
        ) : (
          <p className="text-sm text-muted-foreground">加载中…</p>
        )}
        {!injected && current ? (
          <Alert variant="warning">
            <AlertTriangle className="h-4 w-4" />
            <AlertTitle>本地开发构建</AlertTitle>
            <AlertDescription>
              未通过 CI 构建，缺少版本元信息。OTA 升级会把二进制替换成 GitHub Release 里的 CI 构建产物。
            </AlertDescription>
          </Alert>
        ) : null}
      </div>

      <div className="rounded-xl border border-border bg-card p-5 space-y-4">
        <header className="space-y-1">
          <p className="text-xs uppercase tracking-[0.2em] text-muted-foreground">OTA Update</p>
          <h3 className="text-lg font-semibold">检查并安装更新</h3>
        </header>

        <div className="space-y-2">
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
            <option value="stable">stable - 仅语义化 tag (v*)</option>
            <option value="dev">dev - master 每次推送自动更新</option>
          </select>
        </div>

        <div className="flex flex-wrap gap-2">
          <Button onClick={runCheck} disabled={busy !== null} variant="outline">
            {busy === "check" ? <Loader2 className="h-4 w-4 animate-spin" /> : <RefreshCw className="h-4 w-4" />}
            检查更新
          </Button>
          {check && check.updateAvailable && phase !== "applied" && phase !== "restarting" && phase !== "restarted" ? (
            <Button onClick={runApply} disabled={busy !== null}>
              {busy === "apply" ? <Loader2 className="h-4 w-4 animate-spin" /> : <Download className="h-4 w-4" />}
              立即更新
            </Button>
          ) : null}
          {phase === "applied" ? (
            <Button onClick={runRestart} disabled={busy !== null} variant="destructive">
              {busy === "restart" ? <Loader2 className="h-4 w-4 animate-spin" /> : <Rocket className="h-4 w-4" />}
              重启服务
            </Button>
          ) : null}
        </div>

        {error ? (
          <Alert variant="error">
            <AlertTitle>出错了</AlertTitle>
            <AlertDescription>{error}</AlertDescription>
          </Alert>
        ) : null}

        {check ? (
          <div className="rounded-lg border border-border bg-muted/30 p-4 space-y-2 text-sm">
            <div className="flex flex-wrap items-center gap-2">
              <span className="font-mono">{check.latestTag}</span>
              <Badge variant={check.updateAvailable ? "default" : "outline"}>
                {check.updateAvailable ? "有更新" : "已是最新"}
              </Badge>
            </div>
            {check.publishedAt ? (
              <p className="text-xs text-muted-foreground">发布于 {new Date(check.publishedAt).toLocaleString()}</p>
            ) : null}
            <dl className="grid grid-cols-[auto_1fr] gap-x-3 gap-y-1 text-xs">
              <dt className="text-muted-foreground">Asset</dt>
              <dd className="font-mono break-all">{check.assetName}</dd>
              <dt className="text-muted-foreground">大小</dt>
              <dd className="font-mono">{formatBytes(check.assetSize)}</dd>
              {check.latestCommit ? (
                <>
                  <dt className="text-muted-foreground">Commit</dt>
                  <dd className="font-mono">{shortCommit(check.latestCommit)}</dd>
                </>
              ) : null}
            </dl>
            {check.notes ? (
              <pre className="mt-2 max-h-48 overflow-auto rounded bg-background/60 p-2 text-xs whitespace-pre-wrap break-words">
                {check.notes}
              </pre>
            ) : null}
          </div>
        ) : null}

        {phase === "applied" ? (
          <Alert variant="success">
            <CheckCircle2 className="h-4 w-4" />
            <AlertTitle>二进制已替换</AlertTitle>
            <AlertDescription>点击"重启服务"以加载新版本。SSE 客户端会短暂断开自动重连。</AlertDescription>
          </Alert>
        ) : null}

        {phase === "restarting" ? (
          <Alert>
            <Loader2 className="h-4 w-4 animate-spin" />
            <AlertTitle>正在重启</AlertTitle>
            <AlertDescription>等待服务回到在线状态…</AlertDescription>
          </Alert>
        ) : null}

        {phase === "restarted" ? (
          <Alert variant="success">
            <CheckCircle2 className="h-4 w-4" />
            <AlertTitle>升级完成</AlertTitle>
            <AlertDescription>当前已运行 {current?.version} ({shortCommit(current?.commit ?? "")})。</AlertDescription>
          </Alert>
        ) : null}
      </div>
    </section>
  )
}
