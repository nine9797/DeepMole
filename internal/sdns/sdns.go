package sdns

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

type Provider struct {
	Name     string
	V4       []string
	V6       []string
	Template string
}

var Providers = []Provider{
	{"Google", []string{"8.8.8.8", "8.8.4.4"}, []string{"2001:4860:4860::8888", "2001:4860:4860::8844"}, "https://dns.google/dns-query"},
	{"Cloudflare", []string{"1.1.1.1", "1.0.0.1"}, []string{"2606:4700:4700::1111", "2606:4700:4700::1001"}, "https://cloudflare-dns.com/dns-query"},
	{"Quad9", []string{"9.9.9.9", "149.112.112.112"}, []string{"2620:fe::fe", "2620:fe::9"}, "https://dns.quad9.net/dns-query"},
	{"AdGuard", []string{"94.140.14.14", "94.140.15.15"}, []string{"2a10:50c0::ad1:ff", "2a10:50c0::ad2:ff"}, "https://dns.adguard-dns.com/dns-query"},
}

func Find(name string) Provider {
	for _, p := range Providers {
		if p.Name == name {
			return p
		}
	}
	return Providers[0]
}

type iface struct {
	GUID   string   `json:"guid"`
	Name   string   `json:"name"`
	V4     []string `json:"v4"`
	V6     []string `json:"v6"`
	SetV6  bool     `json:"set_v6"`
	HadDoh bool     `json:"had_doh"`
}

type Backup struct {
	Provider string   `json:"provider"`
	Doh      bool     `json:"doh"`
	AddedEnc []string `json:"added_enc,omitempty"`
	Ifaces   []iface  `json:"ifaces"`
}

