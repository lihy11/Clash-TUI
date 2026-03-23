# clash-tui

`clash-tui` 是一个基于 `Go + Bubble Tea` 的 Mihomo/Clash 终端 UI，目标是尽量对齐 ClashX / Clash Verge 的高频操作体验：代理模式切换、代理组节点切换、连接管理、规则查看、日志查看、控制器配置。

## 功能范围

- Overview：运行状态、Controller 信息、模式切换（Rule/Global/Direct）、Provider 概览
- Proxies：代理组和节点切换、节点延迟测试、模式点击切换（Rule/Global/Direct）、一键批量测速
- Connections：查看当前连接、关闭单条/全部连接
- Rules：规则列表滚动查看
- Logs：`/logs` 实时日志流
- Settings：控制器地址/密钥/轮询频率/日志级别配置与持久化

## 运行要求

- Go `1.22+`
- 已运行的 Mihomo/Clash 内核，且启用 external controller
- 推荐配置示例（mihomo）：

```yaml
external-controller: 127.0.0.1:9090
secret: "your-secret"
```

## 安装与启动

```bash
go mod tidy
go run ./cmd/clash-tui
```

可选环境变量：

- `MIHOMO_CONTROLLER`：例如 `http://127.0.0.1:9090`
- `MIHOMO_SECRET`：controller secret

## 快捷键

- 全局：`q` 退出，`tab`/`shift+tab` 切页，`1-6` 直达页面，鼠标点击顶部标签切页
- Overview：`r`/`g`/`d` 切换 Rule/Global/Direct
- Overview：鼠标可点 `Switch: [✓/○ Rule|Global|Direct]` 按钮切换模式
- Overview：`j/k` 选择 provider，`u` 更新 provider
- Proxies：`h/l` 切换左右面板，`j/k` 或 `↑/↓` 移动，`enter` 切换节点，`t` 测延迟
- Proxies：鼠标可点代理组、节点、Mode 按钮（点击即切换）
- Proxies：`T` 批量测速当前组，或点击 `Action: [⚡ Test All]`
- Connections：`x` 关闭当前连接，`X` 关闭全部连接
- Rules：`j/k` 或 `↑/↓` 滚动
- Logs：`c` 清空日志
- Settings：`tab`/`shift+tab` 切换输入项，`s` 保存并重连

## 文档

- 工程设计与接口定义：`docs/IMPLEMENTATION.md`
- 文件索引：`docs/FILE_INDEX.md`
