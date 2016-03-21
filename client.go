package census

import (
	"bytes"
	"net"
	"text/template"
	"time"
)

type Client struct {
	Broadcast *net.UDPAddr // Where to sent packets to
	UID       string       // Identify ourselves to peers
}

func (cl *Client) send(message string) error {
	socket, err := net.DialUDP("udp4", nil, &net.UDPAddr{
		IP:   cl.Broadcast.IP,
		Port: cl.Broadcast.Port,
	})
	defer socket.Close()

	if err != nil {
		return err
	}
	socket.Write([]byte(message))
	return nil
}

// Keep the swarm notified of our existence.
func (cl *Client) doBeacon(stop <-chan struct{}, ng *NodeGraph) error {
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
			cl.send(msg)
			after = time.After(ng.calcInterval())
		}
	}
}
