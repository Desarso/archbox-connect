# archbox-connect

One-command remote desktop client for Archbox: it opens an authenticated HTTPS tunnel
(chisel) to a Sunshine streaming host and launches Moonlight against `localhost`.

Works on **Windows**, **macOS**, and **Linux**.

## Connect (one command)

```
go install github.com/Desarso/archbox-connect@latest && archbox-connect
```

On first run you'll be prompted for the password. It's saved locally to `~/.archbox/credentials` so you only need to enter it once.

If `archbox-connect` is not found, add Go's bin to your PATH first:

```bash
# Linux/macOS
export PATH=$PATH:$(go env GOPATH)/bin

# Windows (PowerShell)
$env:PATH += ";$(go env GOPATH)\bin"
```

Then run `archbox-connect` again.

## Why

Moonlight/Sunshine expect to reach the host directly over a set of TCP and UDP ports.
The Archbox host sits on a private network, so instead of port-forwarding or a VPN,
this client tunnels exactly those ports through a single HTTPS endpoint with chisel.
From Moonlight's point of view, the desktop is running on `localhost`.

## What happens

1. Reads the password from `~/.archbox/credentials`, or prompts for it (hidden input) and saves it (`0600`).
2. Downloads the chisel client for your OS/arch into `~/.archbox/bin/` if it isn't there yet
   (re-downloads if the existing binary looks truncated or quarantined).
3. Checks for Moonlight. On Windows it fetches the latest installer from the Moonlight GitHub
   releases and launches it; on macOS/Linux it prints the install command.
4. Starts `chisel client` against the Archbox server, forwarding the Sunshine ports to localhost.
5. Launches Moonlight (unless it is already running). Connect to **localhost**.
6. If the tunnel drops, it reconnects automatically after 3 seconds. Ctrl+C disconnects.

```
Moonlight ──► localhost:47984-48010 ──► chisel client ══ HTTPS ══► chisel server ──► Sunshine host
                                        (this tool)                (Archbox)
```

Forwarded ports: TCP 47984, 47989, 47990, 47998, 48010 and UDP 47998, 47999, 48000, 48010.

## Windows notes

Chisel is a tunneling tool and Windows Defender commonly flags it as a false positive.
On Windows, before downloading chisel, archbox-connect asks for elevation (a normal UAC
prompt via `ShellExecuteExW` with `runas`) to add `~/.archbox` to Defender's exclusion list,
and extracts chisel in-process from the release `.zip`. If you decline the prompt it prints
the `Add-MpPreference` command to run yourself.

## Build from source

```bash
go build -o archbox-connect .
./archbox-connect
```

The chisel version, server URL, tunnel user and target host are constants at the top of
`main.go`; change them there to point the client at a different chisel/Sunshine setup.

## Stack

- Go (stdlib + `golang.org/x/term` for hidden password input, `golang.org/x/sys`)
- [chisel](https://github.com/jpillora/chisel) for the HTTP/WebSocket tunnel
- [Moonlight](https://moonlight-stream.org/) / [Sunshine](https://github.com/LizardByte/Sunshine) for streaming

Related: [evdev-bridge](https://github.com/Desarso/evdev-bridge) forwards Sunshine's virtual
keyboard and mouse into a Wayland compositor on the host side.

## Requirements

- [Go](https://go.dev/dl/) installed
- [Moonlight](https://moonlight-stream.org/) (auto-installed on Windows, manual on Mac/Linux)
- The Archbox password (this client is configured for one specific server)

## Status

Personal tool, built for a single Archbox host. Releases are tagged (`v0.2.x`).
There are no automated tests.
