package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"
	"time"
)

const (
	chiselVersion = "1.10.1"
	server        = "https://stream.gabrielmalek.com"
	authUser      = "archbox"
	authPass      = "REDACTED"
	target        = "10.10.10.102"

	moonlightWinURL = "https://github.com/moonlight-stream/moonlight-qt/releases/latest/download/MoonlightSetup-x64.exe"
)

func binDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".archbox", "bin")
}

func chiselPath() string {
	name := "chisel"
	if runtime.GOOS == "windows" {
		name = "chisel.exe"
	}
	return filepath.Join(binDir(), name)
}

func downloadFile(url, dest string) error {
	fmt.Printf("  Downloading %s...\n", filepath.Base(dest))
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	os.MkdirAll(filepath.Dir(dest), 0755)
	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, resp.Body)
	return err
}

func chiselDownloadURL() string {
	osName := runtime.GOOS
	arch := runtime.GOARCH
	if arch == "" {
		arch = "amd64"
	}
	return fmt.Sprintf(
		"https://github.com/jpillora/chisel/releases/download/v%s/chisel_%s_%s_%s.gz",
		chiselVersion, chiselVersion, osName, arch,
	)
}

func installChisel() error {
	if _, err := os.Stat(chiselPath()); err == nil {
		return nil
	}
	fmt.Println("[1/2] Installing chisel tunnel...")
	gzPath := chiselPath() + ".gz"
	if err := downloadFile(chiselDownloadURL(), gzPath); err != nil {
		return fmt.Errorf("download: %w", err)
	}
	fmt.Println("  Extracting...")
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		ps := fmt.Sprintf(
			`$i=[System.IO.File]::OpenRead('%s');`+
				`$g=New-Object System.IO.Compression.GZipStream($i,[System.IO.Compression.CompressionMode]::Decompress);`+
				`$o=[System.IO.File]::Create('%s');`+
				`$g.CopyTo($o);$o.Close();$g.Close();$i.Close()`,
			gzPath, chiselPath())
		cmd = exec.Command("powershell", "-Command", ps)
	} else {
		cmd = exec.Command("gzip", "-d", "-f", gzPath)
	}
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("extract: %s: %w", string(out), err)
	}
	os.Remove(gzPath)
	os.Chmod(chiselPath(), 0755)
	fmt.Println("  Done.")
	return nil
}

func findMoonlight() string {
	switch runtime.GOOS {
	case "windows":
		for _, p := range []string{
			filepath.Join(os.Getenv("ProgramFiles"), "Moonlight Game Streaming", "Moonlight.exe"),
			filepath.Join(os.Getenv("LOCALAPPDATA"), "Moonlight Game Streaming", "Moonlight.exe"),
		} {
			if _, err := os.Stat(p); err == nil {
				return p
			}
		}
		if p, err := exec.LookPath("moonlight"); err == nil {
			return p
		}
	case "darwin":
		if _, err := os.Stat("/Applications/Moonlight.app"); err == nil {
			return "open"
		}
	default:
		if p, err := exec.LookPath("moonlight"); err == nil {
			return p
		}
	}
	return ""
}

func installMoonlight() bool {
	if findMoonlight() != "" {
		return true
	}
	fmt.Println("[2/2] Moonlight not found. Installing...")
	switch runtime.GOOS {
	case "windows":
		installer := filepath.Join(os.TempDir(), "MoonlightSetup.exe")
		if err := downloadFile(moonlightWinURL, installer); err != nil {
			fmt.Printf("  Failed: %v\n", err)
			fmt.Println("  Install manually: https://moonlight-stream.org")
			return false
		}
		fmt.Println("  Launching installer... complete the setup then restart this tool.")
		exec.Command(installer).Start()
		return false
	case "darwin":
		fmt.Println("  Install: brew install --cask moonlight")
		fmt.Println("  Or download from: https://moonlight-stream.org")
		return false
	default:
		fmt.Println("  Arch:   sudo pacman -S moonlight-qt")
		fmt.Println("  Ubuntu: sudo apt install moonlight-qt")
		fmt.Println("  Or:     flatpak install flathub com.moonlight_stream.Moonlight")
		return false
	}
}

func main() {
	fmt.Println()
	fmt.Println("  ╔═════════════════════════════════════╗")
	fmt.Println("  ║        archbox-connect               ║")
	fmt.Println("  ║   Remote Desktop via Moonlight       ║")
	fmt.Println("  ╚═════════════════════════════════════╝")
	fmt.Println()

	// Install chisel
	if err := installChisel(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	// Install/check Moonlight
	hasMoonlight := installMoonlight()

	// Build tunnel args
	tunnels := []string{
		fmt.Sprintf("47984:%s:47984", target),
		fmt.Sprintf("47989:%s:47989", target),
		fmt.Sprintf("47990:%s:47990", target),
		fmt.Sprintf("47998:%s:47998", target),
		fmt.Sprintf("47998:%s:47998/udp", target),
		fmt.Sprintf("47999:%s:47999/udp", target),
		fmt.Sprintf("48000:%s:48000/udp", target),
		fmt.Sprintf("48010:%s:48010", target),
		fmt.Sprintf("48010:%s:48010/udp", target),
	}

	args := []string{"client", "--auth", fmt.Sprintf("%s:%s", authUser, authPass), server}
	args = append(args, tunnels...)

	fmt.Printf("  Connecting to %s...\n\n", server)

	chiselCmd := exec.Command(chiselPath(), args...)
	chiselCmd.Stdout = os.Stdout
	chiselCmd.Stderr = os.Stderr
	if err := chiselCmd.Start(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	time.Sleep(3 * time.Second)

	fmt.Println()
	fmt.Println("  ════════════════════════════════════════")
	fmt.Println("    Tunnel is UP!")
	fmt.Println()
	fmt.Println("    Open Moonlight and connect to:")
	fmt.Println("               localhost")
	fmt.Println()
	fmt.Println("    Press Ctrl+C to disconnect")
	fmt.Println("  ════════════════════════════════════════")
	fmt.Println()

	// Launch Moonlight
	if hasMoonlight {
		ml := findMoonlight()
		if ml != "" {
			fmt.Println("  Launching Moonlight...")
			if runtime.GOOS == "darwin" {
				exec.Command("open", "-a", "Moonlight").Start()
			} else {
				exec.Command(ml).Start()
			}
		}
	}

	// Wait for signal or chisel exit
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	done := make(chan error, 1)
	go func() { done <- chiselCmd.Wait() }()

	select {
	case <-sigCh:
		fmt.Println("\n  Disconnecting...")
		chiselCmd.Process.Kill()
	case err := <-done:
		if err != nil {
			fmt.Printf("\n  Tunnel lost: %v\n", err)
		}
	}
}
