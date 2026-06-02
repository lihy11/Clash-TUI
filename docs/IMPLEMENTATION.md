# clash-tui 实现文档

## 1. 目标与范围

项目目标：在终端内提供接近 ClashX / Clash Verge 的日常操作能力，基于 Mihomo External Controller API 完成控制和状态展示。

当前版本覆盖：

- 自动下载/启动 Mihomo 内核（可关闭）
- Mihomo 内核下载支持 GitHub API、镜像源、断点续传与重试
- 代理模式切换：Rule / Global / Direct
- 代理组与节点切换
- 节点延迟测试
- 连接查看与关闭
- 规则查看
- 日志流查看
- 订阅导入与更新（单个/全部）
- 控制器配置持久化
- 中英文界面切换（Header 语言下拉样式）

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
   - Mihomo 内核下载、解压、启动、重试、重启
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
  - 读取 mode/log-level/ipv6、`system-proxy`、`tun.enable` 等配置
- `PATCH /configs`
  - 请求体：`{ "mode": "rule|global|direct" }`
  - 请求体（系统代理）：`{ "system-proxy": true|false }`
  - 请求体（TUN）：`{ "tun": { "enable": true|false } }`
  - 兼容说明：部分内核版本 `GET /configs` 不返回 `system-proxy` 字段；UI 侧会在这种情况下使用本地状态镜像保持开关可用。

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
- 内核下载优先使用 GitHub Release API；页面解析作为回退路径
- 内核下载会尝试官方 GitHub URL、自定义代理、默认镜像源，并对候选资产进行重试

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
language: zh-CN
```

`language` 可选值：

- `en`
- `zh-CN`

环境变量：

- `MIHOMO_CONTROLLER`：覆盖 External Controller 地址
- `MIHOMO_SECRET`：覆盖 External Controller 密钥
- `CLASH_TUI_GITHUB_PROXY`：内核下载 GitHub 代理前缀，例如 `https://ghproxy.example.com`
- `CLASH_TUI_INSECURE_TLS=1`：内核下载时关闭 TLS 证书校验，仅用于本地网络/证书问题排查

## 9. 运行时目录

- 配置目录：`~/.config/clash-tui`
  - `config.yaml`
  - `subscriptions.yaml`
- 数据目录：`~/.cache/clash-tui`
  - `core/mihomo`（或 `mihomo.exe`）
  - `core/mihomo.download.part`（下载中的临时文件，支持断点续传）
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

## 13. 2026-03-31 TUI 视觉优化落地

本次基于 `tui_design_optimization.md` 完成了以下实现：

- 空间留白
  - `panel` 内边距由 `Padding(0, 1)` 调整为 `Padding(1, 2)`。
  - `panelTitle` 增加 `MarginBottom(1)`，标题与内容分层更清晰。
  - Header/Footer 水平内边距统一提升到 `Padding(0, 2)`，并在渲染时按内容区宽度计算，避免挤压。
- 布局比例
  - Proxies 页双栏布局：左侧组列表宽度由 `max(22, w/3)` 调整为 `max(26, w/4)`，释放右侧节点列表空间。
- 主题与颜色映射
  - Command Palette 浮层背景改为主题 `Surface`，去除硬编码 `235`。
  - 主 Tab 激活态取消下划线，改为 `Surface` 背景 + `Primary` 前景。
  - 光标高亮统一为 `Primary` 背景 + `Surface` 前景。
- 次级导航与按钮
  - SubTab 栏改为 `Panel` 背景，SubTab 项左右 Padding 增加为 `Padding(0, 2)`。
  - SubTab 激活态采用 `Primary` 前景加粗，弱化符号前缀，提升工具栏一致性。
  - 动作按钮（如 `Test All`）水平内边距提升到 `Padding(0, 2)`。
- 列表对齐
  - Rules 与 Connections 页面引入固定列宽渲染（`fitTextWidth`），实现列对齐显示，减少文本抖动和拼接拥挤感。

