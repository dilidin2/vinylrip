// vinylrip — tiny single-binary YouTube music downloader (wrapper over yt-dlp)
//
// Usage:
//   vinylrip <url>                 download to the configured directory
//   vinylrip path <folder>         set the permanent download directory
//   vinylrip <url> -p <folder>     save to <folder> this one time only
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	appName   = "vinylrip"
	cfgDir    = ".config/vinyl"
	cfgFile   = "config.json"
	defFormat = "mp3"
	defQual   = "0"
)

type Config struct {
	DownloadDir string `json:"download_dir"`
}

type opts struct {
	url      string
	onePath  string
	quality  string
	format   string
	noEmbed  bool
	info     bool
	help     bool
	havePath bool
	pathArg  string
}

func main() {
	o, err := parseArgs(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(2)
	}

	if o.help {
		printHelp()
		return
	}

	// Subcommand: set the permanent download directory.
	if o.havePath {
		if o.pathArg == "" {
			fmt.Fprintf(os.Stderr, "Usage: %s path <folder>\n", appName)
			os.Exit(2)
		}
		dir := expandUser(o.pathArg)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			fmt.Fprintf(os.Stderr, "Cannot create directory %q: %v\n", dir, err)
			os.Exit(1)
		}
		c := loadConfig()
		c.DownloadDir = dir
		if err := saveConfig(c); err != nil {
			fmt.Fprintf(os.Stderr, "Cannot save config: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Download directory set to %s\n", dir)
		return
	}

	// A URL is required.
	if o.url == "" {
		printHelp()
		os.Exit(2)
	}

	// Resolve destination directory.
	var dir string
	if o.onePath != "" {
		dir = expandUser(o.onePath)
	} else {
		c := loadConfig()
		if c.DownloadDir == "" {
			fmt.Println("Set a download directory!")
			fmt.Printf("Run: %s path <folder>\n", appName)
			os.Exit(1)
		}
		dir = expandUser(c.DownloadDir)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "Cannot create directory %q: %v\n", dir, err)
		os.Exit(1)
	}

	// Dependency check.
	var missing []string
	for _, dep := range []string{"yt-dlp", "ffmpeg"} {
		if _, err := exec.LookPath(dep); err != nil {
			missing = append(missing, dep)
		}
	}
	if len(missing) > 0 {
		fmt.Fprintf(os.Stderr, "Missing dependency: %s\n", strings.Join(missing, ", "))
		fmt.Fprintln(os.Stderr, "Install on openSUSE:")
		fmt.Fprintln(os.Stderr, "  sudo zypper install yt-dlp ffmpeg")
		os.Exit(1)
	}

	// Soft check: yt-dlp 2026+ recommends a JS runtime for full format access.
	// Audio-only downloads usually work without it, so we only warn.
	if _, err := exec.LookPath("deno"); err != nil {
		fmt.Fprintln(os.Stderr, "Note: 'deno' not found. yt-dlp 2026+ recommends a JS runtime for full format access.")
		fmt.Fprintln(os.Stderr, "  sudo zypper install deno   (optional — audio-only downloads usually work without it)")
	}

	// Build the yt-dlp command.
	format := o.format
	if format == "" {
		format = defFormat
	}
	quality := o.quality
	if quality == "" {
		quality = defQual
	}

	args := []string{
		"-x",
		"--audio-format", format,
		"--audio-quality", quality,
	}
	if !o.noEmbed {
		args = append(args, "--embed-thumbnail", "--convert-thumbnail", "jpg", "--add-metadata")
	}
	if o.info {
		args = append(args, "--no-download", "--print", "%(uploader)s | %(title)s | %(duration_string)s")
	}
	args = append(args, "-o", filepath.Join(dir, "%(title)s.%(ext)s"))
	args = append(args, o.url)

	cmd := exec.Command("yt-dlp", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Download failed: %v\n", err)
		os.Exit(1)
	}
}

func parseArgs(args []string) (opts, error) {
	var o opts
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch a {
		case "-h", "--help":
			o.help = true
		case "-p", "--path":
			if i+1 >= len(args) {
				return o, fmt.Errorf("flag %s requires an argument", a)
			}
			i++
			o.onePath = args[i]
		case "-q", "--quality":
			if i+1 >= len(args) {
				return o, fmt.Errorf("flag %s requires an argument", a)
			}
			i++
			o.quality = args[i]
		case "-f", "--format":
			if i+1 >= len(args) {
				return o, fmt.Errorf("flag %s requires an argument", a)
			}
			i++
			o.format = args[i]
		case "--no-embed":
			o.noEmbed = true
		case "-i", "--info":
			o.info = true
		default:
			if strings.HasPrefix(a, "-") && a != "-" {
				return o, fmt.Errorf("unknown flag %q (see -h)", a)
			}
			switch {
			case o.havePath:
				o.pathArg = a
			case a == "path":
				o.havePath = true
			case o.url == "":
				o.url = a
			}
		}
	}
	return o, nil
}

func configPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, cfgDir, cfgFile), nil
}

func loadConfig() Config {
	p, err := configPath()
	if err != nil {
		return Config{}
	}
	data, err := os.ReadFile(p)
	if err != nil {
		return Config{}
	}
	var c Config
	_ = json.Unmarshal(data, &c)
	return c
}

func saveConfig(c Config) error {
	p, err := configPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, data, 0o644)
}

func expandUser(p string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return p
	}
	if p == "~" {
		return home
	}
	if strings.HasPrefix(p, "~/") {
		return filepath.Join(home, p[2:])
	}
	// Un percorso relativo (senza / o ~ iniziale) è relativo a $HOME, non alla
	// CWD: così "Music/" significa sempre ~/Music/, anche se il tool viene
	// lanciato da quickshell con una working directory diversa.
	if p != "" && !strings.HasPrefix(p, "/") {
		return filepath.Join(home, p)
	}
	return p
}

func printHelp() {
	fmt.Println("vinylrip — tiny single-binary YouTube music downloader")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  vinylrip <url>                 Download to the configured directory")
	fmt.Println("  vinylrip path <folder>         Set the permanent download directory")
	fmt.Println("  vinylrip <url> -p <folder>     Save to <folder> this one time only")
	fmt.Println()
	fmt.Println("Flags:")
	fmt.Println("  -p, --path <folder>    One-off destination (does not change config)")
	fmt.Println("  -f, --format <fmt>     mp3 (default) | m4a | opus")
	fmt.Println("  -q, --quality <0-9>    Audio quality, 0 = best (default 0)")
	fmt.Println("  --no-embed             Skip cover art & ID3 metadata embedding")
	fmt.Println("  -i, --info             Print info only, no download")
	fmt.Println("  -h, --help             This help")
	fmt.Println()
	fmt.Println("First run: set a directory with:  vinylrip path ~/Music")
}
