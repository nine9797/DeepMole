package probe

import (
	"context"
	"crypto/tls"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

type Target struct {
	Block    string `json:"block"`
	URL      string `json:"url"`
	MinBytes int    `json:"min_bytes"`
}

type Result struct {
	Target Target
	OK     bool
	Bytes  int
	Err    string
	Millis int64
	DoH    bool
}

func (r Result) Host() string {
	if u, err := url.Parse(r.Target.URL); err == nil {
		return u.Hostname()
	}
	return r.Target.URL
}

const (
	BlockGoogle  = "google"
	BlockGeneral = "general"
	BlockControl = "control"
)

func Default() []Target {
	return []Target{
		{BlockGoogle, "https://www.youtube.com/", 50000},
		{BlockGoogle, "https://i.ytimg.com/vi/dQw4w9WgXcQ/hqdefault.jpg", 5000},
		{BlockGoogle, "https://redirector.googlevideo.com/", 0},
		{BlockGeneral, "https://discord.com/", 2000},
		{BlockGeneral, "https://cdn.discordapp.com/embed/avatars/0.png", 500},
		{BlockGeneral, "https://gateway.discord.gg/", 0},
		{BlockControl, "https://ya.ru/", 0},
		{BlockControl, "https://vk.com/", 0},
	}
}

const maxRead = 256 * 1024

func newClient(timeout time.Duration, doh *bool) *http.Client {
	d := &net.Dialer{Timeout: timeout}
	tr := &http.Transport{
		DisableKeepAlives:   true,
		TLSHandshakeTimeout: timeout,
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			host, port, err := net.SplitHostPort(addr)
			if err != nil || net.ParseIP(host) != nil {
				return d.DialContext(ctx, network, addr)
			}
			ips, viaDoH, err := resolve(ctx, host)
			if err != nil {
				return nil, err
			}
			if viaDoH {
				*doh = true
			}
			var last error
			for _, ip := range ips {
				c, err := d.DialContext(ctx, network, net.JoinHostPort(ip, port))
				if err == nil {
					return c, nil
				}
				last = err
			}
			return nil, last
		},
		TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12},
	}
	return &http.Client{Transport: tr, Timeout: timeout}
}

func Check(ctx context.Context, t Target, timeout time.Duration) Result {
	start := time.Now()
	res := Result{Target: t}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, t.URL, nil)
	if err != nil {
		res.Err = err.Error()
		return res
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/130.0 Safari/537.36")
	req.Header.Set("Accept", "*/*")

	client := newClient(timeout, &res.DoH)
	defer client.CloseIdleConnections()

	resp, err := client.Do(req)
	if err != nil {
		res.Err = classify(err)
		res.Millis = time.Since(start).Milliseconds()
		return res
	}
	defer resp.Body.Close()

	want := t.MinBytes
	if want < 1 {
		want = 1
	}
	buf := make([]byte, 16*1024)
	for res.Bytes < want && res.Bytes < maxRead {
		n, rerr := resp.Body.Read(buf)
		res.Bytes += n
		if rerr != nil {
			if rerr != io.EOF {
				res.Err = classify(rerr)
			}
			break
		}
	}
	res.Millis = time.Since(start).Milliseconds()
	res.OK = res.Bytes >= t.MinBytes
	if res.OK {
		res.Err = ""
	}
	return res
}

func classify(err error) string {
	s := err.Error()
	switch {
	case strings.Contains(s, "certificate"), strings.Contains(s, "x509"):
		return "ssl"
	case strings.Contains(s, "no such host"):
		return "dns"
	case strings.Contains(s, "deadline"), strings.Contains(s, "timeout"), strings.Contains(s, "Timeout"):
		return "timeout"
	case strings.Contains(s, "reset"), strings.Contains(s, "forcibly closed"):
		return "reset"
	case strings.Contains(s, "EOF"):
		return "eof"
	}
	return s
}

func Run(ctx context.Context, targets []Target, timeout time.Duration) []Result {
	out := make([]Result, len(targets))
	var wg sync.WaitGroup
	for i, t := range targets {
		wg.Add(1)
		go func(i int, t Target) {
			defer wg.Done()
			out[i] = Check(ctx, t, timeout)
		}(i, t)
	}
	wg.Wait()
	return out
}

type Score struct {
	OK    int
	Total int
}

func (s Score) Frac() float64 {
	if s.Total == 0 {
		return 0
	}
	return float64(s.OK) / float64(s.Total)
}

func (s Score) Full() bool { return s.Total > 0 && s.OK == s.Total }

func Summarize(results []Result) map[string]Score {
	m := map[string]Score{}
	for _, r := range results {
		s := m[r.Target.Block]
		s.Total++
		if r.OK {
			s.OK++
		}
		m[r.Target.Block] = s
	}
	return m
}

func Filter(targets []Target, blocks ...string) []Target {
	var out []Target
	for _, t := range targets {
		for _, b := range blocks {
			if t.Block == b {
				out = append(out, t)
				break
			}
		}
	}
	return out
}
