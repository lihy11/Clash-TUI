# clash-tui

`clash-tui` 是一个终端版 Clash/Mihomo 管理工具，支持鼠标和键盘操作，适合在服务器或本地终端快速切换节点与代理模式。

[![Release](https://img.shields.io/github/v/release/lihy11/Clash-TUI?label=download&logo=github)](https://github.com/lihy11/Clash-TUI/releases)
[![License](https://img.shields.io/github/license/lihy11/Clash-TUI)](./LICENSE)
[![Go Version](https://img.shields.io/github/go-mod/go-version/lihy11/Clash-TUI)](./go.mod)

## 快速下载

- Release 页面（推荐）：  
  https://github.com/lihy11/Clash-TUI/releases
- 一键安装脚本（Linux/macOS）：

```bash
curl -fsSL https://raw.githubusercontent.com/lihy11/Clash-TUI/main/scripts/install.sh | sh
```

## 安装与运行

1. 打开 Release 页面下载对应系统包（`macOS / Windows / Linux`）。
2. 安装后运行 `clash-tui`。
3. 启动时会先探测本机已运行的 Clash/Mihomo 控制器（如 `127.0.0.1:9090`），可用则直接连接并自动导入已有 `proxy-providers` 订阅；否则会自动下载并准备 Mihomo 内核（首次可能需要几十秒）。

如果你是开发者，也可以源码启动：

```bash
go mod tidy
go run ./cmd/clash-tui
```

## 怎么使用

- `Overview`：查看状态，切换 `Rule/Global/Direct`。
- `Network`：在 `Proxies / Rules / Connections` 子页间切换操作。
- `Profiles`：导入订阅、更新当前订阅、更新全部订阅。
- `System`：在 `Logs / Settings` 子页间切换。

常用快捷键：
- `q` 退出
- `tab` / `shift+tab` 切页
- `1-4` 快速跳页
- `[` / `]` 在当前主域切换子页
- `:` 或 `Ctrl+K` 打开命令面板
- `F2` 切换主题

## 许可证

- 项目许可证：`MIT`（见 [LICENSE](./LICENSE)）
- 第三方许可证说明：见 [THIRD_PARTY_NOTICES.md](./THIRD_PARTY_NOTICES.md)
