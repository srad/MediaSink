package network

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
)

type SocketEventName string

const (
	ChannelOnlineEvent    SocketEventName = "channel:online"
	ChannelOfflineEvent   SocketEventName = "channel:offline"
	ChannelStartEvent     SocketEventName = "channel:start"
	ChannelThumbnailEvent SocketEventName = "channel:thumbnail"

	JobCreateEvent      SocketEventName = "job:create"
	JobStartEvent       SocketEventName = "job:start"
	JobProgressEvent    SocketEventName = "job:progress"
	JobDoneEvent        SocketEventName = "job:done"
	JobActivate         SocketEventName = "job:activate"
	JobDeactivate       SocketEventName = "job:deactivate"
	JobErrorEvent       SocketEventName = "job:error"
	JobPreviewDoneEvent SocketEventName = "job:preview:done"
	JobDeleteEvent      SocketEventName = "job:delete"

	RecordingAddEvent SocketEventName = "recording:add"
)

var (
	// Queue size.
	broadCastChannel = make(chan SocketEvent, 1000)
	upGrader         = websocket.Upgrader{CheckOrigin: func(r *http.Request) bool {
		return true
	}}
	dispatcher = wsDispatcher{}
)

type SocketEvent struct {
	Name SocketEventName `json:"name"`
	Data interface{}     `json:"data"`
}

func BroadCastClients(name SocketEventName, data interface{}) {
	broadCastChannel <- SocketEvent{Name: name, Data: data}
}

type wsDispatcher struct {
	listeners []wsConnection
	mu        sync.RWMutex
}

func (d *wsDispatcher) addWs(ws wsConnection) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.listeners = append(d.listeners, ws)
}

func (d *wsDispatcher) broadCast(msg SocketEvent) {
	d.mu.RLock()
	listeners := make([]wsConnection, len(d.listeners))
	copy(listeners, d.listeners)
	d.mu.RUnlock()

	var toRemove []*websocket.Conn

	for _, l := range listeners {
		if err := l.send(msg); err != nil {
			log.Errorf("[broadCast] send error: %s", err)
			toRemove = append(toRemove, l.ws)
		}
	}

	// Clean up outside read lock
	for _, ws := range toRemove {
		d.rmWs(ws)
	}
}

func (p *wsConnection) send(v interface{}) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.ws.WriteJSON(v)
}

func (d *wsDispatcher) rmWs(ws *websocket.Conn) {
	d.mu.Lock()
	defer d.mu.Unlock()

	for i, l := range d.listeners {
		if l.ws == ws {
			// This creates a new slice and assigns it back.
			// It's generally safe but can be inefficient if many removes happen.
			d.listeners = append(d.listeners[:i], d.listeners[i+1:]...)
			log.Infof("[wsDispatcher] Removed client. Total listeners: %d", len(d.listeners))
			break // Assuming only one entry per connection
		}
	}
}

type wsConnection struct {
	ws *websocket.Conn
	mu sync.Mutex
}

func WsListen() {
	for {
		m := <-broadCastChannel
		dispatcher.broadCast(m)
	}
}

func WsHandler(c *gin.Context) {
	ws, err := upGrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Errorf("error upgrading connection: %s", err)
		return
	}

	defer func() {
		log.Infoln("[WsHandler] Cleaning up connection: Removing from dispatcher and closing.")
		dispatcher.rmWs(ws) // 1. Remove from our list
		ws.Close()          // 2. Ensure connection is closed
	}()

	// Create the connection object
	connection := wsConnection{ws: ws}
	// Add it to the dispatcher *after* setting up defer, so cleanup is guaranteed
	dispatcher.addWs(connection)
	log.Infof("[WsHandler] Client connected. Total listeners: %d", len(dispatcher.listeners))

	// --- Configure Handlers ---

	// Set Pong Handler: Handles pong messages from the client to keep alive.
	ws.SetPongHandler(func(appData string) error {
		log.Debug("pong received")
		// Optional: If using SetReadDeadline, it'd reset it here.
		// ws.SetReadDeadline(time.Now().Add(pongWait)) // Example
		return nil
	})

	// Set Close Handler: Primarily for logging *why* it closed, if a clean close message is received.
	// We DO NOT call rmWs or Close here anymore.
	ws.SetCloseHandler(func(code int, text string) error {
		log.Infof("[WsHandler] Close message received (code: %d, text: '%s'). Cleanup will occur via defer.", code, text)
		// Returning nil allows the default close process, which will cause ReadJSON to fail.
		return nil
	})

	// --- Start Ping Goroutine ---

	go func() {
		// Time intervals - consider making these constants or configurable
		pingPeriod := 30 * time.Second
		writeWait := 10 * time.Second

		ticker := time.NewTicker(pingPeriod)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				// Lock before writing to avoid concurrent writes
				connection.mu.Lock()
				// Set a deadline for the write operation.
				ws.SetWriteDeadline(time.Now().Add(writeWait))
				// Send a Ping message.
				err := ws.WriteMessage(websocket.PingMessage, nil)
				connection.mu.Unlock()

				if err != nil {
					log.Warnf("[WsHandler-Ping] Ping failed: %v. Closing connection.", err)
					// IMPORTANT: Only call ws.Close(). Do NOT call rmWs.
					// Closing here will cause ws.ReadJSON() in the main loop to fail,
					// which then triggers the defer block for cleanup.
					ws.Close()
					return // Exit the ping goroutine as the connection is now considered dead.
				}
				log.Debug("[WsHandler-Ping] Ping sent.")
				// How to know if the main loop has already closed?
				// The WriteMessage call will eventually fail if ws.Close() was called
				// elsewhere. There isn't a direct channel here, but this approach
				// is generally sufficient. If the main loop exits, this goroutine
				// will fail its next write and then exit.
			}
		}
	}()

	// --- Start Read Loop ---

	// This loop blocks until a message is read or an error occurs.
	for {
		msg := &SocketEvent{}
		err := ws.ReadJSON(msg) // Read incoming messages

		if err != nil {
			// Check if it's a "normal" close error or something unexpected.
			// This helps in logging. websocket.CloseNormalClosure (1000),
			// websocket.CloseGoingAway (1001) are often expected.
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure, websocket.CloseNormalClosure) {
				log.Errorf("[WsHandler-Read] Unexpected read error: %v", err)
			} else {
				// This includes normal closes, read timeouts, ws.Close() being called, etc.
				log.Infof("[WsHandler-Read] Read loop exiting (likely connection closed): %v", err)
			}
			// IMPORTANT: We simply return. The 'defer' block above handles ALL cleanup.
			return
		}

		// If a message is successfully read, process it (currently just logging).
		log.Infof("[Socket] Received: %v", msg)
	}
}
