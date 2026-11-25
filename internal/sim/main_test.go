package sim

import (
	"os"
	"testing"
)

// TestMain runs the tests on the full game even in a demo build (-tags
// demo): they are about the rules, and the demo's own test turns it on.
func TestMain(m *testing.M) {
	SetDemo(false)
	os.Exit(m.Run())
}
