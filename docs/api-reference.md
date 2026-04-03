# EIPC API Reference

Auto-generated style documentation for all public Go packages.

> Generate full godoc with: `godoc -http=:6060` then visit `http://localhost:6060/pkg/github.com/embeddedos-org/eipc/`

---

## Package `core`

`import "github.com/embeddedos-org/eipc/core"`

The core package defines the canonical message envelope, endpoint API, event types, and priority-aware router.

### Types

#### Message

```go
type Message struct {
    Version    uint16      `json:"version"`
    Type       MessageType `json:"type"`
    Source     string      `json:"source"`
    Timestamp  time.Time   `json:"timestamp"`
    SessionID  string      `json:"session_id"`
    RequestID  string      `json:"request_id"`
    Priority   Priority    `json:"priority"`
    Capability string      `json:"capability"`
    Payload    []byte      `json:"payload"`
}
```

The canonical EIPC message envelope exchanged between ENI and EAI.

#### MessageType

```go
type MessageType string

const (
    TypeIntent       MessageType = "intent"        // ENI→EAI: neural intent
    TypeFeatures     MessageType = "features"      // ENI→EAI: feature stream
    TypeToolRequest  MessageType = "tool_request"  // EAI→Tool: tool invocation
    TypeAck          MessageType = "ack"           // Bidirectional: acknowledgement
    TypePolicyResult MessageType = "policy_result" // EAI→ENI: auth decision
    TypeHeartbeat    MessageType = "heartbeat"     // Bidirectional: liveness
    TypeAudit        MessageType = "audit"         // Internal: audit record
    TypeChat         MessageType = "chat"          // ebot→EAI: chat message
    TypeComplete     MessageType = "complete"      // ebot→EAI: completion request
)
```

#### Priority

```go
type Priority uint8

const (
    PriorityP0 Priority = 0  // Control-critical
    PriorityP1 Priority = 1  // Interactive
    PriorityP2 Priority = 2  // Telemetry
    PriorityP3 Priority = 3  // Debug / audit bulk
)
```

### Functions

#### NewMessage

```go
func NewMessage(msgType MessageType, source string, payload []byte) Message
```

Creates a message with defaults: `Version=1`, `Timestamp=now`, `Priority=P1`.

#### MsgTypeToByte

```go
func MsgTypeToByte(mt MessageType) uint8
```

Converts a MessageType to its single-byte wire representation.

### Endpoint Interface

```go
type Endpoint interface {
    Send(msg Message) error
    Receive() (Message, error)
    Close() error
}
```

#### ClientEndpoint

```go
func NewClientEndpoint(conn transport.Connection, codec protocol.Codec, hmacKey []byte, sessionID string) *ClientEndpoint
```

Creates a client endpoint. The endpoint automatically signs all outgoing messages with HMAC-SHA256 and verifies incoming message MACs.

| Method | Description |
|--------|-------------|
| `Send(msg Message) error` | Marshal header, build frame, sign HMAC, transmit |
| `Receive() (Message, error)` | Read frame, verify HMAC, unmarshal header, return Message |
| `Close() error` | Close underlying connection |

#### ServerEndpoint

```go
func NewServerEndpoint(conn transport.Connection, codec protocol.Codec, hmacKey []byte) *ServerEndpoint
```

Creates a server endpoint with replay detection (default window: 128).

| Method | Description |
|--------|-------------|
| `Send(msg Message) error` | Marshal header, build frame, sign HMAC, transmit |
| `Receive() (Message, error)` | Read frame, verify HMAC, check replay, unmarshal, return Message |
| `Close() error` | Close underlying connection |
| `RemoteAddr() string` | Remote peer address |
| `SetPeerCapabilities(caps []string)` | Set authenticated peer's capability list |
| `ValidateCapability(msgCap string) error` | Check if peer has the required capability |

### Router

```go
func NewRouter() *Router
```

Creates a new priority-aware message router.

