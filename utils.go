package censusd

import (
	"crypto/rand"
	"encoding/binary"
	"errors"
	"net"
	"regexp"
)

const (
	letterBytes   = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ" // 52 possibilities
	letterIdxBits = 6                                                      // 6 bits to represent 64 possibilities / indexes
	letterIdxMask = 1<<letterIdxBits - 1                                   // All 1-bits, as many as letterIdxBits
)

var uidRegex = regexp.MustCompile("^[A-Za-z0-9]{32}:")
var formatErr = errors.New("First 32 bytes of a message should contain a UID, followed by a colon")

// SecureRandomBytes returns the requested number of bytes using crypto/rand
func SecureRandomBytes(length int) ([]byte, error) {
	var randomBytes = make([]byte, length)
	_, err := rand.Read(randomBytes)
	if err != nil {
		return randomBytes, errors.New("Unable to generate random bytes")
	}
	return randomBytes, nil
}

func SecureRandomAlphaString(length int) (str string, err error) {
	result := make([]byte, length)
	bufferSize := int(float64(length) * 1.3)
	for i, j, randomBytes := 0, 0, []byte{}; i < length; j++ {
		if j%bufferSize == 0 {
			randomBytes, err = SecureRandomBytes(bufferSize)
			if err != nil {
				return string(result), err
			}
		}
		if idx := int(randomBytes[j%length] & letterIdxMask); idx < len(letterBytes) {
			result[i] = letterBytes[idx]
			i++
		}
	}
	return string(result), nil
}

func processMessage(buffer []byte) (uid string, err error) {
	if uidRegex.Match(buffer[0:33]) {
		uid = string(buffer[0:32])
		return uid, nil
	} else {
		return uid, formatErr
	}
}

func GetAllBroadcastIPNets() ([]*net.IPNet, error) {
	ipnets := []*net.IPNet{}
	interfaces, err := net.Interfaces()

	if err != nil {
		return []*net.IPNet{}, err
	}

	for _, i := range interfaces {
		if i.Flags&net.FlagBroadcast != 0 {
			addrs, err := i.Addrs()
			if err != nil {
				println(err)
			}
			for _, addr := range addrs {
				switch addr.(type) {
				case *net.IPNet:
					ipnets = append(ipnets, addr.(*net.IPNet))
				}
			}
		}
	}
	return ipnets, nil
}

func lastAddr(n *net.IPNet) (net.IP, error) { // works when the n is a prefix, otherwise...
	if n.IP.To4() == nil {
		return net.IP{}, errors.New("does not support IPv6 addresses.")
	}
	ip := make(net.IP, len(n.IP.To4()))
	binary.BigEndian.PutUint32(ip, binary.BigEndian.Uint32(n.IP.To4())|^binary.BigEndian.Uint32(net.IP(n.Mask).To4()))
	return ip, nil
}

func GetBroadcastAddresses() ([]net.IP, error) {
	ipnets, _ := GetAllBroadcastIPNets()
	bcastAddrs := []net.IP{}
	for _, n := range ipnets {
		bcast, err := lastAddr(n)
		if err != nil {
			continue
		}
		bcastAddrs = append(bcastAddrs, bcast)
	}
	return bcastAddrs, nil
}
