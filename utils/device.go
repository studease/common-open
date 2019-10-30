package utils

import (
	"net"
	"regexp"
	"strings"
)

var (
	ip4Re, _ = regexp.Compile("^\\d+\\.\\d+\\.\\d+\\.\\d+$")
)

// Mac returns the using mac address
func Mac() string {
	i, err := net.Interfaces()
	if err != nil {
		return ""
	}

	for _, v := range i {
		if (v.Flags & net.FlagUp) == 0 {
			continue
		}

		addrs, err := v.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			ipnet, ok := addr.(*net.IPNet)
			if ok && !ipnet.IP.IsLoopback() {
				if ip4Re.MatchString(ipnet.IP.String()) {
					return strings.Join(strings.Split(v.HardwareAddr.String(), ":"), "")
				}
			}
		}
	}

	return ""
}