| Method | Description |
|--------|-------------|
| `Handle(msgType MessageType, handler HandlerFunc)` | Register handler for message type |
| `Dispatch(msg Message) (*Message, error)` | Route single message to handler |
| `DispatchBatch(messages []Message) []DispatchResult` | Process batch in priority order (P0 first) |

```go
type HandlerFunc func(msg Message) (*Message, error)

type DispatchResult struct {
    Response *Message
    Err      error
}
```

### Event Types

| Type | Fields | Usage |
|------|--------|-------|
| `IntentEvent` | Intent, Confidence, SessionID, Features | Neural intent from ENI |
| `FeatureStreamEvent` | Features map[string]float64 | Real-time feature vectors |
| `ToolRequestEvent` | Tool, Args, Permission, AuditID | Tool invocation request |
| `AckEvent` | RequestID, Status, Error | Acknowledgement |
| `PolicyResultEvent` | RequestID, Allowed, Reason | Authorization decision |
| `HeartbeatEvent` | Service, Status | Liveness probe |
| `AuditEvent` | RequestID, Actor, Action, Target, Decision, Result | Audit record |
| `ChatRequestEvent` | SessionID, UserPrompt, Model, MaxTokens | Chat prompt |
| `ChatResponseEvent` | SessionID, Response, Model, TokensUsed | Chat response |
| `CompleteRequestEvent` | SessionID, Prompt, Model, MaxTokens | Completion prompt |
| `CompleteResponseEvent` | SessionID, Completion, Model, TokensUsed | Completion response |

### Errors

```go
var (
    ErrAuth         = errors.New("eipc: authentication failed")
    ErrCapability   = errors.New("eipc: capability check failed")
    ErrIntegrity    = errors.New("eipc: integrity verification failed")
    ErrReplay       = errors.New("eipc: replay detected")
    ErrTimeout      = errors.New("eipc: operation timed out")
    ErrBackpressure = errors.New("eipc: backpressure limit reached")
)
```

---

## Package `protocol`

`import "github.com/embeddedos-org/eipc/protocol"`

The protocol package defines the binary wire format, frame encoding/decoding, header structure, and codec interface.

### Frame

```go
type Frame struct {
    Version uint16
    MsgType uint8
    Flags   uint8
    Header  []byte
    Payload []byte
    MAC     []byte
}
```

| Method | Description |
|--------|-------------|
| `Encode(w io.Writer) error` | Write frame in wire format |
| `SignableBytes() []byte` | Returns bytes covered by MAC (preamble + header + payload) |

```go
func Decode(r io.Reader) (*Frame, error)
```

Parse a frame from a reader. Returns `ErrBadMagic`, `ErrBadVersion`, or `ErrFrameTooLarge` on invalid input.

### Constants

| Constant | Value | Description |
|----------|-------|-------------|
| `MagicBytes` | `0x45495043` | ASCII "EIPC" |
| `MaxFrameSize` | `1 << 20` (1 MB) | Maximum frame size |
| `MACSize` | `32` | HMAC-SHA256 output size |
| `ProtocolVersion` | `1` | Current wire version |
| `FlagHMAC` | `1 << 0` | Frame carries HMAC |
| `FlagCompress` | `1 << 1` | Payload compressed (reserved) |
| `FlagEncrypted` | `1 << 2` | Payload encrypted with AES-GCM |

### Header

```go
type Header struct {
    ServiceID     string `json:"service_id"`
    SessionID     string `json:"session_id"`
    RequestID     string `json:"request_id"`
    Sequence      uint64 `json:"sequence"`
    Timestamp     string `json:"timestamp"`
    Priority      uint8  `json:"priority"`
    Capability    string `json:"capability,omitempty"`
    Route         string `json:"route,omitempty"`
    PayloadFormat uint8  `json:"payload_format"`
}
```

### Codec

```go
type Codec interface {
    Marshal(v interface{}) ([]byte, error)
    Unmarshal(data []byte, v interface{}) error
}
```

| Implementation | Description |
|----------------|-------------|
| `JSONCodec` | Default codec using `encoding/json` |
| `DefaultCodec()` | Returns `JSONCodec{}` |

