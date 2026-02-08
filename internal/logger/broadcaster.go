package logger

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"sunnyproxy/internal/rules"
)

type Broadcaster struct {
	mu      sync.RWMutex
	clients map[*websocket.Conn]bool
	console bool
}

var (
	instance *Broadcaster
	once     sync.Once
)

func GetBroadcaster() *Broadcaster {
	once.Do(func() {
		instance = &Broadcaster{
			clients: make(map[*websocket.Conn]bool),
			console: true,
		}
	})
	return instance
}

func (b *Broadcaster) SetConsoleOutput(enabled bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.console = enabled
}

func (b *Broadcaster) AddClient(conn *websocket.Conn) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.clients[conn] = true
}

func (b *Broadcaster) RemoveClient(conn *websocket.Conn) {
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.clients, conn)
	conn.Close()
}

func (b *Broadcaster) Broadcast(entry rules.LogEntry) {
	b.mu.RLock()
	consoleEnabled := b.console
	clients := make([]*websocket.Conn, 0, len(b.clients))
	for c := range b.clients {
		clients = append(clients, c)
	}
	b.mu.RUnlock()

	if consoleEnabled {
		b.printToConsole(entry)
	}

	data, err := json.Marshal(entry)
	if err != nil {
		return
	}

	for _, client := range clients {
		err := client.WriteMessage(websocket.TextMessage, data)
		if err != nil {
			b.RemoveClient(client)
		}
	}
}

func (b *Broadcaster) printToConsole(entry rules.LogEntry) {
	timestamp := entry.Timestamp.Format("15:04:05")

	switch entry.Type {
	case "request":
		modified := ""
		if entry.Modified {
			modified = " [MODIFIED]"
		}
		log.Printf("[%s] %s %s %s%s\n", timestamp, entry.Type, entry.Method, entry.URL, modified)
	case "response":
		log.Printf("[%s] %s %d %s\n", timestamp, entry.Type, entry.StatusCode, entry.URL)
	case "error":
		log.Printf("[%s] ERROR: %s - %s\n", timestamp, entry.URL, entry.Error)
	case "token":
		log.Printf("[%s] TOKEN EXTRACTED: %s\n", timestamp, entry.URL)
	default:
		log.Printf("[%s] %s: %s\n", timestamp, entry.Type, entry.URL)
	}
}

func (b *Broadcaster) LogRequest(method, url string, modified bool, appliedRules []string) {
	entry := rules.LogEntry{
		ID:           fmt.Sprintf("%d", time.Now().UnixNano()),
		Timestamp:    time.Now(),
		Type:         "request",
		Method:       method,
		URL:          url,
		Modified:     modified,
		RulesApplied: appliedRules,
	}
	b.Broadcast(entry)
}

func (b *Broadcaster) LogResponse(method, url string, statusCode int) {
	entry := rules.LogEntry{
		ID:         fmt.Sprintf("%d", time.Now().UnixNano()),
		Timestamp:  time.Now(),
		Type:       "response",
		Method:     method,
		URL:        url,
		StatusCode: statusCode,
	}
	b.Broadcast(entry)
}

func (b *Broadcaster) LogError(url, errMsg string) {
	entry := rules.LogEntry{
		ID:        fmt.Sprintf("%d", time.Now().UnixNano()),
		Timestamp: time.Now(),
		Type:      "error",
		URL:       url,
		Error:     errMsg,
	}
	b.Broadcast(entry)
}

func (b *Broadcaster) LogToken(name, value, url string) {
	entry := rules.LogEntry{
		ID:        fmt.Sprintf("%d", time.Now().UnixNano()),
		Timestamp: time.Now(),
		Type:      "token",
		URL:       url,
		Headers:   map[string]string{name: value},
	}
	b.Broadcast(entry)
}
