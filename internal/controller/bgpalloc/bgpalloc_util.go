package bgpalloc

import (
	"fmt"
	"net"
	"strings"
)

func parseIPRange(s string) ([]net.IP, error) {
	parts := strings.Split(s, "-")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid range format")
	}

	start := net.ParseIP(parts[0])
	end := net.ParseIP(parts[1])
	if start == nil || end == nil {
		return nil, fmt.Errorf("invalid IP address")
	}

	start = start.To4()
	end = end.To4()
	if start == nil || end == nil {
		return nil, fmt.Errorf("only IPv4 supported for now")
	}

	var ips []net.IP
	for ip := start; !ipAfter(ip, end); ip = nextIP(ip) {
		ipCopy := make(net.IP, len(ip))
		copy(ipCopy, ip)
		ips = append(ips, ipCopy)
	}

	return ips, nil
}

func nextIP(ip net.IP) net.IP {
	ip = ip.To4()
	result := make(net.IP, len(ip))
	copy(result, ip)
	for i := len(result) - 1; i >= 0; i-- {
		result[i]++
		if result[i] != 0 {
			break
		}
	}
	return result
}

func ipAfter(a, b net.IP) bool {
	for i := 0; i < len(a); i++ {
		if a[i] > b[i] {
			return true
		} else if a[i] < b[i] {
			return false
		}
	}
	return false
}
