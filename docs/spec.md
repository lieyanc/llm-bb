# llm-bb Spec

## 1. 项目定位

`llm-bb` 是一个网页上的 LLM 群聊/吵架背景板。系统允许为不同角色绑定不同模型、身份、阵营和关系，并在房间内持续产出对话。用户既可以旁观，也可以随时插话、点名、拱火，推动群聊朝新的方向发展。

本项目的核心不是“做一个普通聊天网站”，而是“做一个可编排的多角色 LLM 群聊引擎，并提供一个适合长期观看的网页前台”。

## 2. 设计约束

本项目当前明确采用以下约束：

- LLM 接入层只需要兼容 OpenAI API 形态的 HTTP 接口，用于对接用户自己的 `new-api`。
- 不依赖官方或第三方复杂 SDK，优先使用手写的轻量 HTTP client。
- 数据库和存储尽量轻，不引入 PostgreSQL、Redis、对象存储等重型依赖。
- 部署应尽可能便捷，目标形态为单二进制启动。
- 单机长期运行优先于分布式扩展。
- MVP 默认面向个人或小范围自用，不以多租户 SaaS 为第一目标。

基于以上约束，第一阶段技术方案固定为：

- 后端语言：`Go`
- 存储：`SQLite`
- 实时推送：`SSE`
- 前端：服务端渲染模板 + 少量原生 JS
- 打包：单二进制，静态资源嵌入可执行文件

## 3. 目标

### 3.1 产品目标

- 支持创建一个或多个“房间”，每个房间是一个持续运转的群聊场景。
- 支持为角色指定身份、说话风格、立场、目标、攻击性和阵营。
- 支持为不同角色绑定不同模型和不同 OpenAI-compatible endpoint 配置。
- 支持系统 24h 持续输出，但具备节奏控制、成本控制和退化防护能力。
- 支持用户在网页中实时观看消息流，并通过输入或点名参与互动。
- 支持最低限度的后台管理能力，便于修改角色、房间和运行参数。

### 3.2 工程目标

- 使用最少的外部依赖完成可用系统。
- 保证单进程可运行、可恢复、可观测。
- 让部署复杂度接近“复制二进制 + 填配置 + 启动”。
- 让核心逻辑尽量可测试、可替换、可扩展。

## 4. 非目标

第一阶段不以以下内容为目标：

- 不做复杂工作流编排系统。
- 不做高度可扩展的分布式集群架构。
- 不做完整权限系统和企业级管理后台。
- 不追求高度拟真的长期世界模拟。
- 不把所有决策都交给 LLM，自由生成不是目标，稳定运行才是目标。

## 5. 核心产品形态

系统分为两个界面：

- 前台“背景板”：展示角色持续聊天、吵架、阴阳怪气、结盟或对喷。
- 后台“导演台”：配置房间、角色、阵营、模型、节奏和预算。

前台偏“观看体验”，后台偏“编排能力”。

### 5.1 前台能力

- 实时展示房间消息流。
- 显示角色卡片、阵营和当前状态。
- 显示房间主题、热度、冲突值、运行状态。
- 用户可发送插话消息。
- 用户可 `@角色`，提升该角色被调度和回应的概率。
- 支持暂停/恢复房间。

### 5.2 后台能力

- 创建和编辑房间。
- 创建和编辑角色。
- 配置角色所属阵营。
- 配置角色间关系。
- 配置模型 endpoint、API Key、模型名和默认参数。
- 配置房间调度频率、预算、摘要策略和降级策略。

## 6. 总体架构

系统采用单体架构，一个 Go 进程同时承载以下模块：

- `web`：网页、管理页、REST API、SSE 推送
- `store`：SQLite 存储访问层
- `llm`：OpenAI-compatible API client
- `engine`：多角色对话引擎
- `scheduler`：进程内房间调度器

### 6.1 模块职责

#### web

- 提供前台页面和基础管理页
- 暴露 JSON API
- 提供 SSE 事件流给前端
- 处理用户插话和管理操作

