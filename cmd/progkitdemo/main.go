// Command progkitdemo shows each progkit widget: a list, a text field and a
// button that add to a log, and a spinner. In websh the field, button, list
// and log are the page's own elements laid over the cells; in any other
// terminal they are cells.
package main

import (
	"fmt"
	"os"
	"time"

	"github.com/0magnet/calvin/clihelp"
	"github.com/gdamore/tcell/v3"
	"github.com/spf13/cobra"

	"github.com/0magnet/progkit"
)

func main() {
	cmd := &cobra.Command{
		Use:          "progkitdemo",
		Short:        "show each progkit widget",
		SilenceUsage: true,
		Run:          func(*cobra.Command, []string) { demo() },
	}
	clihelp.Init(cmd, "progkitdemo")
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func demo() {
	app, err := progkit.Open()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	var (
		spin  progkit.Spinner
		focus int
		logv  = &progkit.Text{ID: "log", Follow: true, Selectable: true}
		say   = func(s string) {
			logv.Append(progkit.ParseANSI(time.Now().Format("15:04:05 ")+s, tcell.StyleDefault)...)
		}
		list  = &progkit.List{ID: "list", Items: []string{"dmsg", "skynet", "stcpr", "sudph", "webrtc"}}
		input = &progkit.Input{ID: "input", Placeholder: "type and press Enter"}
		btn   = &progkit.Button{ID: "button", Label: "Add"}
	)
	list.OnActivate = func(i int) { say("chose \x1b[36m" + list.Items[i] + "\x1b[0m") }
	input.OnSubmit = func(v string) {
		if v != "" {
			list.Items = append(list.Items, v)
			say("added \x1b[32m" + v + "\x1b[0m")
			input.SetValue("")
		}
	}
	btn.OnPress = func() { input.OnSubmit(input.Value()) }
	if c := app.Caps(); c != nil {
		say(fmt.Sprintf("host %s, trust %s, placing elements: %v", c.Host, c.Trust, app.Placing()))
	} else {
		say("this terminal draws cells only")
	}

	go func() {
		for range time.Tick(120 * time.Millisecond) {
			spin.Tick()
			app.Redraw()
		}
	}()

	app.Run(func(f *progkit.Frame) {
		all := f.Size()
		head, body := all.SplitTop(1)
		body, foot := body.SplitBottom(1)
		progkit.DrawText(f.Screen, 0, head.Y, head.W, spin.Frame()+" progkit demo  (Tab: next, Esc: quit)", tcell.StyleDefault.Bold(true))
		left, right := body.SplitLeft(24)
		inner := progkit.Box(f.Screen, left, "list", tcell.StyleDefault)
		list.Draw(f, inner, focus == 0)
		inner = progkit.Box(f.Screen, right, "log", tcell.StyleDefault)
		logv.Draw(f, inner)
		field, button := foot.SplitLeft(max(foot.W-10, 0))
		input.Draw(f, field, focus == 1)
		btn.Draw(f, button, focus == 2)
	}, func(ev tcell.Event) bool {
		switch ev := ev.(type) {
		case *tcell.EventKey:
			switch {
			case ev.Key() == tcell.KeyEscape || progkit.IsCtrl(ev, 'c'):
				return false
			case ev.Key() == tcell.KeyTab:
				focus = (focus + 1) % 3
				return true
			}
			switch focus {
			case 0:
				list.Key(ev)
			case 1:
				input.Key(ev)
			case 2:
				btn.Key(ev)
			}
		case *tcell.EventMouse:
			logv.Mouse(ev)
		}
		return true
	})
	app.Close()
	os.Exit(0)
}