---

## Package `transport`

`import "github.com/embeddedos-org/eipc/transport"`

The transport package defines pluggable transport interfaces and the `ConnWrapper` for length-prefixed framing.

### Transport Interface

```go
type Transport interface {
    Listen(address string) error
    Dial(address string) (Connection, error)
    Accept() (Connection, error)
    Close() error
}
```

### Connection Interface

```go
type Connection interface {
    Send(frame *protocol.Frame) error
    Receive() (*protocol.Frame, error)
    RemoteAddr() string
    Close() error
}
```

### ConnWrapper

```go
func NewConnWrapper(conn net.Conn) *ConnWrapper
```

Wraps a `net.Conn` with 4-byte big-endian length-prefixed frame I/O.

### Transport Implementations

| Package | Platform | Address Format |
|---------|----------|----------------|
| `transport/tcp` | All | `"host:port"` |
| `transport/unix` | Linux, macOS | `"/path/to/socket"` |
| `transport/windows` | Windows | `"host:port"` (TCP fallback) |
| `transport/shm` | All (in-process) | Ring buffer config |

---

## Package `security/auth`

`import "github.com/embeddedos-org/eipc/security/auth"`

Challenge-response authentication with session management.

```go
func NewAuthenticator(secret []byte, serviceCapabilities map[string][]string) *Authenticator
```

| Method | Description |
|--------|-------------|
| `Authenticate(serviceID string) (*PeerIdentity, error)` | Direct auth (trusted context) |
| `CreateChallenge(serviceID string) (*Challenge, error)` | Generate 32-byte nonce for service |
| `VerifyResponse(serviceID string, response []byte) (*PeerIdentity, error)` | Verify HMAC response |
| `ValidateSession(token string) (*PeerIdentity, error)` | Check session token validity |
| `RevokeSession(token string)` | Invalidate session |
| `SetSessionTTL(ttl time.Duration)` | Configure session lifetime |
| `CleanupExpired() int` | Remove expired sessions, return count |

---

## Package `security/capability`

`import "github.com/embeddedos-org/eipc/security/capability"`

Capability-based authorization with runtime grant/revoke.

```go
func NewChecker(rules map[string][]string) *Checker
```

| Method | Description |
|--------|-------------|
| `Check(capabilities []string, action string) error` | Verify action is permitted |
| `Grant(capability, action string)` | Add action to capability at runtime |
| `Revoke(capability, action string)` | Remove action from capability |

---

## Package `security/integrity`

`import "github.com/embeddedos-org/eipc/security/integrity"`

HMAC-SHA256 message signing and verification.

```go
func Sign(key, data []byte) []byte
func Verify(key, data, mac []byte) bool
```

---

## Package `security/replay`

`import "github.com/embeddedos-org/eipc/security/replay"`

Sliding-window replay detection.

```go
func NewTracker(windowSize int) *Tracker  // 0 = default 128
```

| Method | Description |
|--------|-------------|
| `Check(seq uint64) error` | Returns `ErrReplay` if duplicate/old |
| `Reset()` | Clear all state |

---

## Package `security/keyring`

`import "github.com/embeddedos-org/eipc/security/keyring"`

Key lifecycle management: generation, storage, rotation, revocation.

```go
func New() *Keyring
```

| Method | Description |
|--------|-------------|
| `Generate(id string, size int, ttl time.Duration) (*Entry, error)` | Create new key |
| `Store(id string, key []byte, ttl time.Duration) error` | Store existing key |
| `Lookup(id string) (*Entry, error)` | Retrieve key by ID |
| `Rotate(id string, size int, ttl time.Duration) (*Entry, error)` | Replace key |
| `Revoke(id string)` | Mark key as revoked |
| `Delete(id string)` | Permanently remove key |
| `ListActive() []Entry` | All non-expired, non-revoked keys |
| `Cleanup() int` | Remove expired keys |

---

## Package `security/encryption`

`import "github.com/embeddedos-org/eipc/security/encryption"`

AES-256-GCM authenticated encryption for payload confidentiality.

