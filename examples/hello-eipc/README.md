# Hello EIPC

A minimal working example of an EIPC server and client.

## Prerequisites

- Go 1.22+

## Run

Open two terminals:

### Terminal 1 — Server

```bash
cd examples/hello-eipc
go run ./server/
```

Output:
```
EIPC server listening on 127.0.0.1:9090
```

### Terminal 2 — Client

```bash
cd examples/hello-eipc
go run ./client/
```

Output:
```
sent message to server
response: {"status":"ok","message":"Hello from EIPC!"}
```

Server output:
```
received: type=chat source=hello-client payload={"text":"Hello, EIPC!"}
```

## What's Happening

1. **Server** listens on `127.0.0.1:9090` using TCP transport
2. **Client** connects and sends a `TypeChat` message with a JSON payload
3. Both sides use HMAC-SHA256 to sign and verify every frame
4. **Server** receives the message, prints it, and sends an `TypeAck` response
5. **Client** receives and prints the response

## Key Concepts Demonstrated

- `tcp.New()` — create a TCP transport
- `core.NewServerEndpoint()` / `core.NewClientEndpoint()` — create endpoints
- `ep.Send()` / `ep.Receive()` — send and receive messages
- HMAC integrity is automatic (both endpoints sign with the shared secret)
- Messages use the canonical `core.Message` envelope

## Next Steps

- Add authentication: See the full server in `cmd/eipc-server/`
- Try Unix sockets: Replace `tcp.New()` with `unix.New()` on Linux/macOS
- Add capability checks: Use `security/capability` package
- Enable TLS: Set `EIPC_TLS_AUTO_CERT=true` environment variable
