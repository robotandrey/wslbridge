package init_ubuntu

import (
	"fmt"
	"net"
	"os"
	"strings"

	"wslbridge/internal/config"
	appruntime "wslbridge/internal/runtime"
)

func detectSocksGateway(rt appruntime.Runtime, cfg config.Config, currentDefaultRouteLine string) (string, error) {
	// 1) current route
	if gw, ok := parseViaIP(currentDefaultRouteLine); ok {
		return gw, nil
	}
	// 2) saved default route file (may contain via)
	if b, err := os.ReadFile(rt.Paths.DefaultRouteFile); err == nil {
		if gw, ok := parseViaIP(string(b)); ok {
			return gw, nil
		}
	}
	// 3) config (values.local)
	if cfg.Socks.Host != "" {
		return cfg.Socks.Host, nil
	}
	// 4) infer from eth0
	gw, err := inferWSLGatewayFromEth0(rt)
	if err != nil {
		return "", fmt.Errorf("could not detect SOCKS gateway: %w", err)
	}
	return gw, nil
}

func parseViaIP(routeLine string) (string, bool) {
	fields := strings.Fields(strings.TrimSpace(routeLine))
	for i := 0; i < len(fields)-1; i++ {
		if fields[i] == "via" {
			return fields[i+1], true
		}
	}
	return "", false
}

func inferWSLGatewayFromEth0(rt appruntime.Runtime) (string, error) {
	out, err := rt.Runner.RunCapture("bash", "-lc", `ip -4 addr show dev eth0 | awk '/inet /{print $2; exit}'`)
	if err != nil {
		return "", fmt.Errorf("read eth0 addr: %w", err)
	}
	cidr := strings.TrimSpace(out)
	if cidr == "" {
		return "", fmt.Errorf("could not read eth0 IPv4 CIDR")
	}

	hostIP, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		return "", fmt.Errorf("parse eth0 cidr %q: %w", cidr, err)
	}

	host4 := hostIP.To4()
	if host4 == nil {
		return "", fmt.Errorf("eth0 ip is not ipv4")
	}

	// Prefer nameserver in the same subnet (often Windows host IP in WSL).
	if ns, ok := readResolvConfNameserver("/etc/resolv.conf"); ok {
		if nsIP := net.ParseIP(ns).To4(); nsIP != nil {
			if ipnet.Contains(nsIP) && !nsIP.Equal(host4) {
				return nsIP.String(), nil
			}
		}
	}

	netIP := ipnet.IP.To4()
	if netIP == nil {
		return "", fmt.Errorf("eth0 ip is not ipv4")
	}
	gw, err := incrementIPv4(netIP)
	if err != nil {
		return "", fmt.Errorf("infer gateway from eth0 network: %w", err)
	}
	if gw.Equal(host4) {
		gw, err = incrementIPv4(gw)
		if err != nil {
			return "", fmt.Errorf("infer gateway from eth0 network: %w", err)
		}
	}
	return gw.String(), nil
}

func readResolvConfNameserver(path string) (string, bool) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", false
	}
	for _, line := range strings.Split(string(b), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) >= 2 && fields[0] == "nameserver" {
			return fields[1], true
		}
	}
	return "", false
}

func incrementIPv4(ip net.IP) (net.IP, error) {
	ip = ip.To4()
	if ip == nil {
		return nil, fmt.Errorf("ip is not ipv4")
	}
	out := make(net.IP, len(ip))
	copy(out, ip)
	for i := len(out) - 1; i >= 0; i-- {
		out[i]++
		if out[i] != 0 {
			return out, nil
		}
	}
	return nil, fmt.Errorf("ipv4 overflow")
}