#### store

- 管理 SQLite 连接
- 负责 schema 初始化和迁移
- 提供对房间、角色、消息、摘要、关系等实体的 CRUD 能力

#### llm

- 对接 OpenAI-compatible 接口
- 提供统一的 `chat completion` 调用
- 统一错误格式、重试和超时处理
- 未来可扩展到流式输出，但 MVP 不强依赖

#### engine

- 根据房间状态判断是否该产生下一条消息
- 选择当前最适合发言的角色
- 生成当前轮次的发言意图
- 构造 prompt
- 调用模型生成文本
- 对生成结果做基本清洗和去重

#### scheduler

- 在进程内维护各房间的推进节奏
- 触发 engine 产出消息
- 在低活跃时段自动降频
- 定期触发摘要与压缩任务

## 7. 核心原则

### 7.1 程序控节奏，模型写台词

系统不采用“让多个 LLM 自由无限对话”的方式。核心决策由程序控制，LLM 只负责具体措辞。

程序负责：

- 这轮是否应该发言
- 应该由谁发言
- 该角色在对谁说
- 当前发言的意图是什么
- 需要带入哪些上下文

模型负责：

- 把该角色这一轮的意图写成自然的消息文本

这样做的收益：

- 避免无限重复和空转
- 更容易控制成本
- 更容易塑造角色差异
- 更容易实现长时间稳定运行

### 7.2 角色是行为模型，不是单条 prompt

角色配置不应仅由一条 system prompt 构成，而应拆成结构化字段，使程序能基于字段做调度和推理。

### 7.3 长期运行优先于极致智能

本项目优先保证：

- 能跑一整天
- 输出不明显重复
- 房间节奏可控
- 成本可控

而不是追求单轮回复的极限效果。

## 8. 领域模型

### 8.1 ProviderConfig

表示一个可用的 OpenAI-compatible 模型接入配置。

建议字段：

- `id`
- `name`
- `base_url`
- `api_key`
- `default_model`
- `timeout_ms`
- `enabled`
- `created_at`
- `updated_at`

### 8.2 Persona

表示一个可参与对话的角色。

建议字段：

- `id`
- `name`
- `avatar`
- `public_identity`
- `speaking_style`
- `stance`
- `goal`
- `taboo`
- `aggression`
- `activity_level`
- `faction_id`
- `provider_config_id`
- `model_name`
- `temperature`
- `max_tokens`
- `cooldown_seconds`
- `enabled`
- `created_at`
- `updated_at`

### 8.3 Faction

表示角色所属群体或阵营。

建议字段：

- `id`
- `name`
- `description`
- `shared_values`
- `shared_style`
- `default_bias`
- `created_at`
- `updated_at`

### 8.4 Relationship

表示角色间关系，直接影响接话和攻击概率。

建议字段：

- `id`
- `source_persona_id`
- `target_persona_id`
- `affinity`
- `hostility`
- `respect`
- `focus_weight`
- `notes`
- `updated_at`

### 8.5 Room

表示一个持续运转的聊天空间。

建议字段：

- `id`
- `name`
- `topic`
- `description`
- `status`
- `heat`
- `conflict_level`
- `tick_min_seconds`
- `tick_max_seconds`
- `daily_token_budget`
- `summary_trigger_count`
- `message_retention_count`
- `created_at`
- `updated_at`

### 8.6 RoomMember

表示房间和角色的关联关系。

建议字段：

- `id`
- `room_id`
- `persona_id`
- `role_weight`
- `can_initiate`
- `can_reply`

### 8.7 Message

表示房间中的一条发言或系统事件。

建议字段：

- `id`
- `room_id`
- `persona_id`
- `kind`
- `content`
- `reply_to_message_id`
- `source`
- `prompt_tokens`
- `completion_tokens`
- `created_at`

其中 `kind` 可包含：

- `chat`
- `user`
- `system`
- `summary`

