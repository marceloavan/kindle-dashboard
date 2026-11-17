#!/bin/sh
# Kindle dashboard — framebuffer display loop.
#
# Start: tap this scriptlet in the Library.
# Exit:  press the POWER button, or touch the screen (after the first 3s),
#        or create /mnt/us/dashboard.stop over USB as a last resort.
# Log:   /mnt/us/dashboard.log
#
# Fetches the rendered PNG from GitHub Pages and draws it fullscreen with
# fbink. Refreshes every INTERVAL seconds; every 4th refresh uses a flashing
# GC16 waveform to clear ghosting.

LOG=/mnt/us/dashboard.log
FBINK=/mnt/us/libkh/bin/fbink
BASE_URL="https://marceloavan.github.io/kindle-dashboard/dashboard.png"
PIDFILE=/mnt/us/dashboard.pid
STOPFLAG=/mnt/us/dashboard.stop
INTERVAL=300

exec > "$LOG" 2>&1
echo "=== dashboard start $(date) ==="

# --- single instance guard ---
if [ -f "$PIDFILE" ] && kill -0 "$(cat "$PIDFILE" 2>/dev/null)" 2>/dev/null; then
    echo "already running (pid $(cat "$PIDFILE")); exit"
    exit 0
fi
echo $$ > "$PIDFILE"
rm -f "$STOPFLAG"

# --- keep the device awake ---
/usr/bin/lipc-set-prop com.lab126.powerd preventScreenSaver 1
echo "preventScreenSaver -> $(/usr/bin/lipc-get-prop com.lab126.powerd preventScreenSaver)"

cleanup() {
    echo "cleanup"
    /usr/bin/lipc-set-prop com.lab126.powerd preventScreenSaver 0
    rm -f "$PIDFILE"
    $FBINK -q -c
    echo "=== dashboard exit $(date) ==="
}
trap cleanup EXIT

# --- helpers ---
png_magic_ok() {
    [ "$(head -c 4 "$1" | od -An -tx1 | tr -d ' \n')" = "89504e47" ]
}

draw() { # $1 = waveform
    $FBINK -q -i /tmp/dash.png -W "$1"
    echo "draw exit=$? w=$1 size=$(wc -c < /tmp/dash.png)"
}

fetch_draw() { # $1 = waveform
    URL="$BASE_URL?v=$(date +%s)"
    rm -f /tmp/dash.new
    if ! curl -s --max-time 30 -o /tmp/dash.new "$URL"; then
        echo "fetch failed"
        return 1
    fi
    if ! png_magic_ok /tmp/dash.new; then
        echo "bad png ($(wc -c < /tmp/dash.new) bytes)"
        return 1
    fi
    mv /tmp/dash.new /tmp/dash.png
    draw "$1"
    return 0
}

# --- exit watchers (background) ---
( /usr/bin/lipc-wait-event -s 0 com.lab126.powerd powerButton
  touch "$STOPFLAG" ) &
POWERW=$!

( sleep 3
  while [ ! -f "$STOPFLAG" ]; do
      if dd if=/dev/input/event0 bs=16 count=1 2>/dev/null | od -An -tx1 | grep -qE '[0-9a-f]'; then
          touch "$STOPFLAG"
          break
      fi
  done ) &
TOUCHW=$!

# --- first draw (retry while network comes up) ---
n=0
until fetch_draw gl16; do
    n=$((n + 1))
    [ "$n" -ge 5 ] && break
    sleep 10
done

# --- main refresh loop ---
i=0
while [ ! -f "$STOPFLAG" ]; do
    sleep "$INTERVAL"
    [ -f "$STOPFLAG" ] && break
    i=$((i + 1))
    W=gl16
    [ $((i % 4)) -eq 0 ] && W=gc16
    fetch_draw "$W"
done

echo "stop requested"
kill "$POWERW" "$TOUCHW" 2>/dev/null
exit 0
