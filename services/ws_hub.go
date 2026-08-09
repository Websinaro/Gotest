package services

import (
	"encoding/json"
	"log"
	"sync"
	"time"
)

// SosHub fans out live location updates for each active SOS alert to every
// connected client (the person in distress's own device, plus every
// protector who has the live map open) without anyone needing to poll the
// REST API. Rooms are keyed by sos alert ID.
//
// This matters most exactly when the network is weakest: a WebSocket is one
// long-lived TCP connection, so on a shaky 2G/3G link in a disaster area it
// avoids repeating a full HTTP request/response (headers, TLS handshake
// overhead already paid once, DNS already resolved) every few seconds the
// way REST polling does. Small frames get through where a fresh HTTP
// request might time out.
type SosHub struct {
	mu    sync.RWMutex
	rooms map[int64]map[*WSConn]wsRole
}

type wsRole struct {
	userID  int64
	isOwner bool // the person who triggered the SOS, vs a protector watching
}

var GlobalSosHub = NewSosHub()

func NewSosHub() *SosHub {
	return &SosHub{rooms: make(map[int64]map[*WSConn]wsRole)}
}

// Join registers a connection in a SOS alert's room.
func (h *SosHub) Join(sosID int64, conn *WSConn, userID int64, isOwner bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	room, ok := h.rooms[sosID]
	if !ok {
		room = make(map[*WSConn]wsRole)
		h.rooms[sosID] = room
	}
	room[conn] = wsRole{userID: userID, isOwner: isOwner}
}

// Leave removes a connection from a room, cleaning up the room entirely
// once it's empty so resolved SOS alerts don't leak memory forever.
func (h *SosHub) Leave(sosID int64, conn *WSConn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	room, ok := h.rooms[sosID]
	if !ok {
		return
	}
	delete(room, conn)
	if len(room) == 0 {
		delete(h.rooms, sosID)
	}
}

// wsLocationEvent mirrors what the Flutter SosSocketService expects. `type`
// lets one WS stream carry every event the live map screen cares about
// (location ticks, resolution) instead of needing separate connections.
type wsLocationEvent struct {
	Type      string  `json:"type"`
	SosID     int64   `json:"sos_id"`
	Latitude  float64 `json:"latitude,omitempty"`
	Longitude float64 `json:"longitude,omitempty"`
	Accuracy  float64 `json:"accuracy_m,omitempty"`
	SpeedMps  float64 `json:"speed_mps,omitempty"`
	Heading   float64 `json:"heading_deg,omitempty"`
	Timestamp string  `json:"ts,omitempty"`
}

// BroadcastLocation sends a location tick to everyone in the room except
// the connection it originated from (the sender doesn't need an echo of
// its own GPS fix).
func (h *SosHub) BroadcastLocation(sosID int64, lat, lon, accuracy, speed, heading float64, ts string, from *WSConn) {
	evt := wsLocationEvent{
		Type: "location", SosID: sosID, Latitude: lat, Longitude: lon,
		Accuracy: accuracy, SpeedMps: speed, Heading: heading, Timestamp: ts,
	}
	h.broadcast(sosID, evt, from)
}

// BroadcastResolved tells every connected client (protectors' live map
// screens, and the sender's own app if still connected) that the alert was
// marked safe, so the UI updates the instant it happens instead of waiting
// for the next poll.
func (h *SosHub) BroadcastResolved(sosID int64, resolvedTime string) {
	evt := wsLocationEvent{Type: "resolved", SosID: sosID, Timestamp: resolvedTime}
	h.broadcast(sosID, evt, nil)
	h.closeRoom(sosID)
}

func (h *SosHub) broadcast(sosID int64, evt wsLocationEvent, from *WSConn) {
	body, err := json.Marshal(evt)
	if err != nil {
		return
	}

	h.mu.RLock()
	room := h.rooms[sosID]
	conns := make([]*WSConn, 0, len(room))
	for c := range room {
		if c == from {
			continue
		}
		conns = append(conns, c)
	}
	h.mu.RUnlock()

	for _, c := range conns {
		if err := c.WriteText(body); err != nil {
			// Dead/broken connection - the read loop that owns it will
			// notice on its next read and call Leave, so just skip it here.
			continue
		}
	}
}

func (h *SosHub) closeRoom(sosID int64) {
	h.mu.Lock()
	room := h.rooms[sosID]
	delete(h.rooms, sosID)
	h.mu.Unlock()

	for c := range room {
		go c.Close()
	}
}

// StartKeepalive pings every open connection on an interval so flaky mobile
// links get pruned quickly (a write to a dead socket fails fast) instead of
// silently hanging, and so carrier-grade NATs / proxies on 2G/3G don't
// silently drop an "idle" TCP connection they think nobody wants anymore.
func (h *SosHub) StartKeepalive(interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for range ticker.C {
			h.mu.RLock()
			all := make([]*WSConn, 0)
			for _, room := range h.rooms {
				for c := range room {
					all = append(all, c)
				}
			}
			h.mu.RUnlock()

			for _, c := range all {
				if err := c.WritePing(); err != nil {
					log.Printf("[SOS WS] keepalive ping failed: %v", err)
				}
			}
		}
	}()
}