受影响文件：

- `internal/ui/styles.go`
- `internal/ui/app.go`

## 14. 2026-03-31 Dashboard 系统代理/TUN 快速开关

本次在 Dashboard 增加了显眼的运行能力开关，并直连 Mihomo 内核配置接口：

- 新增开关
  - `System Proxy`（`ON/OFF`）
  - `TUN Mode`（`ON/OFF`）
- 交互方式
  - 鼠标点击 Dashboard 中的按钮直接切换
  - 键盘快捷键：`s` 切换系统代理，`n` 切换 TUN
  - Command Palette 新增：
    - `Toggle: System Proxy`
    - `Toggle: TUN Mode`
- 调用链路
  - `internal/ui/app.go`：
    - 新增 `setSystemProxyCmd` / `setTunCmd`
    - 新增 `featureSetMsg` 状态回传与 UI 状态同步
  - `internal/mihomo/client.go`：
    - 新增 `SetSystemProxy(ctx, enable)`
    - 新增 `SetTun(ctx, enable)`
  - `internal/mihomo/types.go`：
    - `ConfigResponse` 新增 `system-proxy`、`tun`
    - `TunConfig` 兼容 `tun` 字段 bool/object 两种返回格式

## 15. 2026-03-31 布局空间利用与 Dashboard 挤压换行修复

针对 `png/shot.png` 暴露的问题，本次完成了 3 处直接修复：

- 终端空间利用率
  - `Network` 与 `System` 页面 body 高度计算由 `h-subH-1` 修正为 `h-subH`，避免无意义预留导致底部可视区缩水。
  - `styles.app` 改为带主题 `Surface` 背景，`View()` 末尾强制渲染为 `width x height` 画布，修复透明终端下顶部/底部“露底”和右侧空白不一致观感。
- 顶部可用区域
  - `cmd/clash-tui/main.go` 移除了启动 TUI 前的常规 `log.Printf` 输出，降低在非 AltScreen/兼容性较差终端中的顶部占行问题。
- Dashboard 挤压换行
  - `internal/ui/app.go` 中 `Switch` 与 `Toggles` 行改为按面板内容宽度自适应分行（而非硬拼一行），并同步修正点击热区坐标，确保换行后鼠标交互仍准确。
- Header/Footer 宽度边界
  - Header/Footer 在进行左右对齐前额外预留 2 列安全宽度，并将 footer 状态前缀由 emoji 改为 ASCII（`status`），降低不同终端字体宽度判定差异导致的自动换行概率。
 - 宽高语义修正（关键）
   - 修复 `lipgloss.Style.Width/Height` 使用语义错误：此前把 `outer - frameSize` 传给 `Width/Height`，导致边框块体被二次缩小，出现右侧锯齿空白和底部大空白。
   - 现统一改为 `Width(outerW)` / `Height(outerH)`，文本可用宽度仍按内容区单独计算（`outer - frameSize`）。

受影响文件：

- `internal/ui/app.go`
- `internal/ui/styles.go`
- `cmd/clash-tui/main.go`
- `internal/ui/layout_test.go`

## 16. 2026-04-01 中英文界面与语言下拉

本次新增界面语言切换能力，并将 Header 右上角原版本位替换为语言选择入口：

- 语言支持
  - `English` / `中文`
  - 语言状态持久化到 `~/.config/clash-tui/config.yaml` 的 `language` 字段。
- 交互
  - 鼠标点击 Header 右上角语言开关（`EN/中文`）即时切换。
  - 键盘 `L` 可快速切换语言。
- UI 替换
  - Header 右上角由 `v<version>` 替换为 `Lang: <value> ▾`（中文模式显示 `语言: <value> ▾`）。
- 文案国际化
  - 覆盖 Header、Tabs、Dashboard、Network、Profiles、System、Footer、Settings 等主要可视文本。

受影响文件：

- `internal/config/config.go`
- `internal/ui/i18n.go`
- `internal/ui/app.go`

