// Package histwalk implements the history walking addon.
package histwalk

import (
	"errors"
	"fmt"

	"github.com/elves/elvish/cli"
	"github.com/elves/elvish/cli/el"
	"github.com/elves/elvish/cli/el/codearea"
	"github.com/elves/elvish/cli/el/layout"
	"github.com/elves/elvish/cli/histutil"
	"github.com/elves/elvish/cli/term"
	"github.com/elves/elvish/edit/ui"
)

var ErrHistWalkInactive = errors.New("the histwalk addon is not active")

// Config keeps the configuration for the histwalk addon.
type Config struct {
	// Keybinding.
	Binding el.Handler
	// The history walker.
	Walker histutil.Walker
}

type widget struct {
	app cli.App
	Config
}

func (w *widget) Render(width, height int) *ui.Buffer {
	content := layout.ModeLine(
		fmt.Sprintf(" HISTORY #%d ", w.Walker.CurrentSeq()), false)
	buf := ui.NewBufferBuilder(width).WriteStyled(content).Buffer()
	buf.TrimToLines(0, height)
	return buf
}

func (w *widget) Handle(event term.Event) bool {
	handled := w.Binding.Handle(event)
	if handled {
		return true
	}
	Accept(w.app)
	return w.app.CodeArea().Handle(event)
}

func (w *widget) Focus() bool { return false }

func (w *widget) onWalk() {
	prefix := w.Walker.Prefix()
	w.app.CodeArea().MutateState(func(s *codearea.State) {
		s.Pending = codearea.Pending{
			From: len(prefix), To: len(s.Buffer.Content),
			Content: w.Walker.CurrentCmd()[len(prefix):],
		}
	})
}

// Start starts the histwalk addon.
func Start(app cli.App, cfg Config) {
	if cfg.Walker == nil {
		app.Notify("no history walker")
		return
	}
	if cfg.Binding == nil {
		cfg.Binding = el.DummyHandler{}
	}
	walker := cfg.Walker
	walker.Prev()
	w := widget{app: app, Config: cfg}
	w.onWalk()
	app.MutateState(func(s *cli.State) { s.Addon = &w })
	app.Redraw()
}

// Prev walks to the previous entry in history. It returns ErrHistWalkInactive
// if the histwalk addon is not active.
func Prev(app cli.App) error {
	return walk(app, func(w *widget) error { return w.Walker.Prev() })
}

// Next walks to the next entry in history. It returns ErrHistWalkInactive if
// the histwalk addon is not active.
func Next(app cli.App) error {
	return walk(app, func(w *widget) error { return w.Walker.Next() })
}

// Close closes the histwalk addon. It does nothing if the histwalk addon is not
// active.
func Close(app cli.App) {
	if closeAddon(app) {
		app.CodeArea().MutateState(func(s *codearea.State) {
			s.Pending = codearea.Pending{}
		})
	}
}

// Accept closes the histwalk addon, accepting the current shown command. It does
// nothing if the histwalk addon is not active.
func Accept(app cli.App) {
	if closeAddon(app) {
		app.CodeArea().MutateState(func(s *codearea.State) {
			s.ApplyPending()
		})
	}
}

func closeAddon(app cli.App) bool {
	var closed bool
	app.MutateState(func(s *cli.State) {
		if _, ok := s.Addon.(*widget); !ok {
			return
		}
		s.Addon = nil
		closed = true
	})
	return closed
}

func walk(app cli.App, f func(*widget) error) error {
	w, ok := getWidget(app)
	if !ok {
		return ErrHistWalkInactive
	}
	err := f(w)
	if err == nil {
		w.onWalk()
	}
	return err
}

func getWidget(app cli.App) (*widget, bool) {
	w, ok := app.CopyState().Addon.(*widget)
	return w, ok
}
