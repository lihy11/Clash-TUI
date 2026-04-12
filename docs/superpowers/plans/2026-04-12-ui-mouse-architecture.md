# UI Mouse Architecture Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace scattered manual click-coordinate handling with a component-oriented mouse routing architecture that remains stable under resize and keeps page logic cohesive.

**Architecture:** Keep Bubble Tea/Bubbles as the rendering foundation, but move mouse capture to a central router plus page-level interaction specs. Interactive regions become page-owned registrations with stable action identifiers, and the app model dispatches actions instead of directly mutating page state from raw coordinates. Migrate incrementally so every step compiles and preserves behavior.

**Tech Stack:** Go, Bubble Tea, Bubbles (`list`, `table`, `viewport`, `textinput`), Lip Gloss, Go `testing`

---

### Task 1: Introduce Mouse Action Types And Router Skeleton

**Files:**
- Create: `internal/ui/mouse_router.go`
- Test: `internal/ui/mouse_router_test.go`

- [ ] **Step 1: Write the failing test**

```go
package ui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestMouseRouterDispatchesHighestPriorityAction(t *testing.T) {
	r := mouseRouter{
		handlers: []mouseHandler{
			{id: "page.tab", area: hitBox{x1: 0, y1: 0, x2: 10, y2: 1}},
			{id: "overlay.close", area: hitBox{x1: 2, y1: 0, x2: 6, y2: 1}},
		},
	}

	got, ok := r.dispatch(tea.MouseMsg{X: 3, Y: 0, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress})
	if !ok {
		t.Fatalf("expected dispatch hit")
	}
	if got != "overlay.close" {
		t.Fatalf("expected overlay.close, got %q", got)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/ui -run TestMouseRouterDispatchesHighestPriorityAction -count=1`
Expected: FAIL with undefined `mouseRouter`, `mouseHandler`, or `hitBox`

- [ ] **Step 3: Write minimal implementation**

