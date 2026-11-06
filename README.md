# vinylrip

Tiny single-binary YouTube music downloader — a thin Go wrapper over `yt-dlp`.
One command, an MP3 (or M4A/OPUS) with cover art and ID3 tags embedded.

Built to be called from a music player UI (a "rip this song" button), but works
perfectly fine from a terminal too.

## Features

- **Single static binary** — no Python runtime, no venv, no pip.
- **Cover art + metadata** — embeds the video thumbnail and title/artist into the
  audio file by default (`--embed-thumbnail`, `--add-metadata`).
- **Format & quality** — MP3 (default), M4A, OPUS, with a 0–9 quality scale.
- **Persistent download dir** — set once with `vinylrip path …`, reused forever.
- **One-off dir** — `-p <folder>` for a single download without touching the config.
- **Path safety** — `~` and relative paths resolve against `$HOME`, not the CWD,
  so it works the same no matter where it's launched from.
- **Dependency check** — tells you exactly what's missing (`yt-dlp`, `ffmpeg`).

## Install

Runtime dependencies (needed by `vinylrip` itself) — openSUSE:

```bash
sudo zypper install yt-dlp ffmpeg
# optional: deno — yt-dlp 2026+ recommends a JS runtime for full format access.
# Audio-only downloads usually work without it.
sudo zypper install deno
```

Build toolchain (only needed to compile, **not** at runtime). Go is versioned
in openSUSE — install the latest, e.g.:

```bash
sudo zypper install go1.27
```

Build:

```bash
cd vinylrip
go build -o vinylrip .
# install to a directory in your PATH
sudo install -m 755 vinylrip /usr/local/bin/vinylrip
```

## Usage

```
vinylrip <url>                 Download to the configured directory
vinylrip path <folder>         Set the permanent download directory
vinylrip <url> -p <folder>     Save to <folder> this one time only
```

### Flags

| Flag | Description |
|---|---|
| `-p, --path <folder>` | One-off destination (does **not** change the config) |
| `-f, --format <fmt>` | `mp3` (default) · `m4a` · `opus` |
| `-q, --quality <0-9>` | Audio quality, `0` = best (default) |
| `--no-embed` | Skip cover art & ID3 metadata embedding |
| `-i, --info` | Print info only, no download |
| `-h, --help` | This help |

## First run

Set a download directory once:

```bash
vinylrip path ~/Music
```

Then just pass a URL:

```bash
vinylrip "https://www.youtube.com/watch?v=…"
```

## Config

Stored at `~/.config/vinyl/config.json`:

```json
{
  "download_dir": "/home/matteo/Music/"
}
```

`download_dir` accepts `~`, `~/…`, absolute paths, or paths relative to `$HOME`.

## How it works

For each URL, `vinylrip` builds a `yt-dlp` command:

```
yt-dlp -x --audio-format mp3 --audio-quality 0 \
       --embed-thumbnail --convert-thumbnail jpg --add-metadata \
       -o "<dir>/%(title)s.%(ext)s" "<url>"
```

i.e. extract audio (`-x`), pick format & quality, embed the thumbnail and tags,
and name the file after the title.

## License

MIT