package runner

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

func driverState() string {
	out, _ := exec.Command("sc", "query", "WinDivert").CombinedOutput()
	for _, l := range strings.Split(string(out), "\n") {
		if strings.Contains(l, "STATE") {
			return strings.TrimSpace(l)
		}
	}
	return strings.TrimSpace(string(out))
}

func TestUnloadDriverLive(t *testing.T) {
	if os.Getenv("DEEPMOLE_LIVE_DRIVER") == "" {
		t.Skip("set DEEPMOLE_LIVE_DRIVER=1 and run as administrator")
	}
	t.Logf("before: %s", driverState())
	UnloadDriver()
	after := driverState()
	t.Logf("after: %s", after)
	if strings.Contains(after, "RUNNING") {
		t.Fatal("driver still running")
	}
}