```go
package ui

import tea "github.com/charmbracelet/bubbletea"

type hitBox struct {
	x1 int
	y1 int
	x2 int
	y2 int
}

func (b hitBox) contains(x, y int) bool {
	return x >= b.x1 && x < b.x2 && y >= b.y1 && y < b.y2
}

type mouseHandler struct {
	id   string
	area hitBox
}

type mouseRouter struct {
	handlers []mouseHandler
}

func (r mouseRouter) dispatch(msg tea.MouseMsg) (string, bool) {
	for i := len(r.handlers) - 1; i >= 0; i-- {
		h := r.handlers[i]
		if h.area.contains(msg.X, msg.Y) {
			return h.id, true
		}
	}
	return "", false
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/ui -run TestMouseRouterDispatchesHighestPriorityAction -count=1`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/ui/mouse_router.go internal/ui/mouse_router_test.go
git commit -m "refactor(ui): add mouse router foundation"
```

### Task 2: Move Header And Global Interactions To Registered Actions

**Files:**
- Modify: `internal/ui/app.go`
- Modify: `internal/ui/input_global_mouse.go`
- Modify: `internal/ui/views_shell.go`
- Modify: `internal/ui/mouse_router.go`
- Test: `internal/ui/mouse_test.go`

- [ ] **Step 1: Write the failing test**

```go
func TestHandleMouseMsgDispatchesHeaderAction(t *testing.T) {
	m := NewModel(config.Default(), nil)
	m.width = 120
	m.height = 32
	_ = m.renderHeader(m.width)
	cmd := m.handleMouseMsg(tea.MouseMsg{
		X:      95,
		Y:      0,
		Button: tea.MouseButtonLeft,
		Action: tea.MouseActionPress,
	})
	if cmd != nil {
		t.Fatalf("expected header click to be handled without tea cmd")
	}
	if m.cfg.Language != "en" {
		t.Fatalf("expected language toggle to switch to en, got %q", m.cfg.Language)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/ui -run TestHandleMouseMsgDispatchesHeaderAction -count=1`
Expected: FAIL because header click routing is still based on direct target checks or because the click point is not registered via the router

- [ ] **Step 3: Write minimal implementation**

```go
// add to model
mouse mouseRouter

// before rendering interactive shell
m.mouse.reset()
m.mouse.register("header.language.toggle", hitBox{...})

// handleMouseMsg
if id, ok := m.mouse.dispatch(msg); ok {
	return m.handleMouseAction(id)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/ui -run TestHandleMouseMsgDispatchesHeaderAction -count=1`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/ui/app.go internal/ui/input_global_mouse.go internal/ui/views_shell.go internal/ui/mouse_router.go internal/ui/mouse_test.go
git commit -m "refactor(ui): route header mouse actions centrally"
```

### Task 3: Convert Overview And Proxy Pages To Page-Owned Mouse Actions

**Files:**
- Create: `internal/ui/page_overview.go`
- Create: `internal/ui/page_network.go`
- Modify: `internal/ui/views_overview.go`
- Modify: `internal/ui/views_network.go`
- Modify: `internal/ui/input_tabs.go`
- Modify: `internal/ui/input_network.go`
- Modify: `internal/ui/mouse_router.go`
- Test: `internal/ui/mouse_test.go`

- [ ] **Step 1: Write the failing test**

```go
func TestProxyWheelScrollDoesNotSelectNode(t *testing.T) {
	m := newMouseReadyProxyModel()
	before := m.nodeCursor
	_ = m.handleMouseMsg(tea.MouseMsg{
		X:      40,
		Y:      10,
		Button: tea.MouseButtonWheelDown,
		Action: tea.MouseActionPress,
	})
	if m.nodeCursor != before {
		t.Fatalf("expected wheel to only scroll, cursor changed from %d to %d", before, m.nodeCursor)
	}
	if m.nodeOffset == 0 {
		t.Fatalf("expected wheel to scroll visible window")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/ui -run TestProxyWheelScrollDoesNotSelectNode -count=1`
Expected: FAIL if wheel behavior still couples hit-testing and selection

- [ ] **Step 3: Write minimal implementation**

```go
type pageMouseAction struct {
	id   string
	data string
}

func (m *model) registerProxyNodeAction(idx int, area hitBox) {
	m.mouse.registerWithData("proxy.node.select", strconv.Itoa(idx), area)
}

func (m *model) handleMouseAction(action mouseAction) tea.Cmd {
	switch action.ID {
	case "overview.mode":
		return tea.Batch(setModeCmd(m.client, action.Data), fetchConfigCmd(m.client))
	case "proxy.group.select":
		// update cursor only
	case "proxy.node.select":
		// set cursor and issue switch cmd
	}
	return nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/ui -run TestProxyWheelScrollDoesNotSelectNode -count=1`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/ui/page_overview.go internal/ui/page_network.go internal/ui/views_overview.go internal/ui/views_network.go internal/ui/input_tabs.go internal/ui/input_network.go internal/ui/mouse_router.go internal/ui/mouse_test.go
git commit -m "refactor(ui): move page mouse logic behind actions"
```

### Task 4: Isolate Overlay Components And Remove Legacy Target Arrays

**Files:**
- Create: `internal/ui/overlay_selector.go`
- Create: `internal/ui/overlay_palette.go`
- Modify: `internal/ui/selector.go`
- Modify: `internal/ui/palette_and_settings.go`
- Modify: `internal/ui/app.go`
- Modify: `internal/ui/input_global_mouse.go`
- Test: `internal/ui/selector_test.go`

- [ ] **Step 1: Write the failing test**

```go
func TestSelectorOutsideClickHandledByOverlayRouter(t *testing.T) {
	m := NewModel(config.Default(), nil)
	m.width = 100
	m.height = 30
	m.openSelector("theme")
	_ = m.renderSelectorOverlay(36, 10)
	handled, _ := m.handleSelectorMouse(0, 0)
	if !handled {
		t.Fatalf("expected outside click to be handled by selector overlay")
	}
	if m.selectorOpen {
		t.Fatalf("expected selector closed")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/ui -run TestSelectorOutsideClickHandledByOverlayRouter -count=1`
Expected: FAIL after removing legacy target arrays and before overlay router is in place

- [ ] **Step 3: Write minimal implementation**

```go
type selectorOverlay struct{}

func (selectorOverlay) register(m *model, w, h int) {
	m.mouse.pushScope("selector")
	m.mouse.register("selector.dismiss", hitBox{...})
	m.mouse.register("selector.choose", hitBox{...})
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/ui -run TestSelectorOutsideClickHandledByOverlayRouter -count=1`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/ui/overlay_selector.go internal/ui/overlay_palette.go internal/ui/selector.go internal/ui/palette_and_settings.go internal/ui/app.go internal/ui/input_global_mouse.go internal/ui/selector_test.go
git commit -m "refactor(ui): isolate overlays behind router scopes"
```

### Task 5: Clean Up And Verify Full UI Behavior

**Files:**
- Modify: `docs/FILE_INDEX.md`
- Modify: `docs/IMPLEMENTATION.md`
- Modify: `internal/ui/app.go`
- Modify: `internal/ui/input_global_mouse.go`
- Modify: `internal/ui/input_network.go`
- Modify: `internal/ui/input_tabs.go`
- Test: `internal/ui/mouse_test.go`
- Test: `internal/ui/selector_test.go`

- [ ] **Step 1: Write the failing test**

```go
func TestMouseRouterNoLegacyTargetsNeeded(t *testing.T) {
	m := NewModel(config.Default(), nil)
	m.width = 120
	m.height = 32
	_ = m.View()
	if len(m.mainTabTargets) != 0 {
		t.Fatalf("expected legacy target arrays removed from active path")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/ui -run TestMouseRouterNoLegacyTargetsNeeded -count=1`
Expected: FAIL because legacy click target slices are still populated

- [ ] **Step 3: Write minimal implementation**

```go
// remove legacy target fields and replace remaining references with router registrations
// update docs to reflect mouse architecture and page-owned actions
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/ui -run TestMouseRouterNoLegacyTargetsNeeded -count=1`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add docs/FILE_INDEX.md docs/IMPLEMENTATION.md internal/ui/app.go internal/ui/input_global_mouse.go internal/ui/input_network.go internal/ui/input_tabs.go internal/ui/mouse_test.go internal/ui/selector_test.go
git commit -m "refactor(ui): remove legacy mouse target arrays"
```
