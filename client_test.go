package censusd

import (
	"fmt"
	"net"
	"os"
	"testing"
)

var client = Client{}

func TestClientCanSendMessage(t *testing.T) {
	client.Send("garbage!")
}

func TestMain(m *testing.M) {
	str, _ := SecureRandomAlphaString(10)
	client = Client{
		Address: &net.UDPAddr{
			IP:   net.IPv4(0, 0, 0, 0),
			Port: 19091,
		},
		UID: fmt.Sprint("test-client-", str),
	}
	os.Exit(m.Run())
}
