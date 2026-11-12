package netid

import (
	"crypto/sha1"
	"deepmole/internal/i18n"
	"encoding/hex"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"syscall"

	"golang.org/x/sys/windows"
)

type Network struct {
	ID      string
	Name    string
	Gateway string
	MAC     string
	SSID    string
}

func (n Network) Same(o Network) bool {
	return n.Gateway != "" && n.Gateway == o.Gateway && n.SSID == o.SSID && (n.MAC == "" || o.MAC == "" || n.MAC == o.MAC)
}

func run(name string, args ...string) string {
	cmd := exec.Command(name, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: windows.CREATE_NO_WINDOW}
	out, _ := cmd.Output()
	return string(out)
}

var ipRe = regexp.MustCompile(`^\d+\.\d+\.\d+\.\d+$`)

func gateway() string {
	best, bestMetric := "", 1<<30
	for _, line := range strings.Split(run("route", "print", "-4", "0.0.0.0"), "\n") {
		f := strings.Fields(line)
		if len(f) < 5 || f[0] != "0.0.0.0" || f[1] != "0.0.0.0" || !ipRe.MatchString(f[2]) {
			continue
		}
		m, err := strconv.Atoi(f[4])
		if err != nil {
			continue
		}
		if m < bestMetric {
			best, bestMetric = f[2], m
		}
	}
	return best
}

var macRe = regexp.MustCompile(`([0-9a-fA-F]{2}[-:]){5}[0-9a-fA-F]{2}`)

func mac(gw string) string {
	for _, line := range strings.Split(run("arp", "-a", gw), "\n") {
		if strings.Contains(line, gw+" ") || strings.HasPrefix(strings.TrimSpace(line), gw) {
			if m := macRe.FindString(line); m != "" {
				return strings.ToLower(strings.ReplaceAll(m, ":", "-"))
			}
		}
	}
	return ""
}

var ssidRe = regexp.MustCompile(`(?m)^\s*SSID\s*:\s*(.+?)\s*$`)

func ssid() string {
	if m := ssidRe.FindStringSubmatch(run("netsh", "wlan", "show", "interfaces")); len(m) == 2 {
		return m[1]
	}
	return ""
}

func Current() Network {
	gw := gateway()
	if gw == "" {
		return Network{ID: "offline", Name: i18n.T("net.none")}
	}
	m := mac(gw)
	s := ssid()
	h := sha1.Sum([]byte(gw + "|" + m + "|" + s))
	name := s
	if name == "" {
		name = i18n.T("net.lan", gw)
	}
	return Network{ID: hex.EncodeToString(h[:])[:10], Name: name, Gateway: gw, MAC: m, SSID: s}
}
