package ipx

import "net"

func ParseToIpString(addr net.Addr) string {
	host, _, _ := net.SplitHostPort(addr.String())
	return host
}