其中 `source` 可包含：

- `scheduler`
- `user`
- `manual`

### 8.8 Summary

表示某个房间某一段历史的摘要，用于替代长上下文。

建议字段：

- `id`
- `room_id`
- `from_message_id`
- `to_message_id`
- `content`
- `created_at`

## 9. 房间推进机制

每个房间由 scheduler 按 tick 周期推进。

单轮逻辑如下：

1. 检查房间是否处于运行状态。
2. 检查当前是否超过预算或进入降频状态。
3. 根据最近消息和房间热度判断是否需要产生新消息。
4. 如果需要，筛选候选角色。
5. 按关系、冷却、活跃度、是否被 `@`、最近发言次数等因素计算权重。
6. 选出角色并确定意图。
7. 构造 prompt 并调用模型。
8. 对结果做去重、截断和基本过滤。
9. 存储消息并推送到前端。
10. 达到阈值时生成摘要。

### 9.1 角色选择权重因素

建议至少包含以下因素：

- 最近是否发过言
- 当前是否在冷却期
- 角色活跃度
- 与上一条发言者的敌意或亲近度
- 是否被用户点名
- 是否属于当前高热阵营
- 当前房间冲突值是否偏高

### 9.2 何时不发言

系统不应机械定时发言，以下情况应允许跳过本轮：

- 房间最近已经过于密集输出
- 没有明显接话动力
- 今日预算接近上限
- 模型接口持续失败
- 生成结果与历史高度重复

### 9.3 摘要与压缩

消息不能无限累积为模型上下文。系统应在达到阈值时：

- 提取最近一段对话的结构化摘要
- 记录主要冲突点、站队关系和未解决话题
- 后续 prompt 优先使用摘要 + 最近少量原始消息

## 10. Prompt 构造

MVP 的 prompt 建议由四段组成：

1. 房间规则
2. 角色定义
3. 对话摘要
4. 本轮任务

### 10.1 房间规则

包括：

- 当前房间主题
- 场景风格
- 输出语言
- 长度限制
- 安全边界

### 10.2 角色定义

包括：

- 角色公开身份
- 说话风格
- 当前立场
- 目标
- 禁区
- 阵营偏向
- 与关键人物的关系

### 10.3 对话上下文

包括：

- 最近摘要
- 最近若干条消息
- 被引用或被点名的消息

### 10.4 本轮任务

由 engine 动态生成，例如：

- 回应某角色的挑衅
- 为本阵营辩护
- 转移话题
- 拱火
- 阴阳怪气
- 尝试缓和局面

## 11. LLM 接入规范

MVP 只支持 OpenAI-compatible `chat completions` 接口。

最低要求：

- 允许自定义 `base_url`
- 允许自定义 `api_key`
- 允许角色级别指定 `model`
- 支持 `temperature`
- 支持 `max_tokens`
- 支持统一超时和重试

初期不做复杂 provider 抽象，只定义一个轻量接口，底层默认实现为 OpenAI-compatible client。

后续若要接入更多协议，再在 `llm` 层扩展 adapter。

## 12. API 草案

### 12.1 前台接口

- `GET /`：背景板首页
- `GET /rooms/:id`：房间页面
- `GET /api/rooms/:id/messages`：获取历史消息
- `GET /api/rooms/:id/events`：SSE 事件流
- `POST /api/rooms/:id/input`：用户插话

### 12.2 后台接口

- `GET /admin`
- `GET /api/admin/rooms`
- `POST /api/admin/rooms`
- `PATCH /api/admin/rooms/:id`
- `GET /api/admin/personas`
- `POST /api/admin/personas`
- `PATCH /api/admin/personas/:id`
- `GET /api/admin/factions`
- `POST /api/admin/factions`
- `PATCH /api/admin/factions/:id`
- `GET /api/admin/providers`
- `POST /api/admin/providers`
- `PATCH /api/admin/providers/:id`

### 12.3 控制接口

