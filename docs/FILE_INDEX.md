# 文件索引

## 根目录

- `go.mod`：依赖声明
- `README.md`：项目说明、运行方式、快捷键
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
  - Mihomo 内核下载、解压、启动、重启
- `internal/runtime/manager.go`
  - 启动编排（boot）与配置变更后重载（reload）

## UI 层

- `internal/ui/styles.go`
  - 主题样式、面板/标签/状态等样式定义
- `internal/ui/app.go`
  - Bubble Tea 主 model
  - 页面渲染（Overview/Proxies/Connections/Rules/Logs/Profiles/Settings）
  - 快捷键与鼠标处理
  - 轮询调度与消息处理
