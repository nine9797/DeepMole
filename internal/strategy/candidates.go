package strategy

var Proven = []Params{
	{Method: "multisplit", SplitPos: "1", Pattern: "4pda", SeqOvl: 568},
	{Method: "multisplit", SeqOvl: 681, SplitPos: "1", Pattern: "google"},
	{Method: "fake,multidisorder", SplitPos: "1,midsld", Repeats: 11, Fooling: "badseq", FakeTLS: "zero+mod"},
	{Method: "fake", Repeats: 6, Fooling: "ts", FakeTLS: "stun+google"},
	{Method: "fake,multisplit", SeqOvl: 681, SplitPos: "1", Fooling: "ts", Repeats: 8, Pattern: "google", FakeTLS: "mod:google"},
	{Method: "fake,multisplit", SeqOvl: 664, SplitPos: "1", Fooling: "ts", Repeats: 8, Pattern: "sferum", FakeTLS: "stun2+sferum"},
	{Method: "fake,multisplit", Repeats: 6, Fooling: "badseq", BadseqInc: 1000, FakeTLS: "stun+google"},
	{Method: "fake,fakedsplit", Repeats: 6, Fooling: "ts", FakeTLS: "stun+google"},
	{Method: "fake,fakedsplit", SplitPos: "1", Fooling: "badseq", BadseqInc: 2, Repeats: 8, FakeTLS: "mod:google"},
	{Method: "fake,multisplit", SeqOvl: 681, SplitPos: "1", Fooling: "badseq", BadseqInc: 10000000, Repeats: 8, Pattern: "google", FakeTLS: "mod:google"},
	{Method: "multisplit", SeqOvl: 652, SplitPos: "2", Pattern: "google"},
	{Method: "multisplit", SplitPos: "2,sniext+1", SeqOvl: 679, Pattern: "google"},
	{Method: "fake,multisplit", SeqOvl: 664, SplitPos: "1", Fooling: "ts", Repeats: 8, Pattern: "max", FakeTLS: "stun+max"},
	{Method: "fake,multisplit", SeqOvl: 664, SplitPos: "1", Fooling: "ts", Repeats: 8, Pattern: "sferum", FakeTLS: "stun+sferum"},
	{Method: "fake,multisplit", SeqOvl: 480, SplitPos: "1", Fooling: "ts", Repeats: 4, Pattern: "stun2", FakeTLS: "sferum"},
	{Method: "hostfakesplit", Fooling: "ts", HostFake: "www.google.com"},
	{Method: "fake,hostfakesplit", Fooling: "ts", FakeTLS: "mod:google", HostFake: "www.google.com"},
	{Method: "fake,hostfakesplit", Fooling: "ts", FakeTLS: "mod:ya", HostFake: "ya.ru"},
	{Method: "fake,hostfakesplit", Fooling: "ts", Repeats: 5, FakeTLS: "sochi+stun2", HostFake: "mail.ru"},
	{Method: "hostfakesplit", Repeats: 4, Fooling: "ts", HostFake: "www.google.com"},
	{Method: "hostfakesplit", Repeats: 4, Fooling: "ts,md5sig", HostFake: "ozon.ru"},
	{Method: "fake", Repeats: 6, Fooling: "badseq", BadseqInc: 2, FakeTLS: "none"},
	{Method: "fake", Repeats: 6, Fooling: "ts", FakeTLS: "stun2+sferum"},
	{Method: "fake", Repeats: 6, Fooling: "ts", FakeTLS: "stun+4pda"},
	{Method: "fake", Repeats: 6, Fooling: "badseq", BadseqInc: 2, FakeTLS: "stun+google"},
	{Method: "fake", Repeats: 6, Fooling: "badseq", BadseqInc: 2, FakeTLS: "stun2+google"},
	{Method: "fake", Repeats: 6, Fooling: "ts", FakeTLS: "stun+max"},
	{Method: "syndata,multidisorder"},
}

