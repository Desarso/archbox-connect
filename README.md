# archbox-connect

Remote desktop client for Archbox. Tunnels Moonlight/Sunshine over HTTPS.

## Connect (one command)

```
go install github.com/Desarso/archbox-connect@latest && archbox-connect
```

That's it. It will:
1. Download and set up the tunnel
2. Install Moonlight if you don't have it
3. Connect you to the remote desktop

## Requirements

- [Go](https://go.dev/dl/) installed
- Internet connection

## What it does

Creates an encrypted tunnel to the Archbox remote workstation and launches Moonlight for low-latency game-streaming-quality remote desktop access.