const (
	tcpipKey    = `SYSTEM\CurrentControlSet\Services\Tcpip\Parameters\Interfaces\`
	tcpip6Key   = `SYSTEM\CurrentControlSet\Services\Tcpip6\Parameters\Interfaces\`
	dnscacheKey = `SYSTEM\CurrentControlSet\Services\Dnscache\InterfaceSpecificParameters\`
)

func Pending(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

type adapter struct {
	guid, name string
	v6, v6dns  bool
}

func adapters() ([]adapter, error) {
	size := uint32(16 * 1024)
	for i := 0; i < 4; i++ {
		buf := make([]byte, size)
		aa := (*windows.IpAdapterAddresses)(unsafe.Pointer(&buf[0]))
		err := windows.GetAdaptersAddresses(windows.AF_UNSPEC, windows.GAA_FLAG_INCLUDE_GATEWAYS, 0, aa, &size)
		if err == windows.ERROR_BUFFER_OVERFLOW {
			continue
		}
		if err != nil {
			return nil, err
		}
		var out []adapter
		for ; aa != nil; aa = aa.Next {
			if aa.OperStatus != windows.IfOperStatusUp || aa.FirstGatewayAddress == nil || aa.IfType == windows.IF_TYPE_SOFTWARE_LOOPBACK {
				continue
			}
			a := adapter{guid: windows.BytePtrToString(aa.AdapterName), name: windows.UTF16PtrToString(aa.FriendlyName)}
			a.v6 = aa.Ipv6IfIndex != 0
			for d := aa.FirstDnsServerAddress; d != nil; d = d.Next {
				if ip := d.Address.IP(); ip != nil && ip.To4() == nil && ip.IsGlobalUnicast() {
					a.v6dns = true
				}
			}
			out = append(out, a)
		}
		if len(out) == 0 {
			return nil, errors.New("no active adapter")
		}
		return out, nil
	}
	return nil, errors.New("adapter list too large")
}

func staticServers(root, guid string) []string {
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, root+guid, registry.QUERY_VALUE)
	if err != nil {
		return nil
	}
	defer k.Close()
	v, _, err := k.GetStringValue("NameServer")
	if err != nil {
		return nil
	}
	var out []string
	for _, s := range strings.FieldsFunc(v, func(r rune) bool { return r == ',' || r == ' ' || r == ';' }) {
		if net.ParseIP(s) != nil {
			out = append(out, s)
		}
	}
	return out
}

func netsh(args ...string) error {
	cmd := exec.Command("netsh", args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: windows.CREATE_NO_WINDOW}
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("netsh %s: %s", strings.Join(args, " "), strings.TrimSpace(string(out)))
	}
	return nil
}

func setServers(family, name string, servers []string) error {
	if len(servers) == 0 {
		return netsh("interface", family, "set", "dnsservers", "name="+name, "source=dhcp")
	}
	if err := netsh("interface", family, "set", "dnsservers", "name="+name, "source=static", "address="+servers[0], "register=primary", "validate=no"); err != nil {
		return err
	}
	for i, s := range servers[1:] {
		if err := netsh("interface", family, "add", "dnsservers", "name="+name, "address="+s, fmt.Sprintf("index=%d", i+2), "validate=no"); err != nil {
			return err
		}
	}
	return nil
}

func dohFlag(guid, sub, ip string) error {
	k, _, err := registry.CreateKey(registry.LOCAL_MACHINE, dnscacheKey+guid+`\DohInterfaceSettings\`+sub+`\`+ip, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer k.Close()
	return k.SetQWordValue("DohFlags", 1)
}

func keyExists(path string) bool {
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, path, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	k.Close()
	return true
}

func deleteTree(path string) error {
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, path, registry.ALL_ACCESS)
	if err != nil {
		if err == registry.ErrNotExist {
			return nil
		}
		return err
	}
	subs, err := k.ReadSubKeyNames(-1)
	k.Close()
	if err != nil {
		return err
	}
	for _, s := range subs {
		if err := deleteTree(path + `\` + s); err != nil {
			return err
		}
	}
	return registry.DeleteKey(registry.LOCAL_MACHINE, path)
}

func flush() {
	cmd := exec.Command("ipconfig", "/flushdns")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: windows.CREATE_NO_WINDOW}
	_ = cmd.Run()
}

func encryptionKnown(ip string) bool {
	cmd := exec.Command("netsh", "dns", "show", "encryption", "server="+ip)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: windows.CREATE_NO_WINDOW}
	out, err := cmd.CombinedOutput()
	return err == nil && strings.Contains(strings.ToLower(string(out)), "https://")
}

func Apply(p Provider, path string, doh bool) error {
	if Pending(path) {
		if err := Restore(path); err != nil {
			return err
		}
	}
	ads, err := adapters()
	if err != nil {
		return err
	}
	b := Backup{Provider: p.Name, Doh: doh}
	for _, a := range ads {
		b.Ifaces = append(b.Ifaces, iface{
			GUID:   a.guid,
			Name:   a.name,
			V4:     staticServers(tcpipKey, a.guid),
			V6:     staticServers(tcpip6Key, a.guid),
			SetV6:  a.v6 && a.v6dns,
			HadDoh: keyExists(dnscacheKey + a.guid + `\DohInterfaceSettings`),
		})
	}
	save := func() error {
		data, _ := json.MarshalIndent(b, "", "  ")
		return os.WriteFile(path, data, 0o644)
	}
	if err := save(); err != nil {
		return err
	}
	fail := func(err error) error {
		_ = save()
		_ = Restore(path)
		return err
	}
	if doh {
		for _, ip := range append(append([]string{}, p.V4...), p.V6...) {
			if encryptionKnown(ip) {
				continue
			}
			if err := netsh("dns", "add", "encryption", "server="+ip, "dohtemplate="+p.Template, "autoupgrade=yes", "udpfallback=no"); err == nil {
				b.AddedEnc = append(b.AddedEnc, ip)
			}
		}
	}
	for _, f := range b.Ifaces {
		if err := setServers("ipv4", f.Name, p.V4); err != nil {
			return fail(err)
		}
		if doh {
			for _, ip := range p.V4 {
				if err := dohFlag(f.GUID, "Doh", ip); err != nil {
					return fail(err)
				}
			}
		}
		if f.SetV6 {
			if err := setServers("ipv6", f.Name, p.V6); err != nil {
				return fail(err)
			}
			if doh {
				for _, ip := range p.V6 {
					if err := dohFlag(f.GUID, "Doh6", ip); err != nil {
						return fail(err)
					}
				}
			}
		}
	}
	if err := save(); err != nil {
		return fail(err)
	}
	flush()
	return nil
}

func Restore(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var b Backup
	if err := json.Unmarshal(data, &b); err != nil {
		return err
	}
	p := Find(b.Provider)
	var errs []string
	for _, f := range b.Ifaces {
		if err := setServers("ipv4", f.Name, f.V4); err != nil {
			errs = append(errs, err.Error())
		}
		if f.SetV6 {
			if err := setServers("ipv6", f.Name, f.V6); err != nil {
				errs = append(errs, err.Error())
			}
		}
		if !b.Doh {
			continue
		}
		base := dnscacheKey + f.GUID + `\DohInterfaceSettings`
		if f.HadDoh {
			for _, ip := range p.V4 {
				_ = deleteTree(base + `\Doh\` + ip)
			}
			for _, ip := range p.V6 {
				_ = deleteTree(base + `\Doh6\` + ip)
			}
		} else {
			_ = deleteTree(base)
		}
	}
	for _, ip := range b.AddedEnc {
		_ = netsh("dns", "delete", "encryption", "server="+ip)
	}
	flush()
	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "; "))
	}
	return os.Remove(path)
}