var ProvenTelegram = []Params{
	{Method: "fake", Fooling: "badseq", Repeats: 6, Raw: true},
	{Method: "fake", Fooling: "ts", Repeats: 6, TTL: "autottl=2", Raw: true},
	{Method: "fake", Fooling: "md5sig", Repeats: 6, Raw: true},
	{Method: "multisplit", SplitPos: "1", Raw: true},
	{Method: "multidisorder", SplitPos: "1", Raw: true},
	{Method: "fake,multisplit", Fooling: "badseq", SplitPos: "1", Repeats: 6, Raw: true},
	{Method: "fake,multidisorder", Fooling: "badseq", SplitPos: "1", Repeats: 6, Raw: true},
	{Method: "fake,fakedsplit", Fooling: "badseq", SplitPos: "1", Repeats: 6, Raw: true},
	{Method: "fake,fakeddisorder", Fooling: "ts", SplitPos: "1", Repeats: 6, Raw: true},
	{Method: "fake", Fooling: "badseq", Repeats: 11, WSSize: "1:6", Raw: true},
	{Method: "syndata", Raw: true},
}

var (
	methods  = []string{"fake,multisplit", "fake,multidisorder", "fake,fakedsplit", "fake,fakeddisorder", "multisplit", "multidisorder", "fake"}
	rawPos   = []string{"1", "2", "10"}
	foolings = []string{"ts", "badseq", "ts,md5sig", "md5sig"}
	splitPos = []string{"1", "2", "1,midsld", "2,sniext+1", "midsld", "10"}
	patterns = []string{"google", "max", "4pda", "sferum", "stun2"}
	fakeTLS  = []string{"stun+google", "stun+max", "stun2+sferum", "stun+sferum", "stun2+google", "mod:google", "zero+mod", "mod:padencap", "mod:rndsni", "none"}
	repeats  = []int{6, 8, 11}
	ttls     = []string{"", "autottl=2", "autottl=3", "ttl=3", "ttl=4", "ttl=5"}
	wssizes  = []string{"", "1:6"}
)

func Neighbors(p Params) []Params {
	var out []Params
	add := func(q Params) {
		if q.Key() != p.Key() {
			out = append(out, q)
		}
	}
	if p.hasFake() {
		for _, t := range ttls {
			q := p
			q.TTL = t
			add(q)
		}
		for _, f := range foolings {
			q := p
			q.Fooling = f
			if f != "badseq" {
				q.BadseqInc = 0
			}
			add(q)
		}
		if !p.Raw {
			for _, t := range fakeTLS {
				q := p
				q.FakeTLS = t
				add(q)
			}
		}
	}
	if p.hasSplit() {
		pos := splitPos
		if p.Raw {
			pos = rawPos
		}
		for _, s := range pos {
			q := p
			q.SplitPos = s
			add(q)
		}
	}
	if p.SeqOvl > 0 && !p.Raw {
		for _, pt := range patterns {
			q := p
			q.Pattern = pt
			q.SeqOvl = patternOvl[pt]
			add(q)
		}
	}
	for _, m := range methods {
		q := p
		q.Method = m
		q.HostFake = ""
		if q.hasFake() {
			if q.Fooling == "" {
				q.Fooling = "ts"
			}
			if q.FakeTLS == "" && !q.Raw {
				q.FakeTLS = "stun+google"
			}
			if q.Repeats == 0 {
				q.Repeats = 6
			}
		} else {
			q.Fooling, q.BadseqInc, q.FakeTLS, q.Repeats, q.TTL = "", 0, "", 0, ""
		}
		if q.hasSplit() {
			if q.SplitPos == "" {
				q.SplitPos = "1"
			}
		} else {
			q.SplitPos = ""
		}
		if (m == "multisplit" || m == "fake,multisplit") && !q.Raw {
			if q.Pattern == "" {
				q.Pattern = "google"
				q.SeqOvl = patternOvl["google"]
			}
		} else {
			q.Pattern, q.SeqOvl = "", 0
		}
		add(q)
	}
	if p.hasFake() {
		for _, r := range repeats {
			q := p
			q.Repeats = r
			add(q)
		}
	}
	for _, w := range wssizes {
		q := p
		q.WSSize = w
		add(q)
	}
	return out
}
