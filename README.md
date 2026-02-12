# archbox-connect

Remote desktop client for Archbox. Tunnels Moonlight/Sunshine over HTTPS.

Works on **Windows**, **macOS**, and **Linux**.

## Connect (one command)

```
go install github.com/Desarso/archbox-connect@latest && archbox-connect
```

If `archbox-connect` is not found, add Go's bin to your PATH first:

```bash
# Linux/macOS
export PATH=$PATH:$(go env GOPATH)/bin

# Windows (PowerShell)
$env:PATH += ";$(go env GOPATH)\bin"
```

Then run `archbox-connect` again.

## What happens

1. Downloads the tunnel client (chisel) if you don't have it
2. Installs Moonlight if you don't have it
3. Opens an encrypted tunnel to the remote desktop
4. Launches Moonlight — connect to **localhost**

## Requirements

- [Go](https://go.dev/dl/) installed
- [Moonlight](https://moonlight-stream.org/) (auto-installed on Windows, manual on Mac/Linux)
