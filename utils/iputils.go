package utils

import (
	"fmt"
	"net"
	"strings"
)

const SpearDefaultPort = 8822

type TCPAddr struct {
	Addr *net.TCPAddr
}

func (t *TCPAddr) String() string {
	if t.Addr == nil {
		return "<nil>"
	}
	return t.Addr.String()
}
func (t *TCPAddr) Set(s string) error {
	var err error
	t.Addr, err = ResolveTCPAddrDefaultPort(s)
	return err

}
func (t *TCPAddr) Type() string {
	return "TCPAddr"
}

func ResolveTCPAddrDefaultPort(input string /*, defaultPort string*/) (*net.TCPAddr, error) {
	/*if defaultPort == "" {
		defaultPort = SpearDefaultPort
	}*/
	defaultPort := SpearDefaultPort
	addr, err := net.ResolveTCPAddr("tcp", input)
	if err != nil && strings.HasSuffix(err.Error(), "missing port in address") {
		addr, err = net.ResolveTCPAddr("tcp", fmt.Sprintf("%s:%d", input, defaultPort))
	}
	if err != nil {
		return nil, err
	}
	if addr.IP == nil {
		addr.IP = net.IPv4zero
	}
	return addr, nil
}
