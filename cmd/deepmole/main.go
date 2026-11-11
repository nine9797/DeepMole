package main

import (
	"context"
	"deepmole/internal/i18n"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"golang.org/x/sys/windows"

	"deepmole/internal/engine"
	"deepmole/internal/netid"
	"deepmole/internal/probe"
	"deepmole/internal/strategy"
)

var version = "dev"

type fileConfig struct {
	ZapretDir       string `json:"zapret_dir"`
	Game            string `json:"game_filter"`
	ProbeTimeoutMs  int    `json:"probe_timeout_ms"`
	WatchdogSeconds int    `json:"watchdog_interval_s"`
	MaxCandidates   int    `json:"max_candidates"`
	Language        string `json:"language"`
}

var logger *log.Logger

func main() {
	cli := flag.Bool("cli", false, "run in console without window")
	dry := flag.Bool("dry", false, "print winws arguments and run probes without starting winws")
	zapret := flag.String("zapret", "", "folder with bin/ and lists/ of zapret")
	game := flag.String("game", "", "game filter: off, tcp, udp, all")
	ui := flag.Bool("ui", false, "show the window with sample data, without winws")
	flag.Parse()
	console := *cli || *dry

	exe, _ := os.Executable()
	exeDir := filepath.Dir(exe)
	dataDir := exeDir
	if !writable(exeDir) {
		dataDir = filepath.Join(os.Getenv("ProgramData"), "DeepMole")
		_ = os.MkdirAll(dataDir, 0o755)
	}

	fc := fileConfig{}
	if b, err := os.ReadFile(filepath.Join(exeDir, "deepmole.json")); err == nil {
		_ = json.Unmarshal(b, &fc)
	}
	i18n.Init(fc.Language)
	if *ui {
		runTray(context.Background(), func() {}, nil, "", "")
		return
	}
	if *zapret != "" {
		fc.ZapretDir = *zapret
	}
	if *game != "" {
		fc.Game = *game
	}
	if fc.ZapretDir == "" {
		d, err := engine.FindZapret(exeDir)
		if err != nil {
			fatal(console, i18n.T("main.no_zapret"))
		}
		fc.ZapretDir = d
	}
	if !filepath.IsAbs(fc.ZapretDir) {
		fc.ZapretDir = filepath.Join(exeDir, fc.ZapretDir)
	}

	logPath := filepath.Join(dataDir, "deepmole.log")
	logger = openLog(logPath, console)
	logger.Printf("DeepMole %s, zapret: %s", version, fc.ZapretDir)

	cfg := engine.Config{
		ZapretDir:        fc.ZapretDir,
		Game:             fc.Game,
		ProbeTimeout:     time.Duration(fc.ProbeTimeoutMs) * time.Millisecond,
		WatchdogInterval: time.Duration(fc.WatchdogSeconds) * time.Second,
		MaxCandidates:    fc.MaxCandidates,
		StatePath:        filepath.Join(dataDir, "deepmole-state.json"),
		Log:              logger,
	}

	if *dry {
		runDry(cfg)
		return
	}

	if !elevated() {
		if *cli {
			fatal(true, i18n.T("main.admin"))
		}
		relaunchElevated(exe)
		return
	}

	if !singleInstance() {
		fatal(console, i18n.T("main.running"))
	}

	eng, err := engine.New(cfg)
	if err != nil {
		fatal(console, err.Error())
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if *cli {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
		eng.OnChange = func(s engine.Status) { logger.Printf("status: %s %s", s.Text(), s.Message) }
		go func() { <-sig; cancel() }()
		eng.Run(ctx)
		eng.Stop()
		return
	}

	runTray(ctx, cancel, eng, logPath, exe)
}

func runDry(cfg engine.Config) {
	paths := strategy.Paths{
		Bin:   filepath.Join(cfg.ZapretDir, "bin") + `\`,
		Lists: filepath.Join(cfg.ZapretDir, "lists") + `\`,
	}
	plan := strategy.Plan{Google: strategy.Proven[0], General: strategy.Proven[0]}
	fmt.Println("winws.exe " + strings.Join(strategy.BuildArgs(paths, plan, cfg.Game), " "))
	n := netid.Current()
	fmt.Printf("\nnetwork: %s (%s) gateway %s\n", n.Name, n.ID, n.Gateway)
	fmt.Printf("candidates: %d proven, %d neighbors of the first\n\n", len(strategy.Proven), len(strategy.Neighbors(strategy.Proven[0])))
	timeout := cfg.ProbeTimeout
	if timeout == 0 {
		timeout = 3 * time.Second
	}
	for _, r := range probe.Run(context.Background(), probe.Default(), timeout) {
		mark := "ok"
		if !r.OK {
			mark = "fail " + r.Err
		}
		fmt.Printf("%-8s %-55s %6dB %5dms %s\n", r.Target.Block, r.Target.URL, r.Bytes, r.Millis, mark)
	}
}

func openLog(path string, console bool) *log.Logger {
	if fi, err := os.Stat(path); err == nil && fi.Size() > 2<<20 {
		_ = os.Remove(path)
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	var w io.Writer = os.Stdout
	if err == nil {
		if console {
			w = io.MultiWriter(os.Stdout, f)
		} else {
			w = f
		}
	}
	return log.New(w, "", log.Ltime)
}

func writable(dir string) bool {
	f, err := os.CreateTemp(dir, ".w")
	if err != nil {
		return false
	}
	f.Close()
	_ = os.Remove(f.Name())
	return true
}

func elevated() bool {
	return windows.GetCurrentProcessToken().IsElevated()
}

func singleInstance() bool {
	name, _ := syscall.UTF16PtrFromString("Global\\DeepMole")
	_, err := windows.CreateMutex(nil, false, name)
	return err == nil || err != windows.ERROR_ALREADY_EXISTS
}

func relaunchElevated(exe string) {
	verb, _ := syscall.UTF16PtrFromString("runas")
	file, _ := syscall.UTF16PtrFromString(exe)
	args, _ := syscall.UTF16PtrFromString(strings.Join(os.Args[1:], " "))
	dir, _ := syscall.UTF16PtrFromString(filepath.Dir(exe))
	_ = windows.ShellExecute(0, verb, file, args, dir, windows.SW_NORMAL)
}

func fatal(console bool, msg string) {
	if console {
		fmt.Fprintln(os.Stderr, msg)
	} else {
		text, _ := syscall.UTF16PtrFromString(msg)
		title, _ := syscall.UTF16PtrFromString("DeepMole")
		_, _ = windows.MessageBox(0, text, title, windows.MB_OK|windows.MB_ICONERROR)
	}
	os.Exit(1)
}

func openPath(p string) {
	verb, _ := syscall.UTF16PtrFromString("open")
	file, _ := syscall.UTF16PtrFromString(p)
	if err := windows.ShellExecute(0, verb, file, nil, nil, windows.SW_NORMAL); err != nil && logger != nil {
		logger.Printf("open %s: %v", p, err)
	}
}
