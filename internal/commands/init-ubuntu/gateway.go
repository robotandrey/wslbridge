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

	_, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		return "", fmt.Errorf("parse eth0 cidr %q: %w", cidr, err)
	}

	nip := ipnet.IP.To4()
	if nip == nil {
		return "", fmt.Errorf("eth0 ip is not ipv4")
	}
	nip[3] += 1 // network + 1
	return nip.String(), nil
}
