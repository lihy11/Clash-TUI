# Clash-TUI 视觉与交互优化设计文档 (TUI Design Optimization)

基于对当前项目 `internal/ui/styles.go` 和 `internal/ui/app.go` 的分析，Clash-TUI 已经具备了不错的基础框架和多主题支持。为了进一步提升 TUI 的高级感、呼吸感和交互清晰度，特制定以下视觉优化方案。

---

## 1. 空间与留白优化 (Spacing & Margins)

当前界面略显拥挤，主要是因为面板（Panel）和区块的内边距（Padding）较小，导致文字紧贴边框。

*   **面板内边距 (Panel Padding):**
    *   **现状:** `Padding(0, 1)` (上下0，左右1)
    *   **优化方案:** 修改为 `Padding(1, 2)`。增加上下的留白会让内容更有“呼吸感”，避免内容与圆角边框过于紧凑。Lipgloss 的布局会自动适配高度计算，不会破坏现有结构。
*   **标题间距 (Title Margin):**
    *   **现状:** `panelTitle` 与下方的内容紧贴。
    *   **优化方案:** 为 `panelTitle` 增加 `MarginBottom(1)`，让模块标题和实际内容产生明确的视觉分割。
*   **底部状态栏与页眉 (Header & Footer):**
    *   **优化方案:** 将两者的水平 Padding 增加至 `Padding(0, 2)`，使其与主体内容的左右边缘对齐，视觉上更加规整统一。

---

## 2. 布局宽度分配优化 (Layout Proportions)

主要针对 `Network -> Proxies` 页面（代理节点选择页面）的空间分配进行优化。

*   **侧边栏宽度 (Sidebar Width):**
    *   **现状:** 组列表（Groups）占据了界面的 `1/3` (最小 22 字符)。
    *   **优化方案:** 组名通常较短，不需要占据三分之一的横向空间。建议将左侧边栏（Groups）的宽度计算公式修改为 `w/4`，并设定合理的最小宽度（如 `max(26, w/4)`）。这样可以释放更多横向空间给右侧的节点列表（Nodes），以便更优雅地展示节点名称、连通性状态和延迟条（Delay Bar）。

---

## 3. 颜色与主题运用 (Colors & Themes)

虽然目前预设了 5 款很棒的主题（Dracula, Gruvbox 等），但在颜色应用（Color Mapping）上存在几个硬编码或不协调的地方：

*   **Command Palette (命令面板) 浮层:**
    *   **现状:** Overlay 的背景色硬编码为了 `235` (暗灰色)，这会导致在使用 Geist UI 等浅色或高反差主题时显得非常突兀。
    *   **优化方案:** 将 `overlay` 的背景色改为 `lipgloss.Color(t.Surface)` 或 `t.Panel`，确保与当前使用的主题色板保持一致。
*   **光标与选中态 (Cursor & Active Items):**
    *   **现状:** 列表光标使用了 `Secondary` 颜色作为背景，而 Tab 激活态使用了下划线 `Underline(true)`。
    *   **优化方案:**
        1.  取消顶部 Main Tab 激活态的下划线，改用色块区分：背景色设为 `Surface`，前景色设为 `Primary`，视觉上更像真正的“卡片式”或“抽屉式”标签页。
        2.  将 `cursor` (光标高亮行) 的背景色统一改为 `Primary` (配合 `Surface` 作为前景色)，让界面的视觉焦点（Accent）更加明确和突出。

---

## 4. 细节组件的精细化 (Component Polish)

*   **次级导航 (Sub-Tabs):**
    *   **现状:** SubTab 仅通过前缀字符（如 `• Proxies`）区分，且与下方内容连为一体。
    *   **优化方案:** 让 `subTabBar` 拥有一层微弱的背景色（如 `Panel` 色），并为 SubTab 项目增加左右 Padding（如 `Padding(0, 2)`），激活状态使用 `Primary` 色加粗，形成一条明确的工具栏分割带。
*   **动作按钮 (Action Buttons):**
    *   **现状:** 快速操作（如 Dashboard 的 Command Palette 入口，Proxies 里的 Test All）仅有 `Padding(0, 1)`。
    *   **优化方案:** 为 `action` 样式增加左右留白至 `Padding(0, 2)`，并在必要时配合明亮的背景色，使其看起来像一个真正的“按钮”组件，而不仅仅是反色文本。
*   **列表对齐 (List Alignment):**
    *   **现状:** 在 Rules 和 Connections 页面，文本直接拼接（如 `Host Process Chains`）。
    *   **优化方案:** 引入固定宽度格式化（类似于 `%-20s %-15s %s`）或直接使用 Lipgloss 的表格式布局（Table），确保多行数据的列是严格对齐的。这能大幅提升系统级数据面板的专业度和整洁度。

---

## 5. 实施建议 (Next Steps)

当前的 TUI 逻辑已经写得非常完善（自己实现了计算与渲染），UI 层面的重构成本其实很低。

开发者只需要在 `internal/ui/styles.go` 中微调 `Padding`、`Margin` 和部分 `lipgloss.Color(t.XXX)` 的映射关系；同时在 `internal/ui/app.go` 中修改一下 `proxyPanels` 方法里的 `leftW` 宽度计算公式即可。仅需这几处简单的修改，界面的高级感和现代感立刻就会有质的飞跃。