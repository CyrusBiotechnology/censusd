package censusd

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"net"
	"os"
	"sync"
	"time"
)

type Server struct {
	Address *net.UDPAddr // Listen and broadcast
	SID     []byte       // Name identifies us as part of a swarm
	UID     []byte       // UID identifies us within a swarm
	Graph   *NodeGraph
}

type Beacon struct {
	Time   time.Time
	Sender string
}

type Stats struct {
	Nodes int // Number of nodes in the swarm
	Mutex sync.RWMutex
}

func NewServer(listen *net.UDPAddr, group string, id string) (*Server, error) {
	server := &Server{}

	// Group generation
	if len(group) == 0 {
		return server, errors.New("you must provide a group name")
	}
	sidHash := sha1.Sum([]byte(group))
	server.SID = sidHash[:]

	// UID generation
	uidHash := [20]byte{}
	if len(id) == 0 {
		Info.Println("id not provided, generating one now")
		genID, err := os.Hostname()
		if err != nil {
			Info.Println("failed to get hostname, falling back to random string")
			genID, err = SecureRandomAlphaString(64)
			if err != nil {
				return server, errors.New("failed to get both hostname and a random string")
			}
		} else {
			Info.Println("using hostname", genID, "to generate UID")
		}
		uidHash = sha1.Sum([]byte(genID))
	} else {
		uidHash = sha1.Sum([]byte(id))
	}
	server.UID = uidHash[:]

	server.Address = listen
	graph := NewNodeGraph()
	server.Graph = &graph

	return server, nil
}

// Keep the swarm notified of our existence.
func (s *Server) DoBeacon(stop <-chan struct{}, ng *NodeGraph) error {
	msg := append(s.SID, s.UID...)
	for {
		after := time.After(ng.calcInterval())
		select {
		case <-stop:
			return nil
		case <-after:
			SendUDPBroadcastMessageOnAllInterfaces(msg, 19091)
			after = time.After(ng.calcInterval())
		}
	}
}

func (s *Server) messageIngestor(sm *sockMonster) {
	messages := make(chan []byte, 10)
	uidC := make(chan []byte, 10)
	defer close(messages)
	defer close(uidC)
	go s.messageParser(messages, uidC)
	go s.graphUpdater(uidC)
	for {

		buf, err := sm.read()
		if err != nil {
			panic(err)
		}
		if err != nil {
			Info.Println(err)
			continue
		}
		messages <- buf
	}
}

func (s *Server) messageParser(messages <-chan []byte, uidC chan<- []byte) {
	swarmIdLen := len(s.SID)
	idOffset := swarmIdLen
	idEnd := idOffset + len(s.UID)
	for message := range messages {
		if !bytes.Equal(message[:swarmIdLen], s.SID) {
			// this message was intended for members of a different swarm
			continue
		}
		if bytes.Equal(message[idOffset:idEnd], s.UID) {
			// this message was a beacon from us
			continue
		}
		uidC <- message[idOffset:idEnd]
	}
}

func (s *Server) graphUpdater(uidC <-chan []byte) {
	for uid := range uidC {
		s.Graph.updateNode(uid)
	}
}

type sockMonster struct {
	Proto  string       // only udp4 is supported for now
	Listen *net.UDPAddr // where to listen
	Socket *net.UDPConn // our socket
}

func newSockMonster(proto string, laddr *net.UDPAddr) (*sockMonster, error) {
	socket, err := net.ListenUDP(proto, laddr)
	return &sockMonster{
		Socket: socket,
	}, err
}

func (sm *sockMonster) reconnect() (err error) {
	for i := 0; i < 10; i++ {
		time.Sleep(time.Millisecond * 500)
		socket, err := net.ListenUDP(sm.Proto, sm.Listen)
		if err == nil {
			sm.Socket = socket
			return nil
		}
	}
	return err
}

func (sm *sockMonster) read() (buf []byte, err error) {
	buf = make([]byte, 4096)
	for i := 0; i < 10; i++ {
		_, _, err = sm.Socket.ReadFromUDP(buf)
		if err == nil {
			return buf, nil
		}
		time.Sleep(time.Millisecond * 500)
		sm.reconnect()
	}
	return []byte{}, err
}

// Census server listens for UDP broadcast packets and updates graph data.
// Note that this may not work "out of the box" across network boundaries
// depending on program and networking configuration.
func (s *Server) Serve(exit <-chan struct{}, proto string) (*Stats, error) {
	Info.Println("swarm id:", hex.EncodeToString(s.SID), "id:", hex.EncodeToString(s.UID))

	gc := time.NewTicker(time.Second)
	sm, err := newSockMonster(proto, s.Address)
	if err != nil {
		panic(err)
	}
	Info.Println("listening at:", s.Address.IP, s.Address.Port)
	go s.DoBeacon(exit, s.Graph)
	go s.messageIngestor(sm)

	go func() {
		for {
			select {
			case <-exit:
				gc.Stop()
				return
			case <-gc.C:
				s.Graph.GC()
			}
		}
	}()

	return s.Graph.Stats, nil
}
