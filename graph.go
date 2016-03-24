package censusd

import (
	"container/list"
	"encoding/hex"
	"sync"
	"time"
)

type NodeGraph struct {
	Stats   *Stats                   // Output to callers
	Nodes   map[string]*list.Element // UID: pointer to item in history
	History *list.List               // Stores the time and sender of messages received in order
	Mutex   sync.RWMutex
}

func NewNodeGraph() NodeGraph {
	return NodeGraph{
		Stats: &Stats{
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
func (ng *NodeGraph) updateNode(nodeUID []byte) {
	uidString := string(nodeUID)
	ng.Mutex.Lock()
	defer ng.Mutex.Unlock()
	if ng.hasNode(uidString) {
		ng.History.Remove(ng.Nodes[uidString])
	} else {
		Info.Println("new node:", hex.EncodeToString(nodeUID))
		ng.Stats.Mutex.Lock()
		ng.Stats.Nodes++
		ng.Stats.Mutex.Unlock()
	}
	ng.Nodes[uidString] = ng.History.PushFront(Beacon{
		Time:   time.Now(),
		Sender: uidString,
	})
}

func (ng *NodeGraph) calcInterval() time.Duration {
	ng.Mutex.Lock()
	defer ng.Mutex.Unlock()
	return time.Second * time.Duration(len(ng.Nodes)+1)
}

func (ng *NodeGraph) GC() {
	ng.Mutex.Lock()
	defer ng.Mutex.Unlock()
	if ng.History.Len() > 0 {
		event := ng.History.Back().Value.(Beacon)
		threshold := time.Now().Add(-time.Duration(ng.History.Len()+5) * time.Second)
		for event.Time.Before(threshold) {
			Info.Println(hex.EncodeToString([]byte(event.Sender)), "went away")
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
			if ng.History.Len() == 0 {
				break
			}
			event = ng.History.Back().Value.(Beacon)
		}
	}
}
