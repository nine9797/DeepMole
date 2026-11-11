package strategy

import (
	"fmt"
	"strings"
)

type Params struct {
	Method    string `json:"method"`
	Fooling   string `json:"fooling,omitempty"`
	BadseqInc int    `json:"badseq_inc,omitempty"`
	SplitPos  string `json:"split_pos,omitempty"`
	Pattern   string `json:"pattern,omitempty"`
	SeqOvl    int    `json:"seqovl,omitempty"`
	FakeTLS   string `json:"fake_tls,omitempty"`
	Repeats   int    `json:"repeats,omitempty"`
	HostFake  string `json:"host_fake,omitempty"`
	TTL       string `json:"ttl,omitempty"`
	WSSize    string `json:"wssize,omitempty"`
}

var patternFiles = map[string]string{
	"google": "tls_clienthello_www_google_com.bin",
	"max":    "tls_clienthello_max_ru.bin",
	"4pda":   "tls_clienthello_4pda_to.bin",
}

var patternOvl = map[string]int{"google": 681, "max": 664, "4pda": 568}

const httpFake = "tls_clienthello_max_ru.bin"

func (p Params) Key() string {
	return fmt.Sprintf("%s|%s|%d|%s|%s|%d|%s|%d|%s|%s|%s",
		p.Method, p.Fooling, p.BadseqInc, p.SplitPos, p.Pattern, p.SeqOvl, p.FakeTLS, p.Repeats, p.HostFake, p.TTL, p.WSSize)
}

func (p Params) Name() string {
	parts := []string{strings.ReplaceAll(p.Method, ",", "+")}
	if p.Fooling != "" {
		parts = append(parts, p.Fooling)
	}
	if p.SplitPos != "" {
		parts = append(parts, "pos="+p.SplitPos)
	}
	if p.SeqOvl > 0 {
		parts = append(parts, fmt.Sprintf("ovl=%d/%s", p.SeqOvl, p.Pattern))
	}
	if p.FakeTLS != "" {
		parts = append(parts, "tls="+p.FakeTLS)
	}
	if p.HostFake != "" {
		parts = append(parts, "host="+p.HostFake)
	}
	if p.Repeats > 0 {
		parts = append(parts, fmt.Sprintf("r%d", p.Repeats))
	}
	if p.TTL != "" {
		parts = append(parts, p.TTL)
	}
	if p.WSSize != "" {
		parts = append(parts, "wssize="+p.WSSize)
	}
	return strings.Join(parts, " ")
}

func (p Params) IsZero() bool { return p.Method == "" }

func (p Params) hasFake() bool {
	for _, m := range strings.Split(p.Method, ",") {
		if m == "fake" {
			return true
		}
	}
	return false
}

func (p Params) hasSplit() bool {
	return strings.Contains(p.Method, "multisplit") || strings.Contains(p.Method, "multidisorder") || strings.Contains(p.Method, "fakedsplit")
}

func (p Params) Render(bin string, withHTTP bool) []string {
	args := []string{"--dpi-desync=" + p.Method}
	if p.WSSize != "" {
		args = append(args, "--wssize="+p.WSSize)
	}
	if p.TTL != "" {
		if strings.HasPrefix(p.TTL, "autottl=") {
			args = append(args, "--dpi-desync-autottl="+strings.TrimPrefix(p.TTL, "autottl="))
		} else if strings.HasPrefix(p.TTL, "ttl=") {
			args = append(args, "--dpi-desync-ttl="+strings.TrimPrefix(p.TTL, "ttl="))
		}
	}
	if p.SeqOvl > 0 {
		args = append(args, fmt.Sprintf("--dpi-desync-split-seqovl=%d", p.SeqOvl))
	}
	if p.SplitPos != "" {
		args = append(args, "--dpi-desync-split-pos="+p.SplitPos)
	}
	if p.Fooling != "" {
		args = append(args, "--dpi-desync-fooling="+p.Fooling)
	}
	if p.BadseqInc > 0 {
		args = append(args, fmt.Sprintf("--dpi-desync-badseq-increment=%d", p.BadseqInc))
	}
	if p.Repeats > 0 {
		args = append(args, fmt.Sprintf("--dpi-desync-repeats=%d", p.Repeats))
	}
	if p.SeqOvl > 0 && p.Pattern != "" {
		args = append(args, "--dpi-desync-split-seqovl-pattern="+bin+patternFiles[p.Pattern])
	}
	if strings.Contains(p.Method, "fakedsplit") {
		args = append(args, "--dpi-desync-fakedsplit-pattern=0x00")
	}
	if strings.Contains(p.Method, "hostfakesplit") && p.HostFake != "" {
		mod := "host=" + p.HostFake
		if p.hasFake() {
			mod += ",altorder=1"
		}
		args = append(args, "--dpi-desync-hostfakesplit-mod="+mod)
	}
	if p.hasFake() {
		switch p.FakeTLS {
		case "stun+google":
			args = append(args, "--dpi-desync-fake-tls="+bin+"stun.bin", "--dpi-desync-fake-tls="+bin+patternFiles["google"])
		case "stun+max":
			args = append(args, "--dpi-desync-fake-tls="+bin+"stun.bin", "--dpi-desync-fake-tls="+bin+patternFiles["max"])
		case "stun+4pda":
			args = append(args, "--dpi-desync-fake-tls="+bin+"stun.bin", "--dpi-desync-fake-tls="+bin+patternFiles["4pda"])
		case "mod:google":
			args = append(args, "--dpi-desync-fake-tls-mod=rnd,dupsid,sni=www.google.com")
		case "mod:ya":
			args = append(args, "--dpi-desync-fake-tls-mod=rnd,dupsid,sni=ya.ru")
		case "mod:padencap":
			args = append(args, "--dpi-desync-fake-tls-mod=rnd,dupsid,padencap,sni=www.google.com")
		case "mod:rndsni":
			args = append(args, "--dpi-desync-fake-tls-mod=rnd,rndsni,dupsid")
		case "none":
			args = append(args, "--dpi-desync-fake-tls-mod=none")
		case "zero+mod":
			args = append(args, "--dpi-desync-fake-tls=0x00000000", "--dpi-desync-fake-tls=!", "--dpi-desync-fake-tls-mod=rnd,dupsid,sni=www.google.com")
		}
		if withHTTP {
			args = append(args, "--dpi-desync-fake-http="+bin+httpFake)
		}
	}
	return args
}