```go
func Encrypt(key, plaintext []byte) ([]byte, error)
func Decrypt(key, ciphertext []byte) ([]byte, error)
```

- Key must be 32 bytes (AES-256)
- Output format: `[nonce:12][ciphertext][tag:16]`
- Uses `crypto/aes` + `crypto/cipher` from Go stdlib

---

## Package `services/broker`

`import "github.com/embeddedos-org/eipc/services/broker"`

Pub/sub message broker with priority-ordered delivery.

```go
func NewBroker(reg *registry.Registry, auditLogger audit.Logger) *Broker
```

| Method | Description |
|--------|-------------|
| `Subscribe(sub *Subscriber) error` | Register subscriber endpoint |
| `Unsubscribe(serviceID string)` | Remove subscriber |
| `AddRoute(msgType MessageType, targets ...string)` | Map message type to targets |
| `RemoveRoute(msgType MessageType, target string)` | Remove route mapping |
| `Route(msg Message) []RouteResult` | Deliver to routed subscribers (priority order) |
| `Fanout(msg Message) []RouteResult` | Deliver to all subscribers |
| `Subscribers() []string` | List subscriber IDs |

---

## Package `services/registry`

`import "github.com/embeddedos-org/eipc/services/registry"`

In-memory service registry with capability discovery.

```go
func NewRegistry() *Registry
```

| Method | Description |
|--------|-------------|
| `Register(info ServiceInfo) error` | Add/update service |
| `Deregister(serviceID string)` | Remove service |
| `Lookup(serviceID string) (*ServiceInfo, error)` | Find by ID |
| `List() []ServiceInfo` | All registered services |
| `FindByCapability(cap string) []ServiceInfo` | Find by capability |

---

## Package `services/policy`

`import "github.com/embeddedos-org/eipc/services/policy"`

Three-tier policy engine for action classification.

```go
func NewEngine(defaultDeny bool, auditLogger audit.Logger) *Engine
```

| Action Tier | Verdict | Description |
|-------------|---------|-------------|
| `ActionSafe` | `VerdictAllow` | Always permitted |
| `ActionControlled` | Capability check | Requires matching capability |
| `ActionRestricted` | `VerdictConfirm` | Requires explicit confirmation |

---

## Package `services/audit`

`import "github.com/embeddedos-org/eipc/services/audit"`

JSON-line audit logging with pluggable output.

```go
func NewFileLogger(path string) (*FileLogger, error)  // "" = stdout
```

| Method | Description |
|--------|-------------|
| `Log(entry Entry) error` | Write JSON-line audit record |
| `Close() error` | Close output writer |

---

## Package `services/health`

`import "github.com/embeddedos-org/eipc/services/health"`

Heartbeat tracking with configurable timeout.

```go
func NewService(interval, timeout time.Duration) *Service
```

| Method | Description |
|--------|-------------|
| `RecordHeartbeat(serviceID, status string)` | Update peer liveness |
| `IsAlive(serviceID string) bool` | Check if within timeout |
| `AllPeers() []PeerStatus` | All known peers |
| `LivePeers() []PeerStatus` | Only alive peers |
| `Interval() time.Duration` | Configured interval |

---

## Package `config`

`import "github.com/embeddedos-org/eipc/config"`

Environment variable configuration loader.

| Function | Env Var | Default | Description |
|----------|---------|---------|-------------|
| `LoadHMACKey()` | `EIPC_HMAC_KEY` or `EIPC_KEY_FILE` | (required) | Shared HMAC key |
| `LoadSessionTTL()` | `EIPC_SESSION_TTL` | `1h` | Session lifetime |
| `LoadMaxConnections()` | `EIPC_MAX_CONNECTIONS` | `64` | Max concurrent connections |
| `LoadListenAddr()` | `EIPC_LISTEN_ADDR` | `127.0.0.1:9090` | Server listen address |
| `TLSEnabled()` | `EIPC_TLS_CERT` or `EIPC_TLS_AUTO_CERT` | `false` | TLS mode check |
