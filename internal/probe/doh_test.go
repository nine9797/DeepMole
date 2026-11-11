package probe

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

func TestParseAnswer(t *testing.T) {
	var a dohAnswer
	body := `{"Status":0,"Answer":[{"name":"www.youtube.com.","type":5,"data":"youtube-ui.l.google.com."},{"name":"youtube-ui.l.google.com.","type":1,"data":"142.250.74.78"},{"name":"x","type":1,"data":"127.0.0.1"}]}`
	if err := json.Unmarshal([]byte(body), &a); err != nil {
		t.Fatal(err)
	}
	ips, err := parseAnswer(a)
	if err != nil || len(ips) != 1 || ips[0] != "142.250.74.78" {
		t.Fatalf("got %v %v", ips, err)
	}
	if _, err := parseAnswer(dohAnswer{Status: 3}); err == nil {
		t.Fatal("nxdomain accepted")
	}
}

func TestFallbackLive(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	for _, s := range dohServers {
		ips, err := s.doh(ctx, "www.youtube.com")
		t.Logf("doh %-10s %v %v", s.Name, ips, err)
	}
	for _, s := range udpServers {
		ips, err := s.udp(ctx, "www.youtube.com")
		t.Logf("udp %-10s %v %v", s.Name, ips, err)
	}
	ips, name, secure, err := lookupFallback(ctx, "www.youtube.com")
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("fallback: %s doh=%v %v", name, secure, ips)
}
