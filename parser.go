package lvs

import (
	"errors"
	"net"
	"strconv"
)

var (
	EOFError       = errors.New("ipvsadm terminated prematurely")
	UnexpecedToken = errors.New("Unexpected Token")
)

func parseHostPort(hostPort string) (string, int) {
	host, port, err := net.SplitHostPort(hostPort)
	if err != nil {
		return hostPort, 0
	}
	intPort, err := strconv.Atoi(port)
	if err != nil {
		return hostPort, 0
	}
	return host, intPort
}
