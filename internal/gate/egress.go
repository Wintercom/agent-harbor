package gate

import (
	"fmt"
	"net"
	"strings"
)

// egressAllowLoopback 仅测试打开；生产默认拒绝回环/私网出站。
var egressAllowLoopback bool

// lookupIP 可在单测中替换。
var lookupIP = net.LookupIP

// ValidatePublicHostname 要求主机名可解析且全部地址为公网全局单播。
// 拒绝字面量/解析结果中的回环、私网、链路本地、组播、未指定与 CGNAT。
func ValidatePublicHostname(host string) error {
	host = strings.ToLower(strings.TrimSpace(host))
	if host == "" || strings.ContainsAny(host, "/:#?@") {
		return fmt.Errorf("gate: invalid egress host")
	}
	if ip := net.ParseIP(host); ip != nil {
		if isBlockedEgressIP(ip) {
			return fmt.Errorf("gate: blocked egress address")
		}
		return nil
	}
	ips, err := lookupIP(host)
	if err != nil || len(ips) == 0 {
		return fmt.Errorf("gate: dns lookup failed")
	}
	for _, ip := range ips {
		if isBlockedEgressIP(ip) {
			return fmt.Errorf("gate: host resolves to blocked address")
		}
	}
	return nil
}

func isBlockedEgressIP(ip net.IP) bool {
	if ip == nil {
		return true
	}
	if egressAllowLoopback && ip.IsLoopback() {
		return false
	}
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() ||
		ip.IsMulticast() || ip.IsUnspecified() || ip.IsInterfaceLocalMulticast() {
		return true
	}
	if v4 := ip.To4(); v4 != nil {
		// CGNAT 100.64.0.0/10
		if v4[0] == 100 && v4[1] >= 64 && v4[1] <= 127 {
			return true
		}
		// 文档/基准 0.0.0.0/8 已由 IsUnspecified 覆盖部分；拒绝 0.x
		if v4[0] == 0 {
			return true
		}
	}
	return false
}
