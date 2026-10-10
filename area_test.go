package progkit

import (
	"encoding/json"
	"testing"

	"github.com/0magnet/websh/progressive"
	"github.com/gdamore/tcell/v3"
)

func keys(a *Area, evs ...*tcell.EventKey) {
	for _, ev := range evs {
		a.Key(ev)
	}
}

func typed(s string) *tcell.EventKey      { return tcell.NewEventKey(tcell.KeyRune, s, 0) }
func special(k tcell.Key) *tcell.EventKey { return tcell.NewEventKey(k, "", 0) }

func TestAreaEdits(t *testing.T) {
	changes := 0
	a := &Area{OnChange: func() { changes++ }}
	a.SetText("ab\ncd")
	keys(a, special(tcell.KeyDown), special(tcell.KeyEnd), typed("e"), special(tcell.KeyEnter), typed("f"))
	if got := a.Text(); got != "ab\ncde\nf" {
		t.Fatalf("text = %q", got)
	}
	keys(a, special(tcell.KeyHome), special(tcell.KeyBackspace))
	if got := a.Text(); got != "ab\ncdef" {
		t.Fatalf("backspace at line start = %q", got)
	}
	if l, c := a.Cursor(); l != 1 || c != 3 {
		t.Fatalf("cursor = %d,%d want 1,3", l, c)
	}
	keys(a, special(tcell.KeyUp), special(tcell.KeyEnd), special(tcell.KeyDelete))
	if got := a.Text(); got != "abcdef" {
		t.Fatalf("delete at line end = %q", got)
	}
	keys(a, special(tcell.KeyHome), special(tcell.KeyBackspace))
	if got := a.Text(); got != "abcdef" || changes != 5 {
		t.Fatalf("backspace at the start = %q after %d changes", got, changes)
	}
	a.Paste("x\r\ny")
	if got := a.Text(); got != "x\nyabcdef" {
		t.Fatalf("paste = %q", got)
	}
}

func TestAreaDrawsTabsAndScrolls(t *testing.T) {
	app, _ := testApp(t, nil)
	a := &Area{}
	a.SetText("\tx\n1\n2\n3\n4")
	app.Draw(func(f *Frame) { a.Draw(f, Rect{0, 0, 10, 3}, true) })
	if got := row(app.Screen, 0, 10); got != "    x" {
		t.Errorf("tab row = %q", got)
	}
	keys(a, special(tcell.KeyPgDn), special(tcell.KeyPgDn))
	app.Draw(func(f *Frame) { a.Draw(f, Rect{0, 0, 10, 3}, true) })
	if got := row(app.Screen, 2, 10); got != "4" {
		t.Errorf("after paging, last row = %q", got)
	}
}

func TestAreaFollowsThePage(t *testing.T) {
	app, out := testApp(t, websh)
	a := &Area{ID: "ed"}
	a.SetText("one")
	app.Draw(func(f *Frame) { a.Draw(f, Rect{0, 0, 20, 4}, true) })
	if out.Len() == 0 {
		t.Fatal("no placement shipped")
	}
	data, err := json.Marshal(Msg{Type: "change", Value: "one\ntwo", Index: 6})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := app.Deliver(&progressive.Event{ID: "ed", Type: "message", Data: data}); !ok {
		t.Fatal("change not delivered")
	}
	if l, c := a.Cursor(); a.Text() != "one\ntwo" || l != 1 || c != 2 {
		t.Errorf("text %q cursor %d,%d", a.Text(), l, c)
	}
}
