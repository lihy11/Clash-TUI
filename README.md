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
3. 首次启动会自动准备 Mihomo 内核，并进入 TUI。

如果你是开发者，也可以源码启动：

```bash
go mod tidy
go run ./cmd/clash-tui
```

## 怎么使用

- `Overview`：查看状态，切换 `Rule/Global/Direct`。
- `Proxies`：选择代理组与节点，支持单节点测速和一键测速。
- `Profiles`：导入订阅、更新当前订阅、更新全部订阅。
- `Connections`：查看并关闭活跃连接。
- `Settings`：修改控制器地址、密钥、轮询参数。

常用快捷键：
- `q` 退出
- `tab` / `shift+tab` 切页
- `1-7` 快速跳页

## 许可证

- 项目许可证：`MIT`（见 [LICENSE](./LICENSE)）
- 第三方许可证说明：见 [THIRD_PARTY_NOTICES.md](./THIRD_PARTY_NOTICES.md)
