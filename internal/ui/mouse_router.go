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

type mouseAction struct {
	ID    string
	Index int
	Text  string
	Box   hitBox
}

type mouseRouter struct {
	actions []mouseAction
}

func (r *mouseRouter) reset() {
	r.actions = r.actions[:0]
}

func (r *mouseRouter) register(action mouseAction) {
	r.actions = append(r.actions, action)
}

func (r mouseRouter) dispatch(msg tea.MouseMsg) (mouseAction, bool) {
	for i := len(r.actions) - 1; i >= 0; i-- {
		action := r.actions[i]
		if action.Box.contains(msg.X, msg.Y) {
			return action, true
		}
	}
	return mouseAction{}, false
}
