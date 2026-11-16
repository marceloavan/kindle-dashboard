# Kindle e-ink dashboard — project notes

## Goal
Reuse a Kindle 8th generation (2016) as a personalized, read-only e-ink dashboard with multiple data sources and periodic refreshes, while keeping the device physically intact. There is no dedicated home server.

Confirmed device details:
- **Model**: Kindle (8th Generation, 2016) — model number `SY69JL`, serial prefixes `G000K9` / `G000KA`, community short-name **KT3**
- **Firmware**: **5.16.2.1.1**
- **Display**: 6" e-ink, 600×800 @ 167 ppi, 16-level grayscale
- **Storage**: 4 GB (~3 GB available)

Serial number prefix still to be confirmed from the device itself.

## Proposed architecture
- A GitHub Actions workflow periodically fetches/configures the selected data sources and renders a high-contrast monochrome PNG sized for the Kindle screen.
- GitHub Pages serves the generated image.
- The Kindle opens the hosted image in its browser; refresh behavior and persistence still need practical validation.

This keeps rendering and scheduling off the Kindle and avoids running a home server. Treat the hosted page/image as public unless repository and Pages settings establish otherwise; do not include secrets or private data in the image or published artifacts.

## Compatibility caveats
- **Firmware 5.16.2.1.1 is the last LanguageBreak-compatible version** (LanguageBreak works on any Kindle ≤5.16.2.1.1; 5.16.3+ is patched). **Do not update the firmware**, or the jailbreak window closes permanently.
- Jailbreak compatibility still depends on the exact model/serial prefix; confirm the prefix before attempting anything.
- The Kindle Dashboard reference was tested on Paperwhite; support on the basic Kindle 8th generation (KT3) is unconfirmed.
- Browser behavior, periodic refresh, Wi-Fi reconnection, and sleep/screensaver behavior on this device are unknown and must be tested.

## Prototype (image generator + hosted page)
A jailbreak-agnostic first pass that can be built and tested before any
device-level decision.

Layout:
- `main.go` — CLI entrypoint (flags, PNG + HTML output; `go:embed`s the HTML
  template).
- `render.go` — draws a 600×800 image at 2x, downsamples, then (by default)
  dithers to 1-bit black/white with Floyd–Steinberg. Pass `--grayscale` for
  16-level gray. Fonts are `go:embed`ded, so the binary is self-contained.
- `sources.go` — data sources (`Card`/`Row` types). Currently sample/placeholder
  data; real sources plug in later. Display timezone via `DASH_TZ` (default UTC).
- `index.template.html` — page with a `<meta http-equiv="refresh">` (no JS,
  for the weak browser) and a cache-busted image URL.
- `fonts/` — bundled DejaVu Sans (regular + bold) so rendering is identical
  locally and in CI.
- `.github/workflows/render.yml` — renders and publishes on a 15-minute
  schedule (and on manual dispatch).

Run locally:

    go run . --out site              # 1-bit black/white (default)
    go run . --out site --grayscale  # 16-level grayscale

Deploy (one-time repo setup):
1. In repo **Settings → Pages → Build and deployment → Source**, choose
   **GitHub Actions**.
2. Push to `main`. The workflow runs on schedule and on `workflow_dispatch`,
   deploying `site/` to `https://<user>.github.io/<repo>/`.
3. On the Kindle, open that URL in the browser and bookmark it.

Still to validate on-device (step 4): whether the experimental browser honors
the meta refresh, how it caches the page/image, and Wi-Fi/sleep behavior.

## Staged plan
1. Confirm the serial number prefix from the device (expect `G000K9` / `G000KA`) to finalize the compatibility check; firmware is already confirmed at 5.16.2.1.1.
2. Research current, model-specific compatibility and recovery instructions; make a backup before any modification.
3. Prototype the image generator and hosted page without modifying the Kindle; begin with non-sensitive sample data. *(scaffolded — see "Prototype" above.)*
4. Test the page in the Kindle browser, then observe refresh, sleep, and reconnection behavior.
5. Only if needed and verified compatible, evaluate a reversible jailbreak-based display/refresh setup, following current device-specific instructions.
6. Add real data sources and a schedule, keeping credentials out of published files and generated images.

## Safety notes
- Back up first and keep a recovery path available.
- Use airplane mode while researching or preparing modifications if appropriate for the device state.
- Do not update firmware or jailbreak until the exact model is known and the compatibility and instructions have been checked against current sources.
- Watch for automatic over-the-air updates: the dashboard needs Wi-Fi, but staying online risks an OTA update to 5.16.3+ that closes the jailbreak window. Consider blocking update domains or disconnecting Wi-Fi when not actively testing.
- Avoid irreversible hardware changes; the requirement is to keep the Kindle intact.

## Next action
Confirm the serial number prefix on the back label or in **Settings → Device Options → Device Info** (expect `G000K9` / `G000KA`). Once the prefix is known, the LanguageBreak compatibility question is effectively settled for this firmware — and the device must stay on 5.16.2.1.1.
