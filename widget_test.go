package progkit

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"testing"
)

// The shipped document's script must at least parse: an error there leaves
// every placement empty and transparent, which looks like it is not placed.
// A browser global such as top cannot be redeclared at the top level.
func TestWidgetScriptParses(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node not installed")
	}
	m := regexp.MustCompile(`(?s)<script>(.*)</script>`).FindSubmatch(widgetHTML)
	if m == nil {
		t.Fatal("no script in widget.html")
	}
	// Run it as a classic script in a scope whose globals match a browser's.
	js := "const top = {};\n" + string(m[1])
	p := filepath.Join(t.TempDir(), "w.js")
	if err := os.WriteFile(p, []byte(js), 0o600); err != nil {
		t.Fatal(err)
	}
	if out, err := exec.Command(node, "--check", p).CombinedOutput(); err != nil { //nolint:gosec // node from PATH, file from TempDir
		t.Fatalf("widget.html script does not parse: %v\n%s", err, out)
	}
}
