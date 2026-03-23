# clash-tui 实现文档

## 1. 目标与范围

项目目标：在终端内提供接近 ClashX / Clash Verge 的日常操作能力，基于 Mihomo External Controller API 完成控制和状态展示。

当前版本覆盖：

- 自动下载/启动 Mihomo 内核（可关闭）
- 代理模式切换：Rule / Global / Direct
- 代理组与节点切换
- 节点延迟测试
- 连接查看与关闭
- 规则查看
- 日志流查看
- 订阅导入与更新（单个/全部）
- 控制器配置持久化

## 2. 技术选型

- 语言：Go 1.22+
- TUI：
  - `github.com/charmbracelet/bubbletea`
  - `github.com/charmbracelet/bubbles/textinput`
  - `github.com/charmbracelet/lipgloss`
- 网络：
  - `net/http`（REST）
  - `github.com/gorilla/websocket`（`/logs` 实时流）
- 配置：
  - `gopkg.in/yaml.v3`
  - 配置文件路径：`~/.config/clash-tui/config.yaml`

## 3. 架构概览

分层设计：

1. `internal/mihomo`
   - Mihomo API client
   - 统一请求封装、鉴权、错误返回
   - WebSocket 日志读取
2. `internal/ui`
   - Bubble Tea Model / Update / View
   - 多页面状态与键鼠行为
   - 命令调度（异步请求、轮询、日志流）
3. `internal/config`
   - 本地配置加载与保存
4. `internal/subscription`
   - 订阅元信息管理（导入/持久化）
   - 生成 Mihomo 配置（proxy-providers + groups + rules）
5. `internal/core`
   - Mihomo 内核下载、解压、启动、重启
6. `internal/runtime`
   - 启动编排（boot/reload）
7. `cmd/clash-tui`
   - 程序入口和 TUI 启动参数

## 4. 页面与交互设计

页面：

1. Overview
   - 当前模式、版本、连接状态
   - Provider 概览
   - 快捷模式切换
2. Proxies
   - 左侧代理组（selector/urltest/fallback/loadbalance）
   - 右侧节点列表
   - 切换节点、延迟测试
3. Connections
   - 活跃连接列表
   - 关闭单连接 / 全部连接
4. Rules
   - 规则表滚动浏览
5. Logs
   - `/logs` 实时日志流
6. Profiles
   - 订阅导入
   - 更新当前订阅 / 更新全部订阅
7. Settings
   - endpoint/secret/poll/log-level 修改并保存

## 5. 轮询与流式更新策略

- 固定轮询（默认 2 秒）
  - `/configs`
  - `/proxies`
  - `/connections`
- 稀疏轮询
  - `/providers/proxies`（每 5 次轮询）
  - `/rules`（每 15 次轮询）
- 实时流
  - `/logs?level=<level>` 通过 WebSocket 持续拉取
  - 失败后 2 秒自动重连

## 6. 数据接口定义（Mihomo External Controller）

说明：以下为项目当前使用到的接口。

### 6.1 基础信息

- `GET /version`
  - 返回：`{ "version": "..." }`

### 6.2 配置

- `GET /configs`
  - 读取 mode/log-level/ipv6 等基础配置
- `PATCH /configs`
  - 请求体：`{ "mode": "rule|global|direct" }`

### 6.3 代理与节点

- `GET /proxies`
  - 获取所有代理项及代理组
- `PUT /proxies/{group}`
  - 请求体：`{ "name": "<node-name>" }`
  - 用于切换某代理组的当前节点
- `GET /proxies/{name}/delay?url=...&timeout=...`
  - 节点延迟测试

### 6.4 规则

- `GET /rules`

### 6.5 连接

- `GET /connections`
- `DELETE /connections/{id}`
- `DELETE /connections`

### 6.6 Provider

- `GET /providers/proxies`
- `PUT /providers/proxies/{name}`

`providers.<name>.proxies` 在不同内核/版本中可能出现两种结构：

- `[]string`（仅节点名）
- `[]object`（包含 `name/type/alive/history...`）

当前实现已做兼容解析。

### 6.7 日志

- `GET /logs`（WebSocket）
  - query：`level=debug|info|warning|error`

### 6.8 鉴权

- Header：`Authorization: Bearer <secret>`

## 7. 错误处理

- 所有请求统一检查 HTTP 状态码（>=300 视为错误）
- 请求超时默认 5~10 秒，按接口分配
- UI 层统一接收错误消息并显示在状态栏
- 日志流断开自动重连

## 8. 配置文件结构

`~/.config/clash-tui/config.yaml`

```yaml
endpoint: http://127.0.0.1:9090
secret: ""
poll_interval: 2s
log_level: info
manage_core: true
core_version: latest
mixed_port: 7890
```

## 9. 运行时目录

- 配置目录：`~/.config/clash-tui`
  - `config.yaml`
  - `subscriptions.yaml`
- 数据目录：`~/.cache/clash-tui`
  - `core/mihomo`（或 `mihomo.exe`）
  - `mihomo-config.yaml`
  - `proxy_providers/*.yaml`

## 10. 打包与发布

- 发行配置：`.goreleaser.yaml`
- 支持目标：`linux/darwin/windows` + `amd64/arm64`
- 产物：`tar.gz/zip` + `deb/rpm`
- 一键安装脚本：`scripts/install.sh`
- CI 工作流：`.github/workflows/ci.yml`
- 发布工作流：`.github/workflows/release.yml`（push `v*` tag 或手动触发）

## 11. 依赖清单

- `github.com/charmbracelet/bubbletea v1.2.4`
- `github.com/charmbracelet/bubbles v0.20.0`
- `github.com/charmbracelet/lipgloss v1.0.0`
- `github.com/gorilla/websocket v1.5.3`
- `gopkg.in/yaml.v3 v3.0.1`

## 12. 后续增强建议（对齐 ClashX/Verge 体验）

- 增加 Providers 专属页面（Provider 内节点选择、healthcheck）
- 增加 Traffic 图形化（实时上下行速率图）
- 增加 Profiles 切换与配置 reload
- 增加 DNS Query 页面
- 增加 TUN 状态可视化与切换（取决于内核能力）
