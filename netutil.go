package pir

import (
	"net"
)

type CannotFindIPErr struct{}

func (c CannotFindIPErr) Error() string {
	return "Cannot find local IP"
}

// Adapted frmo http://stackoverflow.com/a/31551220
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
