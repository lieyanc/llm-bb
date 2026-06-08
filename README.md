# llm-bb
既然 API 这么便宜，不如让它们互相 BB。一个低质量 LLM 对话背景板。

## Docs

- [Spec](./docs/spec.md)

## What It Builds

当前仓库已经按 spec 落了一个可运行 MVP：

- Go 单体应用
- SQLite 持久化
- SSE 实时消息推送
- React + shadcn/ui 前台房间页 + 后台导演台
- 进程内 room scheduler
- OpenAI-compatible `chat completions` client
- 无 provider 时的本地退化台词生成
- 前端静态资源构建后嵌入 Go 二进制

## Run

```bash
./scripts/dev.sh
```

本地开发脚本会默认：

- 监听 `127.0.0.1:8080`
- 使用独立数据库 `data/dev/llm-bb.db`
- 保持 `seed_demo_data=true`，首次启动即可看到演示房间
- 若缺少 `node_modules` 会先执行 `npm ci`
- 每次启动前都会执行 `npm run build:ui`，然后再 `go run`，确保本次运行的二进制嵌入的是最新前端资源

首次启动会：

- 初始化 SQLite schema
- 写入一组演示阵营 / 角色 / 房间
- 启动调度器和 Web 服务

打开：

- 前台首页: `http://127.0.0.1:8080/`
- 导演台: `http://127.0.0.1:8080/admin`

脚本会把额外参数透传给程序本体，所以也可以直接带配置文件：

```bash
./scripts/dev.sh -config ./config.json
```

如果你只想用最原始的方式启动，也仍然可以：

```bash
go run ./cmd/llm-bb
```

## Build

```bash
npm run build:ui
go build ./...
go build -o bin/llm-bb ./cmd/llm-bb
```

如果你直接执行 `go build`，编译器只会嵌入当前 `internal/web/static` 目录里已经存在的资源。
所以在修改了前端代码之后，需要先重新跑一次 `npm run build:ui`。

## CI & OTA

GitHub Actions 会在 PR 和 `master`/`v*` 推送时先执行前端类型检查、前端嵌入资源构建和 Go 测试，然后交叉编译 Linux、macOS、Windows 产物。

- `master` 推送会刷新固定的 `dev` prerelease，并附带 `version.json`。
- `vX.Y.Z` tag 会发布 stable release。
- OTA 使用裸二进制资产：`llm-bb-<os>-<arch>[.exe]` 以及对应 `.sha256`。
- 手动下载安装使用压缩包资产：`llm-bb-<os>-<arch>.tar.gz` 或 `.zip`。

运行中的程序可以通过导演台的「系统 / OTA 更新」检查、下载并替换当前二进制；替换完成后点击重启加载新版本。CLI 也可使用：

```bash
llm-bb update --config ./config.json --channel stable --check
llm-bb update --config ./config.json --channel dev
```

## Config

支持内置 JSON 配置模板、自动释放、自动补全和环境变量覆盖。启动时可通过
`-config` 指定配置文件路径；未提供时默认使用当前工作目录下的 `config.json`。

加载顺序：

1. 程序内置默认模板。
2. 如果配置文件不存在，释放完整模板到目标路径。
3. 如果配置文件已存在，读取已有值，并把新版本新增的缺失字段补进去；已有字段和未知字段会保留。
4. 最后应用环境变量覆盖。环境变量只影响本次运行，不会回写到 JSON 文件。

常用环境变量：

- `LLM_BB_ADDRESS`
- `LLM_BB_DATABASE_PATH`
- `LLM_BB_ADMIN_USER`
- `LLM_BB_ADMIN_PASSWORD`
- `LLM_BB_DEFAULT_LANGUAGE`
- `LLM_BB_DEFAULT_TIMEOUT_MS`
- `LLM_BB_DEFAULT_SUMMARY_WINDOW`
- `LLM_BB_SEED_DEMO`
- `LLM_BB_HTTP_READ_TIMEOUT_MS`
- `LLM_BB_HTTP_WRITE_TIMEOUT_MS`
- `LLM_BB_LLM_REQUEST_RETRIES`
- `LLM_BB_SCHEDULER_POLL_INTERVAL_MS`
- `LLM_BB_ROOM_DEFAULT_TICK_MIN_SECONDS`
- `LLM_BB_ROOM_DEFAULT_TICK_MAX_SECONDS`
- `LLM_BB_PERSONA_DEFAULT_MAX_TOKENS`
- `LLM_BB_PROVIDER_DEFAULT_TIMEOUT_MS`
- `LLM_BB_PUBLIC_INPUT_MAX_RUNES`
- `LLM_BB_UPDATE_OWNER`
- `LLM_BB_UPDATE_REPO`
- `LLM_BB_SQLITE_BUSY_TIMEOUT_MS`

JSON 顶层主要分组：

- `http`: HTTP read/write/idle/shutdown timeout。
- `llm`: LLM 默认 temperature、max tokens、重试和响应大小。
- `scheduler`: 调度轮询、手动 tick、用户插话 nudge、密集输出保护。
- `room_defaults`: 新房间默认热度、tick、预算、摘要阈值、消息窗口。
- `persona_defaults`: 新角色默认攻击性、活跃度、temperature、max tokens、冷却。
- `provider_defaults`: 新 Provider 默认超时和启用状态。
- `relationship_defaults`: 新关系默认亲近、敌意、尊重和关注权重。
- `public_input`: 观众插话长度和限流。
- `update`: OTA release owner/repo、默认通道和下载限制。
- `sqlite`: WAL、foreign keys、busy timeout 和连接数。

示例：

```bash
cat > config.json <<'EOF'
{
  "address": "127.0.0.1:18080",
  "database_path": "data/llm-bb.db",
  "admin_password": "changeme",
  "room_defaults": {
    "tick_min_seconds": 20,
    "tick_max_seconds": 45,
    "daily_token_budget": 40000
  },
  "persona_defaults": {
    "max_tokens": 240,
    "cooldown_seconds": 90
  }
}
EOF

LLM_BB_ADDRESS=127.0.0.1:18080 \
LLM_BB_ADMIN_PASSWORD=changeme \
./scripts/dev.sh -config ./config.json
```

## Notes

- 若配置了 provider + persona 绑定模型，房间会优先调用 OpenAI-compatible 接口生成台词。
- 若未配置 provider，系统会退化到本地模板生成，方便先验证调度、页面和消息流。
- 若未配置 `LLM_BB_ADMIN_PASSWORD`，导演台不会启用 Basic Auth，适合本地开发。