## 17. 2026-04-01 Bubble 组件化重构

为减少手写 UI 逻辑并提升可维护性，本次把核心页面切换为 Bubbles 组件驱动：

- Footer 帮助区
  - 从手写提示字符串切换到 `help` + `key` 组合，按当前 tab 动态生成快捷键说明。
- Rules / Connections
  - 从手写列宽拼接切换到 `table` 组件，复用组件内置光标与滚动能力。

## 18. 2026-04-01 UI 结构重构与测试补齐

本次在不改变现有交互行为的前提下，完成 `internal/ui` 的结构性拆分，降低 `app.go` 复杂度并增强回归保障：

- 文件拆分
  - `internal/ui/views.go`
    - 收敛 Header/Tabs/Footer 与主要页面渲染方法。
  - `internal/ui/input_handlers.go`
    - 收敛全部输入处理（全局键盘、鼠标、各页 `handle*`）。
  - `internal/ui/selector.go`
    - 收敛语言/主题/模式选择器逻辑与鼠标命中。
  - `internal/ui/app.go`
    - 保留 model、消息流转与状态同步，减少输入/渲染实现细节耦合。
- 状态命名清理
  - `langOpen` 更名为 `selectorOpen`，语义与当前功能一致（语言/主题/模式共用选择器）。
  - 移除未使用状态字段 `langCursor`。
- 测试补齐
  - 新增 `internal/ui/selector_test.go`：
    - `openSelector` 状态初始化
    - 主题选择应用与关闭行为
    - 选择器鼠标外部点击关闭
    - 选择器鼠标点击条目选择
  - 保留并复用 `internal/ui/mouse_test.go` 对触摸板滚动不触发选择、点击触发选择等行为回归保障。

验证结果：

- `go test ./internal/ui` 通过
- `go build ./...` 通过
- Logs / Notifications
  - 从手写截断渲染切换到 `viewport`，支持统一滚动与定位。
- Profiles
  - 订阅列表切换到 `list` 组件，选择与滚动由组件统一处理。
- 选择器统一
  - 语言、主题、模式统一使用 `list` 弹层选择器，减少重复下拉实现。

受影响文件：

- `internal/ui/app.go`
- `go.mod`
- `go.sum`

## 19. 2026-04-01 二次降复杂重构（消息流与命令分层）

为进一步降低单文件复杂度，本次继续按职责拆分 `internal/ui`，保持行为不变：

- 新增文件
  - `internal/ui/update_flow.go`：承接 `Update` 的键盘/消息分发与输入后处理逻辑。
  - `internal/ui/palette_and_settings.go`：收敛 Palette 动作与 Settings 保存重连。
  - `internal/ui/state_helpers.go`：收敛状态同步、输入初始化、布局与延迟显示辅助。
  - `internal/ui/commands.go`：收敛所有 Tea 命令构造（Mihomo/runtime 请求）。
  - `internal/ui/utils.go`：收敛宽度裁剪、速率格式化与 `min/max/clamp` 工具。
- 主文件变化
  - `internal/ui/app.go` 从“巨型聚合文件”进一步缩减到 model/入口主干，`Update` 改为分发壳层。
- 复杂度结果
  - `internal/ui/app.go` 当前约 387 行（此前约 1686 行）。

验证结果：

- `go test ./internal/ui -count=1` 通过
- `go test ./... -count=1` 通过
- `go build ./...` 通过

## 20. 2026-04-01 三次降复杂重构（渲染按页面分文件）

在不改变 UI 行为前提下，继续拆分原渲染聚合文件，按页面职责落盘：

- 渲染文件拆分
  - `internal/ui/views_shell.go`：Header/Tabs/SubTabs/Footer 与 `renderBody` 路由。
  - `internal/ui/views_overview.go`：Dashboard、Providers、Notifications 与趋势渲染。
  - `internal/ui/views_network.go`：Proxies、Rules、Connections 渲染。
  - `internal/ui/views_system_profiles.go`：Logs、Profiles、Settings 渲染。
