package strategy

import (
	"strings"
	"testing"
)

func TestTelegramRender(t *testing.T) {
	for _, p := range ProvenTelegram {
		args := strings.Join(p.Render(testPaths, true), " ")
		if !strings.Contains(args, "--dpi-desync-any-protocol=1") || !strings.Contains(args, "--dpi-desync-cutoff=n3") {
			t.Errorf("%s: any-protocol missing: %s", p.Name(), args)
		}
		for _, bad := range []string{"fake-tls", "fake-http", "seqovl", "hostfakesplit"} {
			if strings.Contains(args, bad) {
				t.Errorf("%s: unexpected %s in %s", p.Name(), bad, args)
			}
		}
		if !strings.Contains(p.Name(), "raw") {
			t.Errorf("%s: name lacks raw marker", p.Name())
		}
	}
}

func TestTelegramNeighbors(t *testing.T) {
	seen := map[string]bool{}
	for _, p := range ProvenTelegram {
		for _, q := range Neighbors(p) {
			if !q.Raw {
				t.Fatalf("neighbor of %s lost raw flag: %s", p.Name(), q.Name())
			}
			if q.FakeTLS != "" || q.Pattern != "" || q.SeqOvl != 0 || q.HostFake != "" {
				t.Fatalf("neighbor of %s has tls fields: %s", p.Name(), q.Name())
			}
			if strings.Contains(q.SplitPos, "sld") || strings.Contains(q.SplitPos, "sni") {
				t.Fatalf("neighbor of %s has tls split marker: %s", p.Name(), q.Name())
			}
			seen[q.Key()] = true
		}
	}
	if len(seen) < 40 {
		t.Fatalf("too few telegram neighbors: %d", len(seen))
	}
}

func TestBuildArgsTelegram(t *testing.T) {
	plan := Plan{Google: Proven[0], General: Proven[1], Telegram: ProvenTelegram[0]}
	args := strings.Join(BuildArgs(testPaths, plan, "off"), " ")
	for _, want := range []string{
		"--wf-tcp=80,443,2053,2083,2087,2096,8443,5222",
		"--wf-udp=443,19294-19344,50000-50100,590-1400,3478",
		"--hostlist=L\\" + TelegramList,
		"--filter-tcp=80,443,5222 --ipset=L\\" + TelegramIPs + " --dpi-desync=fake",
		"--filter-udp=590-1400,3478 --ipset=L\\" + TelegramIPs,
	} {
		if !strings.Contains(args, want) {
			t.Errorf("missing %q in\n%s", want, args)
		}
	}
	off := strings.Join(BuildArgs(testPaths, Plan{Google: Proven[0], General: Proven[1]}, "off"), " ")
	if strings.Contains(off, "telegram") || strings.Contains(off, "5222") {
		t.Errorf("telegram args present without a telegram plan:\n%s", off)
	}
}
