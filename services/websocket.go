package services

// Thin wrapper around github.com/coder/websocket (the actively maintained
// successor to nhooyr.io/websocket), giving the rest of the codebase the
// same small WSConn-shaped API (UpgradeWebSocket / ReadMessage / WriteText /
// WritePing / Close) that ws_hub.go and routes/sos_ws.go were already
// written against.
//
// This replaces the earlier hand-rolled RFC 6455 implementation that this
// file used to contain. That version existed only because the environment
// this backend was originally generated in had no network access to `go
// get` a WebSocket module. With coder/websocket now vendored in go.mod, we
// get context.Context-aware reads/writes, a spec-correct close handshake,
// concurrent-write safety, and Autobahn-tested framing for free instead of
// maintaining our own frame parser.
//
// coder/websocket has zero dependencies of its own, so this doesn't drag in
// anything beyond the one module (see go.mod / go.sum - run `go mod tidy`
// once to fetch it and let Go verify the checksums already pinned there).

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/coder/websocket"
)

// Message type constants. Kept as the same exported names/values callers
// (ws_hub.go, routes/sos_ws.go) already switch on, so this swap is a
// drop-in replacement - only this file changed.
const (
	OpText   = 0x1
	OpBinary = 0x2
)

var errConnectionClosed = errors.New("websocket: connection closed")

// pingTimeout bounds how long a keepalive ping is allowed to block, so a
// half-dead connection on a flaky mobile link gets reaped instead of
// hanging the keepalive goroutine forever.
const pingTimeout = 10 * time.Second

// WSConn is a single upgraded WebSocket connection. Safe for one reader
// goroutine and one writer goroutine to use concurrently - coder/websocket's
// Conn already supports concurrent writes internally, matching how
// ws_hub.go fans out broadcasts while routes/sos_ws.go's read loop runs on
// its own goroutine per connection.
type WSConn struct {
	c *websocket.Conn
}

// UpgradeWebSocket performs the HTTP -> WebSocket handshake and returns a
// connection ready for ReadMessage/WriteText. OriginPatterns is left
// wide-open ("*") to mirror this backend's existing CORS_ALLOWED_ORIGINS=*
// default (see middleware/cors_middleware.go) - the mobile app, not a
// browser, is the primary client here, and auth still happens via the
// `token` query param checked before this is ever called.
func UpgradeWebSocket(w http.ResponseWriter, r *http.Request) (*WSConn, error) {
	c, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		OriginPatterns: []string{"*"},
	})
	if err != nil {
		return nil, err
	}
	return &WSConn{c: c}, nil
}

// ReadMessage blocks until the next data frame (text or binary) arrives.
// coder/websocket already answers ping frames with pong and enforces the
// close handshake internally, so this just needs to translate its
// MessageType into the OpText/OpBinary constants callers already expect,
// and normalize any close/error into errConnectionClosed so callers can
// keep treating "any error" as "stop reading and clean up this connection".
func (c *WSConn) ReadMessage() (opcode int, payload []byte, err error) {
	typ, data, err := c.c.Read(context.Background())
	if err != nil {
		return 0, nil, errConnectionClosed
	}
	switch typ {
	case websocket.MessageBinary:
		return OpBinary, data, nil
	default:
		return OpText, data, nil
	}
}

// WriteText sends a single text frame.
func (c *WSConn) WriteText(data []byte) error {
	return c.c.Write(context.Background(), websocket.MessageText, data)
}

// WritePing sends a ping frame, used by the hub as a keepalive so idle
// connections on flaky mobile networks (2G/3G) get pruned quickly instead
// of lingering as half-open sockets. Bounded by pingTimeout so one dead
// socket can't stall the keepalive loop for every other connection.
func (c *WSConn) WritePing() error {
	ctx, cancel := context.WithTimeout(context.Background(), pingTimeout)
	defer cancel()
	return c.c.Ping(ctx)
}

// Close performs the WebSocket close handshake (best-effort) and closes the
// underlying TCP connection. Safe to call multiple times - coder/websocket
// already makes repeat Close calls a no-op after the first.
func (c *WSConn) Close() error {
	return c.c.Close(websocket.StatusNormalClosure, "")
}