- 清理
  - 删除旧的 `internal/ui/views.go`，避免多职责集中。

验证结果：

- `go test ./internal/ui -count=1` 通过
- `go test ./... -count=1` 通过
- `go build ./...` 通过

## 21. 2026-04-01 四次降复杂重构（Core/Runtime 分层）

本次将 `core` 与 `runtime` 从单文件重职责结构拆分为职责内聚文件，保持行为不变：

- `internal/core` 拆分
  - `manager.go`：管理器定义与公共入口。
  - `process.go`：进程生命周期与 PID 管理。
  - `release_download.go`：发布页解析、资产筛选、下载/解压、下载进度输出。
- `internal/runtime` 拆分
  - `manager.go`：Boot/Reload/Close 主流程。
  - `discovery.go`：controller 探测与本机配置候选发现。
  - `auto_import.go`：本机配置订阅自动导入与命名规范化。

重构后复杂度（行数）：

- `internal/core/*.go`：`89 + 166 + 364`
- `internal/runtime/*.go`：`108 + 139 + 194`

验证结果：

- `go test ./... -count=1` 通过
- `go build ./...` 通过
- `go vet ./...` 通过

## 23. 2026-04-12 UI 鼠标架构重构（统一动作路由）

本次开始把 UI 鼠标交互从“渲染时生成若干 `clickTarget`，更新时分支命中”迁移到统一动作路由：

- 新增基础设施
  - `internal/ui/mouse_router.go`
  - 统一维护 `hitBox + mouseAction + mouseRouter`
  - 由渲染阶段注册点击动作，`Update` 阶段按动作 ID 分发
- 已迁移区域
  - Header 语言切换
  - 主 Tab / Network 子 Tab / System 子 Tab
  - Overview 模式切换与开关点击
  - Proxies 模式切换、`Test All`、组选择、节点选择
  - Selector 弹层项选择与外部点击关闭
- 结构变化
  - `handleMouseMsg` 变成统一入口，优先走 `mouseRouter`
  - 页面渲染代码开始承担“注册动作”职责，减少更新阶段的页面分支命中逻辑
  - 旧 `clickTarget` 结构暂时保留给未完全迁移路径与兼容测试，后续可继续清理

本次目标不是一次性移除所有旧字段，而是先把高频交互主路径稳定迁到动作化架构，确保后续继续组件化时不再增加手工坐标耦合。

验证结果：

- `go test ./... -count=1` 通过
- `go build ./...` 通过
- `go vet ./...` 通过

## 22. 2026-04-01 五次降复杂重构（UI 输入/状态按职责拆分）

在保持交互语义不变前提下，继续对 `internal/ui` 做“不过度拆分”的职责拆分：

- 输入层拆分（由单一 `input_handlers.go` 拆为 3 个文件）
  - `input_global_mouse.go`：全局键盘与鼠标分发入口。
  - `input_network.go`：Proxies 页组/节点输入与点击选择逻辑。
  - `input_tabs.go`：Overview/Profiles/Connections/Rules/Logs/Settings/Palette 输入处理。
- 状态与渲染辅助拆分（由单一 `state_helpers.go` 拆为 3 个文件）
  - `state_proxy_sync.go`：代理组/节点同步与吞吐更新。
  - `state_inputs_notifications.go`：输入初始化、状态提示、通知队列、系统代理状态判断。
  - `render_helpers.go`：模式行/开关/面板/延迟单元格渲染辅助。

重构后复杂度（行数）：

- `input_global_mouse.go`：157
- `input_network.go`：83
- `input_tabs.go`：197
- `state_proxy_sync.go`：112
- `state_inputs_notifications.go`：88
- `render_helpers.go`：271

验证结果：

- `go test ./... -count=1` 通过
- `go build ./...` 通过
- `go vet ./...` 通过
