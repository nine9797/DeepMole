package update

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"golang.org/x/sys/windows"

	"deepmole/internal/probe"
)

const Repo = "nine9797/DeepMole"

type Release struct {
	Version string
	Page    string
	Setup   string
	Zip     string
	Sums    string
	Notes   string
}

type apiRelease struct {
	TagName string `json:"tag_name"`
	HTMLURL string `json:"html_url"`
	Body    string `json:"body"`
	Assets  []struct {
		Name string `json:"name"`
		URL  string `json:"browser_download_url"`
	} `json:"assets"`
}

func parseVersion(s string) ([3]int, bool) {
	var v [3]int
	s = strings.TrimPrefix(strings.TrimSpace(s), "v")
	parts := strings.Split(s, ".")
	if len(parts) != 3 {
		return v, false
	}
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 {
			return v, false
		}
		v[i] = n
	}
	return v, true
}

func Newer(current, latest string) bool {
	c, ok1 := parseVersion(current)
	l, ok2 := parseVersion(latest)
	if !ok1 || !ok2 {
		return false
	}
	for i := 0; i < 3; i++ {
		if l[i] != c[i] {
			return l[i] > c[i]
		}
	}
	return false
}

func request(ctx context.Context, url, current string, timeout time.Duration) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "DeepMole/"+current)
	req.Header.Set("Accept", "application/vnd.github+json, application/octet-stream, */*")
	resp, err := probe.Client(timeout).Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("http %d", resp.StatusCode)
	}
	return resp, nil
}

func Check(ctx context.Context, current string) (*Release, error) {
	resp, err := request(ctx, "https://api.github.com/repos/"+Repo+"/releases/latest", current, 15*time.Second)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var r apiRelease
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&r); err != nil {
		return nil, err
	}
	rel := &Release{Version: strings.TrimPrefix(r.TagName, "v"), Page: r.HTMLURL, Notes: r.Body}
	for _, a := range r.Assets {
		switch {
		case strings.HasSuffix(a.Name, "-setup.exe"):
			rel.Setup = a.URL
		case strings.HasSuffix(a.Name, "-windows-x64.zip") && !strings.Contains(a.Name, "-full-"):
			rel.Zip = a.URL
		case a.Name == "SHA256SUMS":
			rel.Sums = a.URL
		}
	}
	if _, ok := parseVersion(rel.Version); !ok {
		return nil, errors.New("bad tag " + r.TagName)
	}
	return rel, nil
}

func Installed(exeDir string) bool {
	_, err := os.Stat(filepath.Join(exeDir, "unins000.exe"))
	return err == nil
}

func Download(ctx context.Context, rel *Release, current, dir string, installed bool, progress func(pct int)) (string, error) {
	url := rel.Zip
	if installed {
		url = rel.Setup
	}
	if url == "" {
		return "", errors.New("release has no suitable file")
	}
	name := url[strings.LastIndex(url, "/")+1:]
	dst := filepath.Join(dir, name)
	resp, err := request(ctx, url, current, 10*time.Minute)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	f, err := os.Create(dst)
	if err != nil {
		return "", err
	}
	h := sha256.New()
	buf := make([]byte, 64*1024)
	var done int64
	last := -1
	for {
		n, rerr := resp.Body.Read(buf)
		if n > 0 {
			if _, werr := f.Write(buf[:n]); werr != nil {
				f.Close()
				return "", werr
			}
			h.Write(buf[:n])
			done += int64(n)
			if resp.ContentLength > 0 && progress != nil {
				if pct := int(done * 100 / resp.ContentLength); pct != last {
					last = pct
					progress(pct)
				}
			}
		}
		if rerr == io.EOF {
			break
		}
		if rerr != nil {
			f.Close()
			return "", rerr
		}
	}
	f.Close()
	if resp.ContentLength > 0 && done != resp.ContentLength {
		return "", errors.New("download incomplete")
	}
	if rel.Sums == "" {
		return "", errors.New("release has no checksums")
	}
	want, err := fetchSum(ctx, rel.Sums, current, name)
	if err != nil {
		return "", err
	}
	if got := hex.EncodeToString(h.Sum(nil)); got != want {
		_ = os.Remove(dst)
		return "", errors.New("checksum mismatch")
	}
	return dst, nil
}

func fetchSum(ctx context.Context, url, current, name string) (string, error) {
	resp, err := request(ctx, url, current, 30*time.Second)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	if err != nil {
		return "", err
	}
	for _, l := range strings.Split(string(data), "\n") {
		f := strings.Fields(l)
		if len(f) == 2 && strings.TrimPrefix(f[1], "*") == name {
			return strings.ToLower(f[0]), nil
		}
	}
	return "", errors.New("no checksum for " + name)
}

func Apply(exe, file string, installed bool) error {
	dir := filepath.Dir(exe)
	var script string
	if installed {
		script = fmt.Sprintf("@echo off\r\n\"%s\" /SILENT /SUPPRESSMSGBOXES /NORESTART /CLOSEAPPLICATIONS\r\nstart \"\" \"%s\"\r\ndel \"%s\"\r\n", file, exe, file)
	} else {
		tmp := filepath.Join(filepath.Dir(file), "deepmole-new")
		if err := unpack(file, tmp, "deepmole.exe", "deepmole-cli.exe"); err != nil {
			return err
		}
		pid := os.Getpid()
		script = fmt.Sprintf("@echo off\r\n:wait\r\ntasklist /FI \"PID eq %d\" | find \"%d\" >nul && (timeout /t 1 /nobreak >nul & goto wait)\r\n"+
			"copy /Y \"%s\\deepmole.exe\" \"%s\\deepmole.exe\" >nul\r\ncopy /Y \"%s\\deepmole-cli.exe\" \"%s\\deepmole-cli.exe\" >nul\r\n"+
			"start \"\" \"%s\"\r\nrd /s /q \"%s\"\r\ndel \"%s\"\r\n", pid, pid, tmp, dir, tmp, dir, exe, tmp, file)
	}
	bat := filepath.Join(filepath.Dir(file), "deepmole-update.cmd")
	if err := os.WriteFile(bat, []byte(script), 0o644); err != nil {
		return err
	}
	flags := uint32(windows.CREATE_NO_WINDOW | windows.CREATE_NEW_PROCESS_GROUP)
	cmd := exec.Command("cmd", "/c", bat)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: flags | windows.CREATE_BREAKAWAY_FROM_JOB}
	if err := cmd.Start(); err != nil {
		cmd = exec.Command("cmd", "/c", bat)
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: flags}
		return cmd.Start()
	}
	return nil
}

func unpack(zipPath, dir string, names ...string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer r.Close()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	found := 0
	for _, f := range r.File {
		base := filepath.Base(f.Name)
		want := false
		for _, n := range names {
			if base == n {
				want = true
			}
		}
		if !want || f.FileInfo().IsDir() {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return err
		}
		out, err := os.Create(filepath.Join(dir, base))
		if err != nil {
			rc.Close()
			return err
		}
		_, err = io.Copy(out, rc)
		out.Close()
		rc.Close()
		if err != nil {
			return err
		}
		found++
	}
	if found != len(names) {
		return errors.New("archive is missing program files")
	}
	return nil
}
