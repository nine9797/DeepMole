package strategy

import "os"

type Paths struct {
	Bin    string
	Lists  string
	Exists func(path string) bool
}

func NewPaths(bin, lists string) Paths {
	return Paths{Bin: bin, Lists: lists, Exists: func(p string) bool {
		_, err := os.Stat(p)
		return err == nil
	}}
}

func (p Paths) file(names ...string) string {
	for _, n := range names {
		if p.Exists == nil || p.Exists(p.Bin+n) {
			return p.Bin + n
		}
	}
	return p.Bin + names[0]
}

type Plan struct {
	Google  Params `json:"google"`
	General Params `json:"general"`
}

const gamePorts = "1024-65535"

func BuildArgs(p Paths, plan Plan, game string) []string {
	l := p.Lists
	gameTCP, gameUDP := "", ""
	if game == "tcp" || game == "all" {
		gameTCP = gamePorts
	}
	if game == "udp" || game == "all" {
		gameUDP = gamePorts
	}
	wfTCP := "80,443,2053,2083,2087,2096,8443"
	if gameTCP != "" {
		wfTCP += "," + gameTCP
	}
	wfUDP := "443,19294-19344,50000-50100"
	if gameUDP != "" {
		wfUDP += "," + gameUDP
	}

	general := []string{"--hostlist=" + l + "list-general.txt", "--hostlist=" + l + "list-general-user.txt"}
	ipsetExcl := []string{"--ipset-exclude=" + l + "ipset-exclude.txt", "--ipset-exclude=" + l + "ipset-exclude-user.txt"}
	excl := append([]string{"--hostlist-exclude=" + l + "list-exclude.txt", "--hostlist-exclude=" + l + "list-exclude-user.txt"}, ipsetExcl...)
	ipset := []string{"--ipset=" + l + "ipset-all.txt"}
	quic := []string{"--dpi-desync=fake", "--dpi-desync-repeats=6", "--dpi-desync-fake-quic=" + p.file("quic_initial_www_google_com.bin", "quic_initial_4pda_to.bin")}
	discordFake := p.file("ACTIVE_DISCORD_UDP.bin", "quic_initial_dbankcloud_ru.bin", "quic_initial_www_google_com.bin")
	gameFake := p.file("ACTIVE_GAME_UDP.bin", "quic_initial_dbankcloud_ru.bin", "quic_initial_www_google_com.bin")

	var blocks [][]string
	add := func(parts ...[]string) {
		var b []string
		for _, x := range parts {
			b = append(b, x...)
		}
		blocks = append(blocks, b)
	}

	add([]string{"--filter-udp=443"}, general, excl, quic)
	add([]string{"--filter-udp=19294-19344,50000-50100", "--filter-l7=discord,stun", "--dpi-desync=fake",
		"--dpi-desync-fake-discord=" + discordFake, "--dpi-desync-fake-stun=" + discordFake, "--dpi-desync-repeats=6"})
	add([]string{"--filter-tcp=2053,2083,2087,2096,8443", "--hostlist-domains=discord.media"}, plan.General.Render(p, false))
	add([]string{"--filter-tcp=443", "--hostlist=" + l + "list-google.txt", "--ip-id=zero"}, plan.Google.Render(p, false))
	add([]string{"--filter-tcp=80,443"}, general, excl, plan.General.Render(p, true))
	add([]string{"--filter-udp=443"}, ipset, excl, quic)
	add([]string{"--filter-tcp=80,443,8443"}, ipset, excl, plan.General.Render(p, true))
	if gameTCP != "" {
		add([]string{"--filter-tcp=" + gameTCP}, ipset, ipsetExcl,
			[]string{"--dpi-desync-any-protocol=1", "--dpi-desync-cutoff=n3"}, plan.General.Render(p, true))
	}
	if gameUDP != "" {
		add([]string{"--filter-udp=" + gameUDP}, ipset, ipsetExcl, []string{"--dpi-desync=fake", "--dpi-desync-repeats=12",
			"--dpi-desync-any-protocol=1", "--dpi-desync-fake-unknown-udp=" + gameFake, "--dpi-desync-cutoff=n2"})
	}

	args := []string{"--wf-tcp=" + wfTCP, "--wf-udp=" + wfUDP}
	for i, b := range blocks {
		if i > 0 {
			args = append(args, "--new")
		}
		args = append(args, b...)
	}
	return args
}
