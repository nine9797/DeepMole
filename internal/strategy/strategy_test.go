package strategy

import (
	"sort"
	"strings"
	"testing"
)

func same(t *testing.T, got []string, want string) {
	t.Helper()
	w := strings.Fields(want)
	g := append([]string(nil), got...)
	sort.Strings(g)
	sort.Strings(w)
	if strings.Join(g, " ") != strings.Join(w, " ") {
		t.Errorf("\n got: %s\nwant: %s", strings.Join(g, " "), strings.Join(w, " "))
	}
}

func TestRenderMatchesPresets(t *testing.T) {
	const bin = `B\`
	cases := []struct {
		p    Params
		http bool
		want string
	}{
		{Proven[0], true, `--dpi-desync=multisplit --dpi-desync-split-seqovl=568 --dpi-desync-split-pos=1 --dpi-desync-split-seqovl-pattern=B\tls_clienthello_4pda_to.bin`},
		{Proven[1], true, `--dpi-desync=fake,multidisorder --dpi-desync-split-pos=1,midsld --dpi-desync-repeats=11 --dpi-desync-fooling=badseq --dpi-desync-fake-tls=0x00000000 --dpi-desync-fake-tls=! --dpi-desync-fake-tls-mod=rnd,dupsid,sni=www.google.com --dpi-desync-fake-http=B\tls_clienthello_max_ru.bin`},
		{Proven[2], true, `--dpi-desync=fake --dpi-desync-repeats=6 --dpi-desync-fooling=ts --dpi-desync-fake-tls=B\stun.bin --dpi-desync-fake-tls=B\tls_clienthello_www_google_com.bin --dpi-desync-fake-http=B\tls_clienthello_max_ru.bin`},
		{Proven[2], false, `--dpi-desync=fake --dpi-desync-repeats=6 --dpi-desync-fooling=ts --dpi-desync-fake-tls=B\stun.bin --dpi-desync-fake-tls=B\tls_clienthello_www_google_com.bin`},
		{Proven[5], true, `--dpi-desync=fake,multisplit --dpi-desync-repeats=6 --dpi-desync-fooling=badseq --dpi-desync-badseq-increment=1000 --dpi-desync-fake-tls=B\stun.bin --dpi-desync-fake-tls=B\tls_clienthello_www_google_com.bin --dpi-desync-fake-http=B\tls_clienthello_max_ru.bin`},
		{Proven[10], true, `--dpi-desync=multisplit --dpi-desync-split-pos=2,sniext+1 --dpi-desync-split-seqovl=679 --dpi-desync-split-seqovl-pattern=B\tls_clienthello_www_google_com.bin`},
		{Proven[12], true, `--dpi-desync=fake,hostfakesplit --dpi-desync-fake-tls-mod=rnd,dupsid,sni=ya.ru --dpi-desync-hostfakesplit-mod=host=ya.ru,altorder=1 --dpi-desync-fooling=ts --dpi-desync-fake-http=B\tls_clienthello_max_ru.bin`},
		{Proven[13], true, `--dpi-desync=hostfakesplit --dpi-desync-repeats=4 --dpi-desync-fooling=ts,md5sig --dpi-desync-hostfakesplit-mod=host=ozon.ru`},
		{Proven[14], true, `--dpi-desync=fake --dpi-desync-fake-tls-mod=none --dpi-desync-repeats=6 --dpi-desync-fooling=badseq --dpi-desync-badseq-increment=2 --dpi-desync-fake-http=B\tls_clienthello_max_ru.bin`},
	}
	for i, c := range cases {
		got := c.p.Render(bin, c.http)
		t.Run(c.p.Name(), func(t *testing.T) { same(t, got, c.want) })
		_ = i
	}
}

func TestRenderExtraKnobs(t *testing.T) {
	p := Params{Method: "fake,multisplit", Fooling: "md5sig", SplitPos: "midsld", Pattern: "google", SeqOvl: 681, FakeTLS: "mod:padencap", Repeats: 6, TTL: "autottl=2", WSSize: "1:6"}
	same(t, p.Render(`B\`, false), `--dpi-desync=fake,multisplit --wssize=1:6 --dpi-desync-autottl=2 --dpi-desync-split-seqovl=681 --dpi-desync-split-pos=midsld --dpi-desync-fooling=md5sig --dpi-desync-repeats=6 --dpi-desync-split-seqovl-pattern=B\tls_clienthello_www_google_com.bin --dpi-desync-fake-tls-mod=rnd,dupsid,padencap,sni=www.google.com`)
	p = Params{Method: "fake,fakeddisorder", Fooling: "ts", SplitPos: "1", FakeTLS: "stun+google", Repeats: 6, TTL: "ttl=4"}
	same(t, p.Render(`B\`, true), `--dpi-desync=fake,fakeddisorder --dpi-desync-ttl=4 --dpi-desync-split-pos=1 --dpi-desync-fooling=ts --dpi-desync-repeats=6 --dpi-desync-fake-tls=B\stun.bin --dpi-desync-fake-tls=B\tls_clienthello_www_google_com.bin --dpi-desync-fake-http=B\tls_clienthello_max_ru.bin`)
	q := Params{Method: "multisplit", TTL: "ttl=3"}
	if q.Render(`B\`, true)[1] != "--dpi-desync-ttl=3" {
		t.Errorf("ttl must render for any method")
	}
}

func TestProvenUnique(t *testing.T) {
	seen := map[string]bool{}
	for _, p := range Proven {
		if seen[p.Key()] {
			t.Errorf("duplicate preset: %s", p.Name())
		}
		seen[p.Key()] = true
	}
}

func TestNeighborsValid(t *testing.T) {
	for _, p := range Proven {
		for _, n := range Neighbors(p) {
			if n.Key() == p.Key() {
				t.Errorf("neighbor equals origin: %s", p.Name())
			}
			if n.SeqOvl > 0 && n.Pattern == "" {
				t.Errorf("seqovl without pattern: %s", n.Name())
			}
			if !n.hasFake() && n.FakeTLS != "" {
				t.Errorf("fake tls without fake: %s", n.Name())
			}
			if !n.hasFake() && !strings.Contains(n.Method, "hostfakesplit") && n.Fooling != "" {
				t.Errorf("fooling without fake: %s", n.Name())
			}
			if !n.hasFake() && n.TTL != "" {
				t.Errorf("ttl without fake: %s", n.Name())
			}
			if !n.hasSplit() && n.SplitPos != "" {
				t.Errorf("split-pos without split: %s", n.Name())
			}
			args := n.Render(`B\`, true)
			if len(args) == 0 || !strings.HasPrefix(args[0], "--dpi-desync=") || args[0] == "--dpi-desync=" {
				t.Errorf("bad render: %v", args)
			}
		}
	}
}

func TestBuildArgsBlocks(t *testing.T) {
	args := BuildArgs(Paths{Bin: `B\`, Lists: `L\`}, Plan{Google: Proven[0], General: Proven[2]}, "")
	n := 0
	for _, a := range args {
		if a == "--new" {
			n++
		}
	}
	if n != 6 {
		t.Errorf("expected 7 profiles, got %d", n+1)
	}
	if !strings.HasPrefix(args[0], "--wf-tcp=") || !strings.HasPrefix(args[1], "--wf-udp=") {
		t.Errorf("filters must come first: %v", args[:2])
	}
	all := BuildArgs(Paths{Bin: `B\`, Lists: `L\`}, Plan{Google: Proven[0], General: Proven[2]}, "all")
	if !strings.Contains(strings.Join(all, " "), "--filter-tcp=1024-65535") || !strings.Contains(strings.Join(all, " "), "--filter-udp=1024-65535") {
		t.Errorf("game filter blocks missing")
	}
}
