# Clash-TUI 现代化界面重构设计文档

## 1. 设计愿景与核心原则

*   **视觉现代化 (Modern Aesthetics)**：引入 Nerd Fonts 图标系统，使用类似 Catppuccin、Tokyo Night 等现代代码编辑器的流行配色方案，告别刺眼的基础终端颜色。
*   **信息可视化 (Data Visualization)**：将枯燥的数字转化为微图表（如使用盲文符号绘制的实时流量折线图、延迟条形图）。
*   **高密度与呼吸感并存 (Space & Density)**：合理利用 Lipgloss 的 Padding 和 Margin，让模块之间有清晰的边界，但在列表页保持高信息密度。
*   **所见即所得的交互 (Intuitive Interaction)**：底部栏常驻当前上下文相关的快捷键（Keybindings），避免用户死记硬背。

---

## 2. 全局布局架构 (Global Layout)

我们将整个屏幕划分为 **四大部分**，采用上中下结构，并优化各部分的边界感：

```text
┌────────────────────────────────────────────────────────────────────────┐
│ 🚀 Clash-TUI  │ 🟢 CONNECTED │ ↑ 12.5 KB/s  ↓ 2.4 MB/s │ 🎛️ RULE │ ⚙️ v1.18 │  <-- 1. 全局顶栏 (Header)
├────────────────────────────────────────────────────────────────────────┤
│  󰕮 Dashboard    󰤨 Network(Proxies)    󰈯 Profiles    󰒓 System         │  <-- 2. 导航栏 (Tabs)
├────────────────────────────────────────────────────────────────────────┤
│ ╭─────────────────╮ ╭────────────────────────────────────────────────╮ │
│ │                 │ │                                                │ │
│ │                 │ │                                                │ │
│ │   Left Panel    │ │                 Right Panel                    │ │  <-- 3. 主内容区 (Body)
│ │                 │ │                                                │ │
│ │                 │ │                                                │ │
│ ╰─────────────────╯ ╰────────────────────────────────────────────────╯ │
├────────────────────────────────────────────────────────────────────────┤
│ [Ctrl+K] Command Palette │ [1-4] Switch Tab │ [q] Quit  │ 📝 Log info. │  <-- 4. 底部状态与快捷键 (Footer)
└────────────────────────────────────────────────────────────────────────┘
```

### 2.1 顶栏 (Header Bar) - 增强状态感知
*   **Logo/Title**: 使用渐变色或品牌色高亮显示。
*   **核心指标 (新增)**：必须加入**实时上下行网速 (`tx`/`rx`)**，这是代理软件用户最关心的全局数据。
*   **状态胶囊 (Pills)**：采用圆角背景（如 `lipgloss.RoundedBorder` 的变体或色块反转），显示当前的模式（Rule/Global/Direct）。

### 2.2 导航栏 (Tab Bar) - 图标化
*   废弃目前生硬的 `1 Dashboard` 文字，引入 Nerd Font 图标：
    *   `󰕮 Dashboard` (主面板)
    *   `󰤨 Network` (网络/节点)
    *   `󰈯 Profiles` (配置/订阅)
    *   `󰒓 System` (系统/日志)
*   **活动标签（Active Tab）**：下方使用一条亮色高亮线（如蓝色或主色调），未选中标签保持变暗（Muted）。

### 2.3 底部栏 (Footer) - 上下文感知
*   分为左右两块。左侧提示**当前页面可用**的快捷键（如在节点页显示 `[Enter] Select Node  [t] Test Delay`），右侧显示最新的一条系统日志或提示信息（跑马灯效果）。

---

## 3. 核心页面详细设计 (Page Designs)

### 3.1 Dashboard (总览面板) - 强调可视化
**布局：上侧卡片网格 + 下侧日志/事件池**

*   **实时流量图 (Traffic Graph)**：使用 `charmbracelet/bubbles/timer` 配合自定义的盲文字符（Braille，如 `⡷⡷⣠⣄⡀`）渲染一个占据屏幕 1/3 宽度的双色折线图（上传绿色，下载蓝色）。
*   **运行时状态卡片 (Runtime Card)**：
    *   提供类似仪表盘的数据：总连接数 (`Connections`)、内存占用 (`Memory`)、启动时间 (`Uptime`)。
*   **快速操作区 (Quick Actions)**：
    *   以水平按键列表的形式展示，如 `[R] Rule` `[G] Global` `[D] Direct`，当前激活状态的按键呈按下（反色）状态。

### 3.2 Network -> Proxies (节点选择页) - 重点重构
这是用户停留时间最长的页面，目前的左 Group 右 Node 设计合理，但视觉呈现需要极大优化。

**左侧（Proxy Groups）**：
*   列表需要有明确的选中态（左侧显示一个主色调的竖线 `┃`，整行背景微亮）。
*   列表项不仅显示组名，右侧对齐显示该组**当前的策略**（如 `Auto` / `Select` / `Fallback`）。

