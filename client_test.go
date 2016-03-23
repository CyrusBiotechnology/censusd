package censusd

import (
	"fmt"
	"os"
	"testing"
)

var client = Client{}

func TestClientCanSendMessage(t *testing.T) {
	client.Broadcast("garbage!")
}

func TestMain(m *testing.M) {
	str, _ := SecureRandomAlphaString(10)
	client = Client{
		UID: fmt.Sprint("test-client-", str),
	}
	os.Exit(m.Run())
}
