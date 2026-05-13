#!/usr/bin/env bash
set -euo pipefail

SELF="$(realpath "$0")"
ROOT="$(dirname "$(dirname "$SELF")")"
BINARY="${ROOT}/opencola"
REFERENCE="${ROOT}/tests/fixtures/reference"

TMUX_SESSION="opencola-test-$$"
TEST_HOME="$(mktemp -d)"

PASS=0
FAIL=0

cleanup() {
    tmux kill-session -t "$TMUX_SESSION" 2>/dev/null || true
    rm -rf "$TEST_HOME"
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

assert_not_contains() {
    local name="$1"
    local needle="$2"
    local haystack="$3"

    echo -n "  TEST  $name ... "
    if echo "$haystack" | grep -Fq "$needle"; then
        echo "FAIL"
        echo "    Expected not to contain: '$needle'"
        FAIL=$((FAIL + 1))
    else
        echo "PASS"
        PASS=$((PASS + 1))
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
send "env HOME=${TEST_HOME} ${BINARY}" Enter
wait_for 1

VISIBLE=$(full_text)

assert_contains "Program renders OpenCola" "OpenCola" "$VISIBLE"
assert_contains "Banner main line" "OpenCola - minimal coding agent" "$VISIBLE"
assert_contains "Banner author line" "by Francesco Bianco" "$VISIBLE"
assert_contains "Banner help hint" "Type /help for a list of commands." "$VISIBLE"

STATUS_LINE=$(last_line_clean)
assert_eq "Status bar initial format" \
    " OpenCola v0.1.0  Provider: none  Model: none  Status: Disconnected" \
	"$STATUS_LINE"

PROMPT_LINE=$(capture_visible | tail -3 | head -1 | sed 's/[[:space:]]*$//')
BLANK_LINE=$(capture_visible | tail -2 | head -1 | sed 's/[[:space:]]*$//')
assert_eq "Initial prompt has blank line before status bar" ">" "$PROMPT_LINE"
assert_eq "Initial spacer line before status bar is empty" "" "$BLANK_LINE"

echo ""

# ============================================================================
echo "[2] Spinner Animation"
send "/spin" Enter
wait_for 0.3

# After 300ms (6 ticks) the frame should have changed from "..."
STATUS_LINE_AFTER=$(last_line_clean)
assert_contains "Spinner activated statusbar still has logo" \
    "OpenCola v0.1.0" "$STATUS_LINE_AFTER"

# The frame should be visible only after the Status value and should advance.
echo -n "  TEST  Spinner frame advanced ... "
case "$STATUS_LINE_AFTER" in
    *"Status: Disconnected "*)
        echo "PASS  (line: '$STATUS_LINE_AFTER')"
        PASS=$((PASS + 1))
        ;;
    *)
        echo "FAIL  (spinner suffix not visible)"
        FAIL=$((FAIL + 1))
        ;;
esac

sleep 0.3

# After another 300ms, frame should have advanced again
STATUS_LINE_LATER=$(last_line_clean)
echo -n "  TEST  Spinner frame keeps advancing ... "
if [ "$STATUS_LINE_AFTER" != "$STATUS_LINE_LATER" ]; then
    echo "PASS  (changed: '${STATUS_LINE_AFTER##* }' -> '${STATUS_LINE_LATER##* }')"
    PASS=$((PASS + 1))
else
    echo "FAIL  (same frame: '${STATUS_LINE_AFTER##* }')"
    FAIL=$((FAIL + 1))
fi

send "/spin" Enter
wait_for 0.2

# After turning off, frame should be back to " - "
STATUS_LINE_OFF=$(last_line_clean)
assert_eq "Status bar after spinner off" \
    " OpenCola v0.1.0  Provider: none  Model: none  Status: Disconnected" \
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

VISIBLE=$(full_text)
assert_not_contains "Clear hides startup banner" "OpenCola - minimal coding agent" "$VISIBLE"
assert_not_contains "Clear hides credits" "by Francesco Bianco" "$VISIBLE"

PROMPT_LINE=$(capture_visible | tail -3 | head -1 | sed 's/[[:space:]]*$//')
BLANK_LINE=$(capture_visible | tail -2 | head -1 | sed 's/[[:space:]]*$//')
assert_eq "Clear prompt has blank line before status bar" ">" "$PROMPT_LINE"
assert_eq "Clear spacer line before status bar is empty" "" "$BLANK_LINE"

STATUS_LINE=$(last_line_clean)
assert_eq "Status bar after clear" \
    " OpenCola v0.1.0  Provider: none  Model: none  Status: Disconnected" \
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
