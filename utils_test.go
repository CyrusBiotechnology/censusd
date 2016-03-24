package censusd

import "testing"

func TestSecureRandomAlphaString(t *testing.T) {
	str, err := SecureRandomAlphaString(100)
	if err != nil {
		t.Fatal(err)
	}
	if len(str) != 100 {
		t.Fatal("Bad length")
	}
}

func TestSendUDPBroadcastMessageOnAllInterfaces(t *testing.T) {
	SendUDPBroadcastMessageOnAllInterfaces([]byte("garbage!"), 19091)
}
