package probe

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"
)

type resolver struct {
	Name, IP, Host, Path string
}

var dohServers = []resolver{
	{"Google", "8.8.8.8", "dns.google", "/resolve"},
	{"Cloudflare", "1.1.1.1", "cloudflare-dns.com", "/dns-query"},
	{"AdGuard", "94.140.14.14", "dns.adguard-dns.com", "/resolve"},
}

var udpServers = []resolver{
	{Name: "Google", IP: "8.8.8.8"},
	{Name: "Cloudflare", IP: "1.1.1.1"},
	{Name: "Quad9", IP: "9.9.9.9"},
	{Name: "AdGuard", IP: "94.140.14.14"},
}

type cacheEntry struct {
	ips   []string
	until time.Time
}

var (
	cacheMu    sync.Mutex
	cache      = map[string]cacheEntry{}
	lastName   string
	lastSecure bool
	cacheTTL   = 2 * time.Minute
)

func Fallback() (name string, secure bool) {
	cacheMu.Lock()
	defer cacheMu.Unlock()
	return lastName, lastSecure
}

func ResetCache() {
	cacheMu.Lock()
	cache = map[string]cacheEntry{}
	cacheMu.Unlock()
}

func usable(ips []net.IP) []string {
	var v4, v6 []string
	for _, ip := range ips {
		if !ip.IsGlobalUnicast() || ip.IsPrivate() {
			continue
		}
		if ip.To4() != nil {
			v4 = append(v4, ip.String())
		} else {
			v6 = append(v6, ip.String())
		}
	}
	return append(v4, v6...)
}

const systemLookupTimeout = 1500 * time.Millisecond

func resolve(ctx context.Context, host string) ([]string, bool, error) {
	sctx, cancel := context.WithTimeout(ctx, systemLookupTimeout)
	addrs, err := net.DefaultResolver.LookupIPAddr(sctx, host)
	cancel()
	if err == nil {
		ips := make([]net.IP, 0, len(addrs))
		for _, a := range addrs {
			ips = append(ips, a.IP)
		}
		if u := usable(ips); len(u) > 0 {
			return u, false, nil
		}
		err = errors.New("lookup " + host + ": no such host")
	}
	cacheMu.Lock()
	c, ok := cache[host]
	cacheMu.Unlock()
	if ok && time.Now().Before(c.until) {
		return c.ips, true, nil
	}
	ips, name, secure, ferr := lookupFallback(ctx, host)
	if ferr != nil {
		return nil, false, err
	}
	cacheMu.Lock()
	cache[host] = cacheEntry{ips: ips, until: time.Now().Add(cacheTTL)}
	lastName, lastSecure = name, secure
	cacheMu.Unlock()
	return ips, true, nil
}

func lookupFallback(ctx context.Context, host string) ([]string, string, bool, error) {
	var last error = errors.New("no resolver")
	for _, s := range dohServers {
		ips, err := s.doh(ctx, host)
		if err == nil {
			return ips, s.Name, true, nil
		}
		last = err
	}
	for _, s := range udpServers {
		ips, err := s.udp(ctx, host)
		if err == nil {
			return ips, s.Name, false, nil
		}
		last = err
	}
	return nil, "", false, last
}

type dohAnswer struct {
	Status int `json:"Status"`
	Answer []struct {
		Type int    `json:"type"`
		Data string `json:"data"`
	} `json:"Answer"`
}

func (s resolver) doh(ctx context.Context, host string) ([]string, error) {
	d := &net.Dialer{Timeout: 2 * time.Second}
	tr := &http.Transport{
		DisableKeepAlives: true,
		DialContext: func(ctx context.Context, network, _ string) (net.Conn, error) {
			return d.DialContext(ctx, network, net.JoinHostPort(s.IP, "443"))
		},
		TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12, ServerName: s.Host},
	}
	client := &http.Client{Transport: tr, Timeout: 3 * time.Second}
	defer client.CloseIdleConnections()
	q := url.Values{"name": {host}, "type": {"A"}}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://"+s.Host+s.Path+"?"+q.Encode(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/dns-json")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var a dohAnswer
	if err := json.NewDecoder(resp.Body).Decode(&a); err != nil {
		return nil, err
	}
	return parseAnswer(a)
}

func (s resolver) udp(ctx context.Context, host string) ([]string, error) {
	r := &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, _ string) (net.Conn, error) {
			d := &net.Dialer{Timeout: 2 * time.Second}
			return d.DialContext(ctx, network, net.JoinHostPort(s.IP, "53"))
		},
	}
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	ips, err := r.LookupIP(ctx, "ip4", host)
	if err != nil {
		return nil, err
	}
	u := usable(ips)
	if len(u) == 0 {
		return nil, errors.New("no address")
	}
	return u, nil
}

func parseAnswer(a dohAnswer) ([]string, error) {
	if a.Status != 0 {
		return nil, errors.New("dns status " + strconv.Itoa(a.Status))
	}
	var ips []net.IP
	for _, r := range a.Answer {
		if r.Type != 1 && r.Type != 28 {
			continue
		}
		if ip := net.ParseIP(r.Data); ip != nil {
			ips = append(ips, ip)
		}
	}
	u := usable(ips)
	if len(u) == 0 {
		return nil, errors.New("no address")
	}
	return u, nil
}
