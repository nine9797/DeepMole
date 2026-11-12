package sdns

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"unicode/utf16"
)

func TestAdapters(t *testing.T) {
	ads, err := adapters()
	if err != nil {
		t.Skip(err)
	}
	for _, a := range ads {
		t.Logf("%s %s v6=%v v6dns=%v static4=%v", a.guid, a.name, a.v6, a.v6dns, staticServers(tcpipKey, a.guid))
	}
}

func TestApplyRestoreLive(t *testing.T) {
	if os.Getenv("DEEPMOLE_LIVE_DNS") == "" {
		t.Skip("set DEEPMOLE_LIVE_DNS=1 and run as administrator")
	}
	path := filepath.Join(t.TempDir(), "dns.json")
	before := dump(t)
	p := Find(os.Getenv("DEEPMOLE_LIVE_DNS_PROVIDER"))
	if err := Apply(p, path, os.Getenv("DEEPMOLE_LIVE_DNS_DOH") != ""); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(path)
	t.Logf("backup:\n%s", data)
	t.Logf("after apply:\n%s", dump(t))
	if err := Restore(path); err != nil {
		t.Fatal(err)
	}
	after := dump(t)
	t.Logf("after restore:\n%s", after)
	if Pending(path) {
		t.Fatal("backup not removed")
	}
	if before != after {
		t.Fatal("settings differ after restore")
	}
}

func TestDohActive(t *testing.T) {
	if os.Getenv("DEEPMOLE_LIVE_DNS") == "" {
		t.Skip("set DEEPMOLE_LIVE_DNS=1 and run as administrator")
	}
	dir := t.TempDir()
	capture := func(label string) int {
		etl := filepath.Join(dir, label+".etl")
		txt := filepath.Join(dir, label+".txt")
		_, _ = exec.Command("pktmon", "filter", "remove").CombinedOutput()
		_, _ = exec.Command("pktmon", "filter", "add", "-p", "53").CombinedOutput()
		if out, err := exec.Command("pktmon", "start", "--capture", "--pkt-size", "0", "-f", etl).CombinedOutput(); err != nil {
			t.Fatalf("pktmon start: %v %s", err, out)
		}
		_, _ = exec.Command("ipconfig", "/flushdns").CombinedOutput()
		for _, h := range []string{"www.youtube.com", "discord.com", "example.org"} {
			_, _ = exec.Command("powershell", "-NoProfile", "-Command", "Resolve-DnsName "+h+" -Type A").CombinedOutput()
		}
		_, _ = exec.Command("pktmon", "stop").CombinedOutput()
		_, _ = exec.Command("pktmon", "filter", "remove").CombinedOutput()
		_, _ = exec.Command("pktmon", "etl2txt", etl, "-o", txt).CombinedOutput()
		raw, _ := os.ReadFile(txt)
		u16 := make([]uint16, 0, len(raw)/2)
		for i := 0; i+1 < len(raw); i += 2 {
			u16 = append(u16, uint16(raw[i])|uint16(raw[i+1])<<8)
		}
		n := 0
		for _, l := range strings.Split(string(utf16.Decode(u16)), "\n") {
			if strings.Contains(l, ".53: ") {
				n++
			}
		}
		t.Logf("%s: %d packets on port 53", label, n)
		return n
	}
	before := capture("before")
	path := filepath.Join(t.TempDir(), "dns.json")
	if err := Apply(Find("Google"), path, true); err != nil {
		t.Fatal(err)
	}
	with := capture("with doh")
	if err := Restore(path); err != nil {
		t.Fatal(err)
	}
	after := capture("after restore")
	if before == 0 || after == 0 {
		t.Skip("no plain DNS traffic captured, cannot judge")
	}
	if with != 0 {
		t.Fatalf("plain DNS still used with DoH enabled: %d packets", with)
	}
}

func TestResolvesLive(t *testing.T) {
	if os.Getenv("DEEPMOLE_LIVE_DNS") == "" {
		t.Skip("set DEEPMOLE_LIVE_DNS=1 and run as administrator")
	}
	look := func(label string) {
		_, _ = exec.Command("ipconfig", "/flushdns").CombinedOutput()
		for _, h := range []string{"ya.ru", "example.org"} {
			start := time.Now()
			ips, err := net.DefaultResolver.LookupIPAddr(context.Background(), h)
			t.Logf("%s: go %s -> %v %v (%s)", label, h, ips, err, time.Since(start).Round(time.Millisecond))
		}
		out, _ := exec.Command("powershell", "-NoProfile", "-Command", "Resolve-DnsName ya.ru -Type A -DnsOnly | Select-Object -First 1 | Format-List").CombinedOutput()
		t.Logf("%s: powershell %s", label, strings.TrimSpace(string(out)))
	}
	look("before")
	path := filepath.Join(t.TempDir(), "dns.json")
	for _, doh := range []bool{true, false} {
		if err := Apply(Find("Google"), path, doh); err != nil {
			t.Fatal(err)
		}
		look(fmt.Sprintf("google doh=%v", doh))
		if err := Restore(path); err != nil {
			t.Fatal(err)
		}
	}
	look("restored")
}

func dump(t *testing.T) string {
	out, err := exec.Command("netsh", "interface", "ipv4", "show", "dnsservers").CombinedOutput()
	if err != nil {
		t.Fatal(err)
	}
	reg, _ := exec.Command("reg", "query", `HKLM\SYSTEM\CurrentControlSet\Services\Dnscache\InterfaceSpecificParameters`, "/s").CombinedOutput()
	var doh []string
	for _, l := range strings.Split(string(reg), "\n") {
		if strings.Contains(l, "DohInterfaceSettings") || strings.Contains(l, "DohFlags") {
			doh = append(doh, strings.TrimSpace(l))
		}
	}
	return string(out) + strings.Join(doh, "\n")
}
