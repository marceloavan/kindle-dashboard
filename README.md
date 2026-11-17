# Kindle e-ink dashboard

Turn an old Kindle into a low-power, always-on dashboard. A GitHub Actions
workflow renders a high-contrast 600×800 image from live data sources and
publishes it to GitHub Pages.

Two display modes:

- **Stock**: the Kindle's browser opens the page and reloads it periodically.
- **Jailbroken** (what this project uses): a small scriptlet draws the image
  straight to the e-ink framebuffer with FBInk — fullscreen, always awake,
  timer-based refresh. See [On-device display](#on-device-display-jailbroken).

No home server required; rendering and scheduling happen in GitHub Actions.

## How it works

1. A scheduled GitHub Actions workflow (`.github/workflows/render.yml`)
   renders a monochrome PNG sized exactly for the Kindle screen.
2. The workflow deploys the PNG and a tiny HTML page to GitHub Pages.
3. The page reloads every 5 minutes via `<meta http-equiv="refresh">` — no
   JavaScript, because the Kindle's experimental browser handles JS poorly —
   and references the image with a cache-busting query string.
4. The Kindle shows it: either the browser displays the page, or (jailbroken)
   a scriptlet draws the PNG fullscreen on the framebuffer.

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
| `sources.go` | Live data sources (clock, Open-Meteo weather, Yahoo Finance quotes) |
| `index.template.html` | The page the Kindle loads (browser mode) |
| `fonts/` | Bundled DejaVu Sans (regular + bold) |
| `device/dashboard.sh` | Jailbroken-Kindle display loop (scriptlet) |
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

Sources and defaults are set via environment variables (repository variables
in Actions, or env vars when running locally):

| Variable | Default | Purpose |
| --- | --- | --- |
| `DASH_TZ` | `America/Sao_Paulo` | Display timezone (IANA name) |
| `DASH_CITY` | `Blumenau` | Weather card title |
| `DASH_LAT` / `DASH_LON` | `-26.9194` / `-49.0661` | Weather coordinates |
| `DASH_TZ_LABEL` | `Brasília` | Timezone label on the clock card |
| `DASH_TITLE` | *(empty)* | Header title (blank = timestamp only) |

Weather uses [Open-Meteo](https://open-meteo.com) and market quotes use
Yahoo Finance — both free and **no API keys**. Tickers live in
`defaultQuotes` in `sources.go` (edit to taste). To add a source, write a
function returning `Card`s in `sources.go` and append it in `build_cards()`.
Keep any API keys in GitHub Actions secrets — never in the code or the
rendered image.

## ⚠️ Public by design

GitHub Pages serves this site publicly: the rendered image is world-readable.
Don't put secrets, private calendar entries, or anything else you wouldn't
post publicly into a data source. Repo visibility can hide the *source code*,
never the image.

## On-device display (jailbroken)

The jailbroken path, validated on this device (KT3, firmware 5.16.2.1.1):

1. **Jailbreak**: [WinterBreak2](https://kindlemodding.org/jailbreaking/WinterBreak2/)
   — the wizard's supported method for the KT3 on this firmware. Browser-based,
   reversible via factory reset, and it auto-blocks OTA updates.
2. **Display**: copy `device/dashboard.sh` into the Kindle's `documents/`
   folder; it appears as a book and runs when tapped. It:
   - keeps the device awake (`preventScreenSaver`),
   - fetches the rendered PNG from GitHub Pages and draws it fullscreen with
     FBInk (bundled at `/mnt/us/libkh/bin/fbink`),
   - refreshes every 5 minutes (every 4th refresh flashes to clear ghosting),
   - exits on power-button press, screen tap, or a `/mnt/us/dashboard.stop`
     flag file, logging to `/mnt/us/dashboard.log`.
3. The image URL lives in `BASE_URL` at the top of the script; point it at
   your own Pages deployment.

## Compatibility notes

- The device must stay on firmware **5.16.2.1.1**: it's the last version
  supported by the current jailbreaks, and the OTA blocker is what keeps the
  window open. Do not update.
- Validated on-device: WinterBreak2 jailbreak, Wi-Fi fetch of the Pages image
  over TLS, FBInk fullscreen draw (600×800), power-button exit.
- Still to observe long-term: Wi-Fi reconnection after drops, behavior across
  deep sleep/reboots, and e-ink ghosting over multi-hour runs.
