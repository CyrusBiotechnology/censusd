package censusd

import (
	"net"
	"sync"
	"time"
)

type Server struct {
	Address *net.UDPAddr // Listen and broadcast
	UID     string       // Identify us to other peers in the swarm
}

type Beacon struct {
	Time   time.Time
	Sender string
}

type Stats struct {
	Nodes int // Number of nodes in the swarm
	Mutex sync.RWMutex
}

// Census server listens for UDP broadcast packets and updates graph data.
// Note that this may not work "out of the box" across network boundaries
// depending on program and networking configuration.
func Serve(exit <-chan struct{}, proto string, listen *net.UDPAddr) (*Stats, error) {
	uid, err := SecureRandomAlphaString(32)
	if err != nil {
		return &Stats{}, nil
	}
	Info.Println("uid:", uid)
	client := Client{
		UID: uid,
	}
	graph := NewNodeGraph()
	gc := time.NewTicker(time.Second)
	socket, err := net.ListenUDP(proto, listen)
	if err != nil {
		return &Stats{}, err
	}
	Info.Println("listening at:", listen.IP, listen.Port)
	go client.DoBeacon(exit, &graph)

	go func() {
		for {
			select {
			case <-exit:
				return
			case <-gc.C:
				graph.gc()
			default:
				buf := make([]byte, 4096)
				_, _, err := socket.ReadFromUDP(buf)
				if err != nil {
					Info.Println(err)
					continue
				}
				uid, err := processMessage(buf)
				if err != nil {
					Info.Println(err)
					continue
				}
				graph.updateNode(uid)
			}
		}
	}()

	return &graph.Stats, nil
}
