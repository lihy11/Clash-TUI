# TODO

## UI 信息架构重构（Mode / Group / Node）

- 问题：当前 `Network -> Proxies` 界面把 `Mode` 放在节点区域，层级表达不清晰，容易误解为与组/节点同级。
- 目标：按“`Mode`（全局） -> `Policy Group`（策略） -> `Node`（出站）”的真实关系重构布局。

### 待办项

- 将 `Mode` 切换从节点面板移到更高层（建议顶栏或全局控制区）。
- `Network` 主体仅保留“左侧 Group / 右侧 Node”的操作流。
- 在右侧标题增加上下文信息：`Group`、`Type(select/url-test/fallback)`、`Now`。
- 明确 `Global` 模式下的交互提示（弱化规则，突出当前全局出站）。
- 设计后补充交互规范：鼠标点击、键盘切换、状态提示的一致性。

