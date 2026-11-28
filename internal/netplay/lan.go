package netplay

import (
	"net"
	"sort"
	"strings"
)

// Address is one way friends may reach this machine.
type Address struct {
	IP    string
	Iface string // the network adapter's name
	Kind  string // "LAN", "Tailscale", "ZeroTier" or "" when unknown
}

// HostAddresses lists this machine's IPv4 addresses a friend could try, the
// most likely first: the home or office LAN, then Tailscale and ZeroTier
// (what to use over the Internet without opening ports on the router), then
// anything else that is not loopback or link-local.
func HostAddresses() []Address {
	ifs, err := net.Interfaces()
	if err != nil {
		return nil
	}
	var out []Address
	for _, ifc := range ifs {
		if ifc.Flags&net.FlagUp == 0 || ifc.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := ifc.Addrs()
		if err != nil {
			continue
		}
		for _, a := range addrs {
			ipn, ok := a.(*net.IPNet)
			if !ok {
				continue
			}
			ip := ipn.IP.To4()
			if ip == nil || ip.IsLoopback() || ip.IsLinkLocalUnicast() {
				continue
			}
			out = append(out, Address{IP: ip.String(), Iface: ifc.Name, Kind: kindOf(ifc.Name, ip)})
		}
	}
	rank := func(a Address) int {
		switch a.Kind {
		case "LAN":
			return 0
		case "Tailscale":
			return 1
		case "ZeroTier":
			return 2
		}
		return 3
	}
	sort.SliceStable(out, func(i, j int) bool { return rank(out[i]) < rank(out[j]) })
	return out
}

var (
	_, cgnat, _ = net.ParseCIDR("100.64.0.0/10") // Tailscale's range
	virtualKeys = []string{"vethernet", "vmware", "virtualbox", "vbox", "docker", "wsl", "hyper-v", "loopback", "br-", "veth"}
)

func kindOf(name string, ip net.IP) string {
	n := strings.ToLower(name)
	switch {
	case cgnat.Contains(ip) || strings.Contains(n, "tailscale"):
		return "Tailscale"
	case strings.Contains(n, "zerotier") || strings.HasPrefix(n, "zt"):
		return "ZeroTier"
	}
	for _, k := range virtualKeys {
		if strings.Contains(n, k) {
			return "" // a virtual switch: friends cannot reach it
		}
	}
	if ip.IsPrivate() {
		return "LAN"
	}
	return ""
}
