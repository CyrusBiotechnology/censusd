package censusd

import (
	"bytes"
	"net"
	"text/template"
	"time"
)

type Client struct {
	UID string // Identify ourselves to peers
}

func SendUDP(address *net.UDPAddr, message string) error {
	socket, err := net.DialUDP("udp4", nil, address)
	defer socket.Close()
	if err != nil {
		return err
	}
	socket.Write([]byte(message))
	return nil
}

func (cl *Client) BroadcastOnAllInterfaces(message string) {
	ips, _ := GetBroadcastAddresses()
	for _, ip := range ips {
		SendUDP(&net.UDPAddr{
			IP:   ip,
			Port: 19091,
		}, message)
	}
}

// Keep the swarm notified of our existence.
func (cl *Client) DoBeacon(stop <-chan struct{}, ng *NodeGraph) error {
	msgTmpl, err := template.New("message").Parse("{{.UID}}:")
	buf := new(bytes.Buffer)
	err = msgTmpl.Execute(buf, cl)
	if err != nil {
		Error.Println("error executing template")
		return err
	}
	msg := buf.String()
	if err != nil {
		return err
	}
	for {
		after := time.After(ng.calcInterval())
		select {
		case <-stop:
			return nil
		case <-after:
			cl.BroadcastOnAllInterfaces(msg)
			after = time.After(ng.calcInterval())
		}
	}
}