**右侧（Nodes / 节点列表）**：
*   **延迟可视化**：延迟不能只显示数字，需增加可视化的彩色条（Bar）或小圆点：
    *   `< 100ms`：🟢 绿色 + 短条
    *   `100 - 300ms`：🟡 黄色 + 中条
    *   `> 300ms`：🔴 红色 + 长条
    *   `Timeout`：⚪ 灰色 `Timeout`
*   **当前选中节点**：节点名前增加显眼的 `✓` 或 `★` 图标，并且该行文字加粗并使用高亮色。
*   **布局对齐**：使用类似表格的对齐方式，左侧节点名称，右侧固定宽度显示延迟。

### 3.3 Profiles (订阅管理) - 卡片式列表
*   摒弃简单的文本罗列。每个订阅项渲染为一个“卡片”（由上下的 Lipgloss Border 包围）。
*   **内容分布**：
    *   首行：`[订阅名称]   🟢 Enabled`
    *   次行：`Provider: https://...`
    *   第三行：`Last Updated: 2h ago | 剩余流量信息 (如果 API 支持)`
*   选中的卡片边框变色（如变成蓝色 `Primary`），未选中的保持灰色边框。

### 3.4 System -> Connections (连接管理)
*   改用真正的**表格组件**（推荐引入 `charmbracelet/bubbles/table`）。
*   **表头**：`Host / IP` | `Process` | `Network` | `Traffic` | `Chain`
*   **样式**：表头加粗带下划线，斑马纹背景（奇偶行颜色交替），使得长串的连接信息便于扫视。

---

## 4. 视觉与色彩系统 (Color System)

废弃基于 0-255 纯色终端的粗糙感，引入现代化的、经过精心调配的主题色盘（可通过 Lipgloss 兼容 true-color 或优雅降级）：

建议以 **Tokyo Night** 或 **Catppuccin Mocha** 作为默认主题（提取其 HEX 值）：

| 逻辑角色 | 颜色分配 (以 Catppuccin 为例) | 用途说明 |
| :--- | :--- | :--- |
| **Background** | `#1e1e2e` (Base) | 面板内部背景色 (若终端支持) |
| **Surface** | `#313244` (Surface0) | 选中项背景色、非活动 Tab 背景 |
| **Border/Muted** | `#45475a` (Surface1) | 面板边框、次要文本、分隔线 |
| **Primary/Accent** | `#cba6f7` (Mauve/紫) | 标题、激活状态、高亮边框、主要图标 |
| **Secondary** | `#89b4fa` (Blue/蓝) | 当前选中的 Tab、下载流量图色 |
| **Success** | `#a6e3a1` (Green) | 🟢 节点延迟极低、连接成功状态 |
| **Warning** | `#f9e2af` (Yellow) | 🟡 节点延迟中等、加载中状态 |
| **Error** | `#f38ba8` (Red) | 🔴 节点超时、错误提示、关闭连接危险操作 |

*注：在 `styles.go` 中，应将颜色定义从 "86", "244" 等数字转为明确的 HEX 色值（Lipgloss 会根据终端类型自动降级）。*

---

## 5. 组件与动效设计 (Components & Micro-Interactions)

1.  **面板边框 (Borders)**: 
    *   使用 `lipgloss.RoundedBorder()`。
    *   当焦点（Focus）位于某个面板（如切换到了右侧节点列表）时，该面板的**边框颜色应亮起**（变为 Primary色），而非焦点面板边框变为 Muted 灰色。这是 TUI 中极度重要的方向感知设计。
2.  **加载状态 (Loading)**:
    *   在测试节点延迟（`t` 或 `T`）时，延迟数字位置替换为旋转的 Spinner（使用 `bubbles/spinner`，如 `⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏`），测试完成后再跳回数字。
3.  **Command Palette (命令面板)**:
    *   增加背景阴影（Shadow）或半透明效果（使用暗色色块）。
    *   使其在屏幕正中间悬浮，使用明确的双线边框 `lipgloss.DoubleBorder()` 突出层级，输入框增加光标闪烁效果。

---

## 6. 代码级重构建议 (Implementation Guide)

如果你准备按照此设计落地，建议在 `app.go` 和代码结构中引入以下改动：

1.  **引入外部 Bubble 组件**：
    *   由于目前列表逻辑都是自己写的（通过数组和游标），缺乏滚动条和丝滑交互。建议重构时引入 `github.com/charmbracelet/bubbles/list` 处理 Profiles 和 Proxy Groups。
    *   引入 `github.com/charmbracelet/bubbles/table` 处理 Connections。
    *   引入 `github.com/charmbracelet/bubbles/spinner` 处理测速加载。
2.  **提取组件渲染**：
    `app.go` 目前过于庞大，所有的 `renderXXX` 都在一个文件。建议拆分：
    *   `internal/ui/views/dashboard.go`
    *   `internal/ui/views/proxies.go`
    *   `internal/ui/views/connections.go`
3.  **Focus 管理机制**：
    在 `model` 中维护一个显式的 `focusedPanel` 状态枚举。在按 `h`/`l` 或 `Tab` 切换面板时，根据 `focusedPanel` 改变该面板的边框颜色函数调用。
