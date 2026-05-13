#!/usr/bin/env bash
set -euo pipefail

SELF="$(realpath "$0")"
ROOT="$(dirname "$(dirname "$SELF")")"

OPENCODE_BIN="${OPENCODE_BIN:-$(which opencode)}"
OPENCOLA_BIN="${ROOT}/opencola"
PROMPTS_DIR="${ROOT}/tests/prompts"
FIXTURES_DIR="${ROOT}/tests/fixtures/demo-project"
CAPTURES_DIR="${ROOT}/tests/captures"

TMUX_WIDTH="${TMUX_WIDTH:-120}"
TMUX_HEIGHT="${TMUX_HEIGHT:-40}"
WAIT_START="${WAIT_START:-2}"
WAIT_PROMPT="${WAIT_PROMPT:-5}"
WAIT_EXIT="${WAIT_EXIT:-1}"

mkdir -p "$CAPTURES_DIR"

if [ ! -x "$OPENCODE_BIN" ]; then
    echo "ERROR: opencode not found at '$OPENCODE_BIN'"
    exit 1
fi
if [ ! -x "$OPENCOLA_BIN" ]; then
    echo "ERROR: opencola binary not found at '$OPENCOLA_BIN'"
    echo "Run 'make build' first."
    exit 1
fi

prompt_files=("$PROMPTS_DIR"/*.txt)
if [ ! -e "${prompt_files[0]}" ]; then
    echo "ERROR: no prompt files in '$PROMPTS_DIR'"
    exit 1
fi

run_prompt() {
    local binary="$1"
    local name="$2"
    local prompt_file="$3"
    local output_file="$4"
    local session="capture-$$-$RANDOM"

    local test_home
    test_home="$(mktemp -d)"

    cleanup() {
        tmux kill-session -t "$session" 2>/dev/null || true
        rm -rf "$test_home"
    }
    trap cleanup EXIT INT

    tmux new-session -d -s "$session" -x "$TMUX_WIDTH" -y "$TMUX_HEIGHT" \
        -c "$FIXTURES_DIR" \
        "env HOME=${test_home} ${binary}"
    sleep "$WAIT_START"

    local prompt_text
    prompt_text="$(cat "$prompt_file")"
    if [ -n "$prompt_text" ]; then
        tmux send-keys -t "$session" "$prompt_text" Enter
        sleep "$WAIT_PROMPT"
    fi

    tmux capture-pane -t "$session" -p > "$output_file"

    tmux send-keys -t "$session" "/exit" Enter
    sleep "$WAIT_EXIT"

    cleanup
    trap - EXIT INT
}

echo "========================================"
echo "  Prompt Comparison Tests"
echo "========================================"
echo "Fixtures: $FIXTURES_DIR"
echo "Captures: $CAPTURES_DIR"
echo ""

total=0
failed=0

for prompt_file in "${prompt_files[@]}"; do
    basename="$(basename "$prompt_file" .txt)"
    opencode_out="${CAPTURES_DIR}/${basename}-opencode.txt"
    opencola_out="${CAPTURES_DIR}/${basename}-opencola.txt"

    total=$((total + 1))
    echo "[$total] $basename"

    echo -n "  opencode ... "
    rm -f "$opencode_out"
    if run_prompt "$OPENCODE_BIN" "opencode" "$prompt_file" "$opencode_out" 2>/dev/null; then
        if [ -s "$opencode_out" ]; then
            echo "OK ($(wc -l < "$opencode_out") lines)"
        else
            echo "EMPTY"
        fi
    else
        echo "FAIL"
        failed=$((failed + 1))
    fi

    echo -n "  opencola ... "
    rm -f "$opencola_out"
    if run_prompt "$OPENCOLA_BIN" "opencola" "$prompt_file" "$opencola_out" 2>/dev/null; then
        if [ -s "$opencola_out" ]; then
            echo "OK ($(wc -l < "$opencola_out") lines)"
        else
            echo "EMPTY"
        fi
    else
        echo "FAIL"
        failed=$((failed + 1))
    fi

    echo ""
done

echo "========================================"
if [ "$failed" -gt 0 ]; then
    echo "Failed: $failed / $total"
    exit 1
else
    echo "All $total tests completed."
fi
echo "========================================"
