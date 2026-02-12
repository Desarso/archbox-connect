package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"
)

const (
	chiselVersion   = "1.10.1"
	moonlightWinURL = "https://github.com/moonlight-stream/moonlight-qt/releases/latest/download/MoonlightSetup-x64.exe"
)

type Config struct {
	Server   string `json:"server"`
	Username string `json:"username"`
	Password string `json:"password"`
	Target   string `json:"target"` // archbox container IP
}

func configDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".archbox")
}

func configPath() string {
	return filepath.Join(configDir(), "config.json")
}

func chiselPath() string {
	name := "chisel"
	if runtime.GOOS == "windows" {
		name = "chisel.exe"
	}
	return filepath.Join(configDir(), "bin", name)
}

func loadConfig() (*Config, error) {
	data, err := os.ReadFile(configPath())
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func saveConfig(cfg *Config) error {
	os.MkdirAll(configDir(), 0700)
	data, _ := json.MarshalIndent(cfg, "", "  ")
	return os.WriteFile(configPath(), data, 0600)
}

func prompt(label string, defaultVal string) string {
	reader := bufio.NewReader(os.Stdin)
	if defaultVal != "" {
		fmt.Printf("%s [%s]: ", label, defaultVal)
	} else {
		fmt.Printf("%s: ", label)
	}
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)
	if input == "" {
		return defaultVal
	}
	return input
}

func promptPassword(label string) string {
	fmt.Printf("%s: ", label)
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	return strings.TrimSpace(input)
}

