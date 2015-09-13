package pir

import (
	"fmt"
	"net"
	"net/url"
)

type CannotFindIPErr struct{}

func (c CannotFindIPErr) Error() string {
	return "Cannot find local IP"
}

// Adapted from http://stackoverflow.com/a/31551220
func GetLocalIP() (string, error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "", err
	}

	for _, address := range addrs {
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String(), nil
			}
		}
	}

	return "", &CannotFindIPErr{}
}

type UnsupportedURISpecErr struct {
	uriSpec *url.URL
}

func (u UnsupportedURISpecErr) Error() string {
	return fmt.Sprintf("Unsupported URI spec [%s]", u.uriSpec)
}

func ResolveURISpec(uriSpecStr string) (net.Addr, error) {
	uriSpec, err := url.Parse(uriSpecStr)
	if err != nil {
		return nil, err
	}

	switch uriSpec.Scheme {
	case "tcp":
		return net.ResolveTCPAddr("tcp", uriSpec.Host)
	case "udp":
		return net.ResolveUDPAddr("udp", uriSpec.Host)
	default:
		return nil, &UnsupportedURISpecErr{uriSpec}
	}
}
