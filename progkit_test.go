package progkit

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/0magnet/websh/progressive"
	"github.com/gdamore/tcell/v3"
	"github.com/gdamore/tcell/v3/color"
	"github.com/gdamore/tcell/v3/vt"
)

func testApp(t *testing.T, caps *progressive.Caps) (*App, *bytes.Buffer) {
	t.Helper()
	sc, err := tcell.NewTerminfoScreenFromTty(vt.NewMockTerm(vt.MockOptSize{X: 40, Y: 10}))
	if err != nil {
		t.Fatal(err)
	}
	if err := sc.Init(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(sc.Fini)
	var out bytes.Buffer
	return NewApp(sc, &out, caps), &out
}

func row(sc tcell.Screen, y, w int) string {
	var b strings.Builder
	for x := 0; x < w; x++ {
		s, _, _ := sc.Get(x, y)
		b.WriteString(s)
	}
	return strings.TrimRight(b.String(), " ")
}

var websh = &progressive.Caps{V: 1, Host: "websh", Trust: "local", Features: []string{"ship", "place.input", "event", "post"}}

func TestCellsWithoutHost(t *testing.T) {
	a, out := testApp(t, nil)
	in := &Input{ID: "q", Placeholder: "search"}
	in.SetValue("hello")
	l := &List{ID: "l", Items: []string{"one", "two"}, Selected: 1}
	a.Draw(func(f *Frame) {
		in.Draw(f, Rect{0, 0, 20, 1}, true)
		l.Draw(f, Rect{0, 1, 20, 2}, true)
		Box(f.Screen, Rect{0, 4, 12, 3}, "t", tcell.StyleDefault)
	})
	if got := row(a.Screen, 0, 20); got != "hello" {
		t.Errorf("input row = %q", got)
	}
	if got := row(a.Screen, 2, 20); got != "two" {
		t.Errorf("list row = %q", got)
	}
	if got := row(a.Screen, 4, 12); got != "┌─ t ──────┐" {
		t.Errorf("box top = %q", got)
	}
	if out.Len() != 0 {
		t.Errorf("a host that offers nothing was sent %q", out.String())
	}
}

func TestPlacementsFollowFrames(t *testing.T) {
	a, out := testApp(t, websh)
	in := &Input{ID: "q"}
	b := &Button{ID: "go", Label: "Go"}
	a.Draw(func(f *Frame) {
		in.Draw(f, Rect{0, 0, 20, 1}, true)
		b.Draw(f, Rect{0, 1, 20, 1}, false)
	})
	s := out.String()
	if !strings.Contains(s, "\x1b]7337;ship;progkit;") {
		t.Fatal("the widget document was not shipped")
	}
	if strings.Count(s, "\x1b]7337;place;") != 2 || strings.Count(s, "\x1b]7337;post;") != 2 {
		t.Fatalf("want two placements and two posts, got %q", s)
	}

	out.Reset()
	a.Draw(func(f *Frame) {
		in.Draw(f, Rect{0, 0, 20, 1}, true)
		b.Draw(f, Rect{0, 1, 20, 1}, false)
	})
	if out.Len() != 0 {
		t.Errorf("an unchanged frame sent %q", out.String())
	}

	out.Reset()
	in.SetValue("x")
	a.Draw(func(f *Frame) { in.Draw(f, Rect{0, 0, 20, 1}, true) })
	s = out.String()
	if strings.Contains(s, "place;") || !strings.Contains(s, "post;q;") || !strings.Contains(s, "remove;go") {
		t.Errorf("want a post to q and go removed, got %q", s)
	}
}

func TestMessagesReachTheWidget(t *testing.T) {
	a, _ := testApp(t, websh)
	var submitted string
	in := &Input{ID: "q", OnSubmit: func(v string) { submitted = v }}
	a.Draw(func(f *Frame) { in.Draw(f, Rect{0, 0, 20, 1}, true) })

	data, err := json.Marshal(Msg{Type: "submit", Value: "typed in the page"})
	if err != nil {
		t.Fatal(err)
	}
	m, ok := a.Deliver(&progressive.Event{ID: "q", Type: "message", Data: data})
	if !ok || m.Msg.Type != "submit" {
		t.Fatalf("Deliver = %+v, %v", m, ok)
	}
	if submitted != "typed in the page" || in.Value() != "typed in the page" {
		t.Errorf("submitted %q, value %q", submitted, in.Value())
	}
}

func TestInputKeys(t *testing.T) {
	in := &Input{}
	for _, ev := range []*tcell.EventKey{
		tcell.NewEventKey(tcell.KeyRune, "a", 0),
		tcell.NewEventKey(tcell.KeyRune, "b", 0),
		tcell.NewEventKey(tcell.KeyLeft, "", 0),
		tcell.NewEventKey(tcell.KeyRune, "x", 0),
		tcell.NewEventKey(tcell.KeyRune, "u", tcell.ModCtrl),
	} {
		in.Key(ev)
	}
	if in.Value() != "b" {
		t.Errorf("value = %q, want b", in.Value())
	}
}

func TestParseANSI(t *testing.T) {
	base := tcell.StyleDefault
	lines := ParseANSI("plain \x1b[31mred\x1b[0m\n\x1b[1;38;5;208mbold\x1b[m\tx\x1b]8;;u\x1b\\", base)
	if len(lines) != 2 {
		t.Fatalf("got %d lines", len(lines))
	}
	if lines[0].Plain() != "plain red" || lines[1].Plain() != "bold    x" {
		t.Errorf("text = %q / %q", lines[0].Plain(), lines[1].Plain())
	}
	if lines[0][1].Style.GetForeground() != color.PaletteColor(1) {
		t.Error("red span is not red")
	}
	st := lines[1][0].Style
	if st.GetForeground() != color.PaletteColor(208) || st.GetAttributes()&tcell.AttrBold == 0 {
		t.Error("256-color bold span lost its style")
	}
}

func TestTextFollows(t *testing.T) {
	a, _ := testApp(t, nil)
	tx := &Text{Follow: true}
	tx.SetANSI("a\nb\nc\nd\ne")
	a.Draw(func(f *Frame) { tx.Draw(f, Rect{0, 0, 10, 2}) })
	tx.Append(Line{{"f", tcell.StyleDefault}})
	a.Draw(func(f *Frame) { tx.Draw(f, Rect{0, 0, 10, 2}) })
	if got := row(a.Screen, 1, 10); got != "f" {
		t.Errorf("last row = %q, want the newest line", got)
	}
}

func TestElementPassesKeysOn(t *testing.T) {
	if k := (Msg{Type: "key", Value: "Escape"}).Key(); k == nil || k.Key() != tcell.KeyEscape {
		t.Errorf("Escape from an element = %v", k)
	}
	if k := (Msg{Type: "change", Value: "Escape"}).Key(); k != nil {
		t.Error("a change message became a key")
	}
	if k := (Msg{Type: "key", Value: "q"}).Key(); k == nil || Typed(k) != "q" {
		t.Errorf("q from an element = %v", k)
	}
	if k := (Msg{Type: "key", Value: "ArrowDown"}).Key(); k == nil || k.Key() != tcell.KeyDown {
		t.Errorf("ArrowDown from an element = %v", k)
	}
	if k := (Msg{Type: "key", Value: "Ctrl+s"}).Key(); k == nil || !IsCtrl(k, 's') {
		t.Errorf("Ctrl+s from an element = %v", k)
	}
	if k := (Msg{Type: "key", Value: "Shift"}).Key(); k != nil {
		t.Error("a lone modifier became a key")
	}
}

// A goroutine that redraws after the program has quit must not panic.
func TestRedrawAfterClose(t *testing.T) {
	a, _ := testApp(t, nil)
	a.Close()
	a.Redraw()
}
