# Kindle e-ink dashboard

Turn an old Kindle into a low-power, always-on dashboard. A GitHub Actions
workflow renders a high-contrast 600×800 image from your data sources and
publishes it to GitHub Pages; the Kindle's built-in browser opens the page
and reloads it periodically.

No home server, no device modifications, nothing to install on the Kindle.

## How it works

1. A scheduled GitHub Actions workflow (`.github/workflows/render.yml`)
   renders a monochrome PNG sized exactly for the Kindle screen.
2. The workflow deploys the PNG and a tiny HTML page to GitHub Pages.
3. The page reloads every 5 minutes via `<meta http-equiv="refresh">` — no
   JavaScript, because the Kindle's experimental browser handles JS poorly —
   and references the image with a cache-busting query string.
4. The Kindle displays the page in its browser.

Rendering and scheduling happen off-device, so the Kindle only needs Wi-Fi.

## Target device

- Kindle (8th generation, 2016), model `SY69JL` (KT3)
- Firmware **5.16.2.1.1** — see [Compatibility notes](#compatibility-notes); do not update
- 6" e-ink, 600×800 @ 167 ppi, 16-level grayscale

Other Kindles with a browser and a ~600×800 screen should work by adjusting
`width`/`height` in `render.go`.

## Rendering

- Draws at 2× resolution, then downsamples for smooth anti-aliased edges.
- Default output is **1-bit black/white** — best contrast on e-ink — via a
  hand-rolled Floyd–Steinberg dither; `--grayscale` keeps the 16-level gray
  the display supports.
- The DejaVu Sans fonts and the HTML template are embedded with `go:embed`,
  so output is identical locally and in CI.

## Repository layout

| Path | Purpose |
| --- | --- |
| `main.go` | CLI entrypoint; writes `site/dashboard.png` + `site/index.html` |
| `render.go` | Layout, drawing, dithering |
| `sources.go` | Data sources (sample data by default) |
| `index.template.html` | The page the Kindle loads |
| `fonts/` | Bundled DejaVu Sans (regular + bold) |
| `.github/workflows/render.yml` | Scheduled render + Pages deployment |

## Quick start

Requires Go 1.26+.

```sh
go run . --out site              # 1-bit black/white (default)
go run . --out site --grayscale  # 16-level grayscale
```

Open `site/dashboard.png` (or `site/index.html` in a browser) to inspect the
result.

## Deployment

1. Enable Pages: **Settings → Pages → Build and deployment → Source → GitHub
   Actions**.
2. Push to the default branch — the workflow runs every 15 minutes and on
   manual dispatch.
3. On the Kindle, open `https://<user>.github.io/<repo>/` and bookmark it.

Notes: scheduled workflows only run from the default branch, and GitHub
pauses them after ~60 days of repository inactivity.

## Configuration

- **Timezone** — set `DASH_TZ` (IANA name, e.g. `Europe/Lisbon`; default UTC)
  as a repository variable in Actions, or as an env var when running locally.
- **Data sources** — add a function returning `Card`s in `sources.go` and
  append it in `build_cards()`. The current sources are sample/placeholder.
- **Secrets** — keep API keys in GitHub Actions secrets. Never in the code or
  the rendered image.

## ⚠️ Public by design

GitHub Pages serves this site publicly: the rendered image is world-readable.
Don't put secrets, private calendar entries, or anything else you wouldn't
post publicly into a data source. Repo visibility can hide the *source code*,
never the image.

## Compatibility notes

- The device must stay on firmware **5.16.2.1.1**: it's the last version
  supported by the LanguageBreak jailbreak, which this project keeps as a
  fallback option. An OTA update would close that window.
- Browser auto-refresh, Wi-Fi reconnection, and sleep/screensaver behavior on
  the Kindle 8th gen are not yet validated. Stock firmware can't disable
  sleep, so if the browser doesn't keep the page alive, a jailbreak-based
  display setup is the likely next step.