func setupConfig() *Config {
	fmt.Println("=== Archbox Connect - First Time Setup ===")
	fmt.Println()

	cfg := &Config{}
	cfg.Server = prompt("Chisel server URL", "https://stream.gabrielmalek.com")
	cfg.Target = prompt("Archbox container IP", "10.10.10.102")
	cfg.Username = prompt("Username", "archbox")
	cfg.Password = promptPassword("Password")

	if cfg.Password == "" {
		fmt.Println("Error: password is required")
		os.Exit(1)
	}

	if err := saveConfig(cfg); err != nil {
		fmt.Printf("Error saving config: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("\nConfig saved to %s\n\n", configPath())
	return cfg
}

func chiselDownloadURL() string {
	var os_name, ext string
	switch runtime.GOOS {
	case "windows":
		os_name = "windows"
		ext = "gz"
	case "darwin":
		os_name = "darwin"
		ext = "gz"
	default:
		os_name = "linux"
		ext = "gz"
	}

	var arch string
	switch runtime.GOARCH {
	case "arm64":
		arch = "arm64"
	default:
		arch = "amd64"
	}

	return fmt.Sprintf(
		"https://github.com/jpillora/chisel/releases/download/v%s/chisel_%s_%s_%s.%s",
		chiselVersion, chiselVersion, os_name, arch, ext,
	)
}

func downloadFile(url, dest string) error {
	fmt.Printf("Downloading %s...\n", filepath.Base(dest))
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

func installChisel() error {
	if _, err := os.Stat(chiselPath()); err == nil {
		return nil // already installed
	}

	url := chiselDownloadURL()
	gzPath := chiselPath() + ".gz"

	if err := downloadFile(url, gzPath); err != nil {
		return fmt.Errorf("download chisel: %w", err)
	}

	// Decompress gzip
	fmt.Println("Extracting chisel...")
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		// Use PowerShell to decompress on Windows
		ps := fmt.Sprintf(
			`$i=[System.IO.File]::OpenRead('%s');`+
				`$g=New-Object System.IO.Compression.GZipStream($i,[System.IO.Compression.CompressionMode]::Decompress);`+
				`$o=[System.IO.File]::Create('%s');`+
				`$g.CopyTo($o);$o.Close();$g.Close();$i.Close()`,
			gzPath, chiselPath())
		cmd = exec.Command("powershell", "-Command", ps)
	} else {
		cmd = exec.Command("sh", "-c", fmt.Sprintf("gunzip -f -k '%s' && mv '%s' '%s'",
			gzPath, strings.TrimSuffix(gzPath, ".gz"), chiselPath()))
		// Actually gunzip removes the .gz suffix automatically
		cmd = exec.Command("sh", "-c", fmt.Sprintf("gzip -d -f '%s'", gzPath))
	}

	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("decompress: %s: %w", string(out), err)
	}

	os.Remove(gzPath)
	os.Chmod(chiselPath(), 0755)
	fmt.Println("Chisel installed.")
	return nil
}

func checkMoonlight() bool {
	switch runtime.GOOS {
	case "windows":
		// Check common install paths
		paths := []string{
			filepath.Join(os.Getenv("ProgramFiles"), "Moonlight Game Streaming", "Moonlight.exe"),
			filepath.Join(os.Getenv("LOCALAPPDATA"), "Moonlight Game Streaming", "Moonlight.exe"),
		}
		for _, p := range paths {
			if _, err := os.Stat(p); err == nil {
				return true
			}
		}
		// Check PATH
		if _, err := exec.LookPath("moonlight"); err == nil {
			return true
		}
		return false
	case "darwin":
		if _, err := os.Stat("/Applications/Moonlight.app"); err == nil {
			return true
		}
		return false
	default: // Linux
		if _, err := exec.LookPath("moonlight"); err == nil {
			return true
		}
		return false
	}
}

func installMoonlight() {
	if checkMoonlight() {
		return
	}

	fmt.Println()
	fmt.Println("Moonlight is not installed.")

	switch runtime.GOOS {
	case "windows":
		fmt.Println("Downloading Moonlight installer...")
		installer := filepath.Join(os.TempDir(), "MoonlightSetup.exe")
		if err := downloadFile(moonlightWinURL, installer); err != nil {
			fmt.Printf("Failed to download Moonlight: %v\n", err)
			fmt.Println("Please install manually from: https://moonlight-stream.org")
			return
		}
		fmt.Println("Launching Moonlight installer...")
		cmd := exec.Command(installer)
		cmd.Start()
		fmt.Println("Please complete the Moonlight installation, then restart this tool.")
		os.Exit(0)
	case "darwin":
		fmt.Println("Please install Moonlight from: https://moonlight-stream.org")
		fmt.Println("Or: brew install --cask moonlight")
		os.Exit(1)
	default:
		fmt.Println("Please install Moonlight:")
		fmt.Println("  Arch: sudo pacman -S moonlight-qt")
		fmt.Println("  Ubuntu/Debian: sudo apt install moonlight-qt")
		fmt.Println("  Flatpak: flatpak install flathub com.moonlight_stream.Moonlight")
		os.Exit(1)
	}
}

func findMoonlight() string {
	switch runtime.GOOS {
	case "windows":
		paths := []string{
			filepath.Join(os.Getenv("ProgramFiles"), "Moonlight Game Streaming", "Moonlight.exe"),
			filepath.Join(os.Getenv("LOCALAPPDATA"), "Moonlight Game Streaming", "Moonlight.exe"),
		}
		for _, p := range paths {
			if _, err := os.Stat(p); err == nil {
				return p
			}
		}
		if p, err := exec.LookPath("moonlight"); err == nil {
			return p
		}
	case "darwin":
		return "open"
	default:
		if p, err := exec.LookPath("moonlight"); err == nil {
			return p
		}
	}
	return ""
}

func sunshinePort(base string, target string, proto string) string {
	if proto != "" {
		return fmt.Sprintf("%s:%s:%s/%s", base, target, base, proto)
	}
	return fmt.Sprintf("%s:%s:%s", base, target, base)
}

func main() {
	fmt.Println("╔═══════════════════════════════════════╗")
	fmt.Println("║       Archbox Connect                 ║")
	fmt.Println("║   Remote Desktop via Moonlight        ║")
	fmt.Println("╚═══════════════════════════════════════╝")
	fmt.Println()

	// Handle --setup flag to reconfigure
	for _, arg := range os.Args[1:] {
		if arg == "--setup" || arg == "-s" {
			setupConfig()
			fmt.Println("Setup complete. Run again without --setup to connect.")
			return
		}
		if arg == "--help" || arg == "-h" {
			fmt.Println("Usage: archbox-connect [options]")
			fmt.Println()
			fmt.Println("Options:")
			fmt.Println("  --setup, -s    Reconfigure server/credentials")
			fmt.Println("  --no-moonlight Don't auto-launch Moonlight")
			fmt.Println("  --help, -h     Show this help")
			return
		}
	}

	noMoonlight := false
	for _, arg := range os.Args[1:] {
		if arg == "--no-moonlight" {
			noMoonlight = true
		}
	}

	// Load or create config
	cfg, err := loadConfig()
	if err != nil {
		cfg = setupConfig()
	}

	// Install chisel if needed
	if err := installChisel(); err != nil {
		fmt.Printf("Error installing chisel: %v\n", err)
		os.Exit(1)
	}

	// Check/install Moonlight
	if !noMoonlight {
		installMoonlight()
	}

	// Build chisel tunnel arguments
	target := cfg.Target
	tunnels := []string{
		fmt.Sprintf("47984:%s:47984", target),     // HTTPS control
		fmt.Sprintf("47989:%s:47989", target),     // HTTP
		fmt.Sprintf("47990:%s:47990", target),     // Web UI
		fmt.Sprintf("47998:%s:47998", target),     // TCP
		fmt.Sprintf("47998:%s:47998/udp", target), // UDP
		fmt.Sprintf("47999:%s:47999/udp", target), // Video
		fmt.Sprintf("48000:%s:48000/udp", target), // Audio
		fmt.Sprintf("48010:%s:48010", target),     // TCP
		fmt.Sprintf("48010:%s:48010/udp", target), // UDP
	}

	args := []string{
		"client",
		"--auth", fmt.Sprintf("%s:%s", cfg.Username, cfg.Password),
		cfg.Server,
	}
	args = append(args, tunnels...)

	fmt.Printf("Connecting to %s...\n", cfg.Server)
	fmt.Println()

	// Start chisel
	chiselCmd := exec.Command(chiselPath(), args...)
	chiselCmd.Stdout = os.Stdout
	chiselCmd.Stderr = os.Stderr

	if err := chiselCmd.Start(); err != nil {
		fmt.Printf("Error starting tunnel: %v\n", err)
		os.Exit(1)
	}

	// Wait a moment for tunnel to establish
	time.Sleep(3 * time.Second)

	// Check if chisel is still running
	if chiselCmd.ProcessState != nil && chiselCmd.ProcessState.Exited() {
		fmt.Println("Error: tunnel failed to start. Check your credentials.")
		os.Exit(1)
	}

	fmt.Println()
	fmt.Println("════════════════════════════════════════")
	fmt.Println("  Tunnel is UP! Connect Moonlight to:")
	fmt.Println()
	fmt.Println("         localhost")
	fmt.Println()
	fmt.Println("  Press Ctrl+C to disconnect")
	fmt.Println("════════════════════════════════════════")
	fmt.Println()

	// Launch Moonlight if available
	if !noMoonlight {
		moonlightPath := findMoonlight()
		if moonlightPath != "" {
			fmt.Println("Launching Moonlight...")
			var mlCmd *exec.Cmd
			if runtime.GOOS == "darwin" {
				mlCmd = exec.Command("open", "-a", "Moonlight")
			} else {
				mlCmd = exec.Command(moonlightPath)
			}
			mlCmd.Start()
		}
	}

	// Wait for interrupt
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// Also wait for chisel to exit
	chiselDone := make(chan error, 1)
	go func() {
		chiselDone <- chiselCmd.Wait()
	}()

	select {
	case <-sigCh:
		fmt.Println("\nDisconnecting...")
		chiselCmd.Process.Kill()
	case err := <-chiselDone:
		if err != nil {
			fmt.Printf("\nTunnel disconnected: %v\n", err)
		} else {
			fmt.Println("\nTunnel disconnected.")
		}
	}
}
