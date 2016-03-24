package censusd

import (
	"encoding/binary"
	"errors"
	"net"
)

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

func SendUDPBroadcastMessageOnAllInterfaces(message []byte, port int) {
	ips, _ := GetBroadcastAddresses()
	for _, ip := range ips {
		SendUDP4(&net.UDPAddr{
			IP:   ip,
			Port: port,
		}, message)
	}
}

func SendUDP4(address *net.UDPAddr, message []byte) error {
	socket, err := net.DialUDP("udp4", nil, address)
	defer socket.Close()
	if err != nil {
		return err
	}
	socket.Write(message)
	return nil
}
