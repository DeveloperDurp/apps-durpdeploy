package runner

import "sync"

type LogBroker struct {
	mu      sync.RWMutex
	clients map[int64]map[chan string]struct{} // deploymentID -> clients
}

func NewLogBroker() *LogBroker {
	return &LogBroker{clients: make(map[int64]map[chan string]struct{})}
}

func (b *LogBroker) Subscribe(deploymentID int64) chan string {
	ch := make(chan string, 64)
	b.mu.Lock()
	if b.clients[deploymentID] == nil {
		b.clients[deploymentID] = make(map[chan string]struct{})
	}
	b.clients[deploymentID][ch] = struct{}{}
	b.mu.Unlock()
	return ch
}

func (b *LogBroker) Unsubscribe(deploymentID int64, ch chan string) {
	b.mu.Lock()
	if clients, ok := b.clients[deploymentID]; ok {
		delete(clients, ch)
		if len(clients) == 0 {
			delete(b.clients, deploymentID)
		}
	}
	b.mu.Unlock()
	close(ch)
}

func (b *LogBroker) Broadcast(deploymentID int64, line string) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if clients, ok := b.clients[deploymentID]; ok {
		for ch := range clients {
			select {
			case ch <- line:
			default:
			}
		}
	}
}