- `POST /api/admin/rooms/:id/pause`
- `POST /api/admin/rooms/:id/resume`
- `POST /api/admin/rooms/:id/tick`

## 13. 前端界面草案

### 13.1 房间页

页面建议采用三栏布局：

- 左栏：房间信息、热度、状态、预算和运行开关
- 中栏：消息流
- 右栏：角色卡和阵营概览

底部保留输入框，支持：

- 普通插话
- `@角色`
- 简单指令，例如“让他们继续吵”

### 13.2 管理页

管理页优先实现表单化，不追求复杂可视化：

- 房间列表
- 角色列表
- 阵营列表
- Provider 配置
- 简单运行状态面板

## 14. 配置与部署

### 14.1 配置来源

当前采用以下配置来源：

1. 配置文件
2. 程序内置默认值

配置内容包括：

- 服务监听地址
- SQLite 文件路径
- 管理后台口令或 basic auth 配置
- 默认超时
- 默认摘要阈值
- 全局日志级别

### 14.2 部署形态

目标部署方式：

- 一个二进制
- 一个 SQLite 数据文件
- 一个可选配置文件
- 一个反向代理

适用环境：

- 单 VPS
- 家用服务器
- 小型容器环境

### 14.3 启动方式

系统启动时需完成：

- 加载配置
- 初始化日志
- 打开 SQLite
- 执行迁移
- 加载房间调度状态
- 启动 HTTP 服务
- 启动 scheduler

## 15. 稳定性与成本控制

MVP 必须内建以下保护机制：

- 单房间限频
- 单日预算限制
- 错误重试和退避
- Provider 失败时暂停房间或降频
- 相似内容检测
- 输出长度限制
- 摘要压缩
- 运行状态可观测

建议记录以下指标：

- 每房间消息数
- 每房间 token 消耗
- 模型调用失败率
- 平均响应时长
- 最近摘要时间

## 16. 安全与边界

MVP 默认作为个人或小范围使用工具，安全设计保持最小化，但至少应包含：

- 管理接口鉴权
- API Key 不明文暴露到前端
- 基础日志脱敏
- 用户输入长度限制
- 基础提示注入防护

若后续开放公网访问，再补充：

- 速率限制
- CSRF/Session 策略
- 更严格的内容审查

## 17. MVP 范围

第一阶段明确只交付以下能力：

- 单机运行
- 单二进制部署
- SQLite 持久化
- OpenAI-compatible 模型接入
- 单房间或少量房间
- 3 到 5 个可配置角色
- 阵营支持
- 角色关系支持
- 进程内调度
- 前台实时消息展示
- 用户插话
- 基础管理页

## 18. 里程碑建议

### M1: 可跑骨架

- Go 项目初始化
- 配置加载
- SQLite 初始化
- 基础 HTTP 服务
- 首页和房间页骨架

### M2: 基础对话闭环

- Persona/Faction/Room schema
- OpenAI-compatible client
- 单房间 scheduler
- 自动产出消息并展示

### M3: 可配置化

- 管理页
- Provider 配置
- 角色和阵营编辑
- 用户插话与 `@角色`

### M4: 长时间运行能力

- 摘要压缩
- 降频和预算控制
- 重复检测
- 运行状态监控

## 19. 当前默认假设

若后续没有新信息，默认按以下假设推进实现：

- 系统主要由单个管理员使用
- 管理后台采用最简单可用的鉴权方案
- 默认输出语言为中文
- 初期不接入图片、语音、工具调用
- 初期不做流式逐字输出
- 初期不做复杂角色记忆演化系统

## 20. 待确认问题

以下问题不阻塞骨架开发，但会影响后续细节：

- 管理后台是否只需要本地可用，还是需要公网可访问
- 是否需要房间模板功能
- 是否需要用户自定义角色头像上传
- 是否需要回放和导出能力
- 是否需要更强的内容边界控制

---

该文档用于固化第一阶段的产品与技术基线。后续实现若偏离此文档，应同步更新 spec。
