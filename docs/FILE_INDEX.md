# 文件索引

## 根目录

- `go.mod`：依赖声明
- `README.md`：项目说明、运行方式、快捷键
- `LICENSE`：项目主许可证（MIT）
- `THIRD_PARTY_NOTICES.md`：第三方依赖许可证索引
- `.goreleaser.yaml`：多平台构建与打包发布配置
- `scripts/install.sh`：一键安装脚本
- `.github/workflows/ci.yml`：基础构建校验工作流
- `.github/workflows/release.yml`：Tag 触发的自动发布工作流
- `docs/IMPLEMENTATION.md`：实现设计与接口定义
- `docs/FILE_INDEX.md`：本文件

## 可执行入口

- `cmd/clash-tui/main.go`
  - 加载配置
  - 初始化 UI model
  - 以 AltScreen + 鼠标模式启动程序

## 配置层

- `internal/config/config.go`
  - `Settings` 结构
  - 默认配置
  - 配置文件路径解析
  - `Load/Save` 读写逻辑
  - 环境变量覆盖：`MIHOMO_CONTROLLER`、`MIHOMO_SECRET`

## Mihomo API 层

- `internal/mihomo/types.go`
  - 所有 REST / WS 使用到的数据结构定义
- `internal/mihomo/client.go`
  - REST 请求封装
  - 鉴权 header 注入
  - 代理模式切换、节点切换、连接管理等接口实现
  - `/logs` websocket 连接与读取

## 运行时编排层

- `internal/subscription/manager.go`
  - 订阅导入与持久化（`subscriptions.yaml`）
  - 生成 Mihomo 配置（proxy-providers/groups/rules）
- `internal/core/manager.go`
  - Core 管理器结构与公共入口（`NewManager`/`EnsureBinary`/`WriteConfig`）
- `internal/core/process.go`
  - Mihomo 进程生命周期管理（start/stop/restart/pid）
- `internal/core/release_download.go`
  - Release 解析、资产筛选、下载进度、解压与 HTTP 客户端
- `internal/runtime/manager.go`
  - 启动编排主流程（boot/reload/close）
- `internal/runtime/discovery.go`
  - Controller 可用性探测与本机配置文件候选发现
- `internal/runtime/auto_import.go`
  - 从本机配置自动导入订阅条目与名称归一化

## UI 层

- `internal/ui/styles.go`
  - 主题样式、面板/标签/状态等样式定义
- `internal/ui/i18n.go`
  - 中英文文案映射与标签翻译辅助
- `internal/ui/input_global_mouse.go`
  - 全局键盘处理与主鼠标分发入口
  - 统一鼠标动作分发与少量兼容回退逻辑
- `internal/ui/mouse_router.go`
  - 命中框与鼠标动作注册/分发
  - 通过动作 ID 把渲染层点击区与业务处理解耦
- `internal/ui/input_network.go`
  - Network/Proxies 页键盘与鼠标处理（组/节点选择、延迟测试）
- `internal/ui/input_tabs.go`
  - Overview/Profiles/Rules/Connections/Logs/Settings/Palette 输入处理
- `internal/ui/update_flow.go`
  - `Update` 拆分后的消息分发流程
  - 键盘更新、消息处理与输入组件后处理
- `internal/ui/selector.go`
  - 主题/模式/语言选择器（`list` 组件）
  - 选择器渲染与路由化鼠标动作处理
- `internal/ui/palette_and_settings.go`
  - Command Palette 动作与设置保存重连逻辑
- `internal/ui/state_proxy_sync.go`
  - 代理组/节点状态同步、节点可视区与吞吐速率更新
- `internal/ui/state_inputs_notifications.go`
  - 输入初始化、状态提示与通知队列、系统代理状态镜像
- `internal/ui/render_helpers.go`
  - 面板与模式行渲染、开关渲染、延迟单元格渲染辅助
- `internal/ui/commands.go`
  - Mihomo 与 runtime 相关 Tea 命令构造器
- `internal/ui/utils.go`
  - 文本宽度处理、速率格式化、基础数值工具
- `internal/ui/views_shell.go`
  - Header/Tabs/SubTabs/Footer 壳层渲染
  - `renderBody` 页面路由
- `internal/ui/views_overview.go`
  - Dashboard/Providers/Notifications 渲染
- `internal/ui/views_network.go`
  - Proxies/Rules/Connections 渲染
- `internal/ui/views_system_profiles.go`
  - Logs/Profiles/Settings 渲染
- `internal/ui/app.go`
  - Bubble Tea 主 model
  - 主结构体与入口（`NewModel` / `Init` / `View`）
- `internal/ui/layout_test.go`
  - Header 行宽约束测试
- `internal/ui/mouse_test.go`
  - 鼠标滚轮/点击行为回归测试（节点选择、语言切换、统一路由）
- `internal/ui/mouse_router_test.go`
  - 鼠标动作优先级与分发测试
- `internal/ui/selector_test.go`
  - 选择器打开/关闭/鼠标选择行为测试（含统一路由路径）
