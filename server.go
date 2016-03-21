package census

import (
	"container/list"
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

type NodeGraph struct {
	Stats   Stats                    // Output to callers
	Nodes   map[string]*list.Element // UID: pointer to item in history
	History *list.List               // Stores the time and sender of messages received in order
	Mutex   sync.RWMutex
}

func NewNodeGraph() NodeGraph {
	return NodeGraph{
		Stats: Stats{
			Nodes: 0,
			Mutex: sync.RWMutex{},
		},
		Nodes:   make(map[string]*list.Element),
		History: list.New(),
		Mutex:   sync.RWMutex{},
	}
}

func (ng *NodeGraph) hasNode(UID string) bool {
	if _, ok := ng.Nodes[UID]; ok {
		return true
	} else {
		return false
	}
}

// Update node adds or updates a node.
func (ng *NodeGraph) updateNode(nodeUID string) {
	ng.Mutex.Lock()
	defer ng.Mutex.Unlock()
	if ng.hasNode(nodeUID) {
		ng.History.Remove(ng.Nodes[nodeUID])
	} else {
		Info.Println("new node:", nodeUID)
		ng.Stats.Mutex.Lock()
		ng.Stats.Nodes++
		ng.Stats.Mutex.Unlock()
	}
	ng.Nodes[nodeUID] = ng.History.PushFront(Beacon{
		Time:   time.Now(),
		Sender: nodeUID,
	})
}

func (ng *NodeGraph) calcInterval() time.Duration {
	ng.Mutex.Lock()
	defer ng.Mutex.Unlock()
	return time.Second * time.Duration(len(ng.Nodes)+1)
}

func (ng *NodeGraph) gc() {
	ng.Mutex.Lock()
	defer ng.Mutex.Unlock()
	if ng.History.Len() > 0 {
		event := ng.History.Back().Value.(Beacon)
		threshold := time.Now().Add(-time.Duration(ng.History.Len()+5) * time.Second)
		for event.Time.Before(threshold) {
			delete(ng.Nodes, event.Sender)
			// Locking outside is O(n-1), but will block callers for the entire
			// duration of the GC run. Since garbage collection is only expensive
			// when the cluster is scaling down rapidly, on average nodes will have
			// spare CPU resources this is not a big deal and allows us to continue
			// draining our network buffer during GC operations.
			ng.Stats.Mutex.Lock()
			ng.Stats.Nodes--
			ng.Stats.Mutex.Unlock()
			ng.History.Remove(ng.History.Back())
			event = ng.History.Back().Value.(Beacon)
		}
	}
}

// Census server listens for UDP broadcast packets and updates graph data.
// Note that this may not work "out of the box" across network boundaries
// depending on program and networking configuration.
func Serve(exit <-chan struct{}, proto string, listen *net.UDPAddr, bcast *net.UDPAddr) (*Stats, error) {
	uid, err := SecureRandomAlphaString(32)
	if err != nil {
		return &Stats{}, nil
	}
	Info.Println("uid:", uid)
	client := Client{
		Broadcast: bcast,
		UID:       uid,
	}
	graph := NewNodeGraph()
	gc := time.NewTicker(time.Second)
	socket, err := net.ListenUDP(proto, listen)
	if err != nil {
		return &Stats{}, err
	}
	Info.Println("listening at:", listen.IP, listen.Port)
	go client.doBeacon(exit, &graph)

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
