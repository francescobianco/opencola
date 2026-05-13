#!/usr/bin/env bash
set -euo pipefail

SELF="$(realpath "$0")"
ROOT="$(dirname "$(dirname "$SELF")")"
BINARY="${ROOT}/opencola"
REFERENCE="${ROOT}/tests/fixtures/reference"

TMUX_SESSION="opencola-test-$$"

PASS=0
FAIL=0

cleanup() {
    tmux kill-session -t "$TMUX_SESSION" 2>/dev/null || true
}
trap cleanup EXIT
trap cleanup INT

capture_visible() {
    tmux capture-pane -t "$TMUX_SESSION" -p
}

last_line_clean() {
    capture_visible | tail -1 | sed 's/[[:space:]]*$//'
}

full_text() {
    capture_visible | sed 's/[[:space:]]*$//'
}

start_tmux() {
    tmux new-session -d -s "$TMUX_SESSION" -x 80 -y 24
    sleep 0.3
}

send() {
    tmux send-keys -t "$TMUX_SESSION" "$@"
}

wait_for() {
    sleep "$1"
}

assert_contains() {
    local name="$1"
    local needle="$2"
    local haystack="$3"

    echo -n "  TEST  $name ... "
    if echo "$haystack" | grep -Fq "$needle"; then
        echo "PASS"
        PASS=$((PASS + 1))
    else
        echo "FAIL"
        echo "    Expected to contain: '$needle'"
        echo "    $(echo "$haystack" | head -c 120)"
        FAIL=$((FAIL + 1))
    fi
}

assert_eq() {
    local name="$1"
    local expected="$2"
    local actual="$3"

    echo -n "  TEST  $name ... "
    if [ "$expected" = "$actual" ]; then
        echo "PASS"
        PASS=$((PASS + 1))
    else
        echo "FAIL"
        echo "    Expected: '$expected'"
        echo "    Actual:   '$actual'"
        FAIL=$((FAIL + 1))
    fi
}

###############################################################################
echo ""
echo "TUI Layout & Spinner Tests"
echo "=========================="
echo ""

cd "$ROOT"

# ============================================================================
echo "[1] Initial Layout"
start_tmux
send "${BINARY}" Enter
wait_for 1

VISIBLE=$(full_text)

assert_contains "Program renders OpenCola" "OpenCola" "$VISIBLE"
assert_contains "Banner main line" "OpenCola - minimal coding agent" "$VISIBLE"
assert_contains "Banner author line" "by Francesco Bianco" "$VISIBLE"
assert_contains "Banner help hint" "Type /help for a list of commands." "$VISIBLE"

STATUS_LINE=$(last_line_clean)
assert_eq "Status bar initial format" \
    "... OpenCola v0.1.0  Provider: none  Model: none  Status: Disconnected" \
	"$STATUS_LINE"

echo ""

# ============================================================================
echo "[2] Spinner Animation"
send "/spin" Enter
wait_for 0.3

# After 300ms (6 ticks) the frame should have changed from "..."
STATUS_LINE_AFTER=$(last_line_clean)
assert_contains "Spinner activated statusbar still has logo" \
    "OpenCola v0.1.0" "$STATUS_LINE_AFTER"

# The frame should NOT be "..." anymore (it should have advanced)
echo -n "  TEST  Spinner frame advanced ... "
case "$STATUS_LINE_AFTER" in
    "... "*)
        echo "FAIL  (frame did not change)"
        FAIL=$((FAIL + 1))
        ;;
    *)
        echo "PASS  (frame: '${STATUS_LINE_AFTER:0:3}')"
        PASS=$((PASS + 1))
        ;;
esac

sleep 0.3

# After another 300ms, frame should have advanced again
STATUS_LINE_LATER=$(last_line_clean)
echo -n "  TEST  Spinner frame keeps advancing ... "
if [ "$STATUS_LINE_AFTER" != "$STATUS_LINE_LATER" ]; then
    echo "PASS  (changed: '${STATUS_LINE_AFTER:0:3}' -> '${STATUS_LINE_LATER:0:3}')"
    PASS=$((PASS + 1))
else
    echo "FAIL  (same frame: '${STATUS_LINE_AFTER:0:3}')"
    FAIL=$((FAIL + 1))
fi

send "/spin" Enter
wait_for 0.2

# After turning off, frame should be back to " - "
STATUS_LINE_OFF=$(last_line_clean)
assert_eq "Status bar after spinner off" \
    "... OpenCola v0.1.0  Provider: none  Model: none  Status: Disconnected" \
    "$STATUS_LINE_OFF"

echo ""

# ============================================================================
echo "[3] Commands"
send "/help" Enter
wait_for 0.3

VISIBLE=$(full_text)
assert_contains "/help shows Available commands" "Available commands:" "$VISIBLE"
assert_contains "/help shows /connect" "/connect" "$VISIBLE"
assert_contains "/help shows /models" "/models" "$VISIBLE"

send "clear" Enter
wait_for 0.3

STATUS_LINE=$(last_line_clean)
assert_eq "Status bar after clear" \
    "... OpenCola v0.1.0  Provider: none  Model: none  Status: Disconnected" \
    "$STATUS_LINE"

echo ""

# ============================================================================
echo "[4] Exit"
send "/exit" Enter
wait_for 0.3

VISIBLE=$(full_text)
assert_contains "Goodbye message on exit" \
    "Goodbye! Thanks for using OpenCola. See you next time!" \
    "$VISIBLE"

echo ""

# ============================================================================
echo "=========================="
echo "Results: $PASS passed, $FAIL failed"
echo "=========================="

if [ $FAIL -gt 0 ]; then
    exit 1
fi
