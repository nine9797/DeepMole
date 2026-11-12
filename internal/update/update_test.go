package update

import (
	"archive/zip"
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewer(t *testing.T) {
	cases := []struct {
		cur, latest string
		want        bool
	}{
		{"1.0.2", "1.0.3", true},
		{"1.0.2", "1.0.2", false},
		{"1.0.3", "1.0.2", false},
		{"1.0.9", "1.1.0", true},
		{"1.9.9", "2.0.0", true},
		{"dev", "1.0.3", false},
		{"1.0.2", "v1.0.3", true},
		{"1.0.2", "latest", false},
	}
	for _, c := range cases {
		if got := Newer(c.cur, c.latest); got != c.want {
			t.Errorf("Newer(%q,%q)=%v", c.cur, c.latest, got)
		}
	}
}

func TestUnpack(t *testing.T) {
	dir := t.TempDir()
	zp := filepath.Join(dir, "a.zip")
	f, _ := os.Create(zp)
	w := zip.NewWriter(f)
	for _, n := range []string{"deepmole.exe", "deepmole-cli.exe", "README.md"} {
		fw, _ := w.Create(n)
		fw.Write([]byte("x" + n))
	}
	w.Close()
	f.Close()
	out := filepath.Join(dir, "out")
	if err := unpack(zp, out, "deepmole.exe", "deepmole-cli.exe"); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(filepath.Join(out, "deepmole.exe")); string(b) != "xdeepmole.exe" {
		t.Fatal("bad content")
	}
	if _, err := os.Stat(filepath.Join(out, "README.md")); err == nil {
		t.Fatal("extra file extracted")
	}
	if err := unpack(zp, out, "missing.exe"); err == nil {
		t.Fatal("missing file not reported")
	}
}

func TestLive(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	rel, err := Check(ctx, "1.0.0")
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("latest %s setup=%s zip=%s sums=%s", rel.Version, rel.Setup, rel.Zip, rel.Sums)
	if !Newer("1.0.0", rel.Version) {
		t.Fatal("1.0.0 must be older than the latest release")
	}
	if rel.Sums == "" {
		t.Skip("release has no checksums yet")
	}
	file, err := Download(ctx, rel, "1.0.0", t.TempDir(), false, func(pct int) {})
	if err != nil {
		t.Fatal(err)
	}
	fi, _ := os.Stat(file)
	t.Logf("downloaded %s (%d bytes)", filepath.Base(file), fi.Size())
	if err := unpack(file, filepath.Join(t.TempDir(), "x"), "deepmole.exe", "deepmole-cli.exe"); err != nil {
		t.Fatal(err)
	}
}
