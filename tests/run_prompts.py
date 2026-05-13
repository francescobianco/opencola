#!/usr/bin/env python3
"""Capture TUI output using pexpect with a virtual terminal screen buffer."""

import os, sys, time, tempfile, shutil, warnings
warnings.filterwarnings('ignore')
try:
    import pexpect
except ImportError:
    print("ERROR: pexpect is required", file=sys.stderr)
    sys.exit(1)


class VirtualTerminal:

    def __init__(self, rows=40, cols=120):
        self.rows = rows
        self.cols = cols
        self.reset()

    def reset(self):
        self.screen = [[' '] * self.cols for _ in range(self.rows)]
        self.cr = 0
        self.cc = 0

    def put(self, ch):
        if 32 <= ord(ch) < 127 or ord(ch) >= 160:
            if self.cc < self.cols:
                self.screen[self.cr][self.cc] = ch
                self.cc += 1
                if self.cc >= self.cols:
                    self.cc = 0
                    self.cr += 1
                    self._scroll()
        elif ch == '\n':
            self.cr += 1
            self._scroll()
        elif ch == '\r':
            self.cc = 0

    def _scroll(self):
        while self.cr >= self.rows:
            self.screen.pop(0)
            self.screen.append([' '] * self.cols)
            self.cr -= 1

    def handle_csi(self, params, cmd):
        pts = []
        for p in params.split(';'):
            try:
                pts.append(int(p) if p else 0)
            except ValueError:
                pts.append(0)
        if not pts:
            pts = [0]

        if cmd == 'A':
            self.cr = max(self.cr - (pts[0] or 1), 0)
        elif cmd == 'B':
            self.cr = min(self.cr + (pts[0] or 1), self.rows - 1)
        elif cmd == 'C':
            self.cc = min(self.cc + (pts[0] or 1), self.cols - 1)
        elif cmd == 'D':
            self.cc = max(self.cc - (pts[0] or 1), 0)
        elif cmd in ('H', 'f'):
            self.cr = max(min((pts[0] or 1) - 1, self.rows - 1), 0)
            self.cc = max(min((pts[1] or 1) - 1, self.cols - 1), 0)
        elif cmd == 'J':
            n = pts[0] if pts[0] > 0 else 0
            if n == 2:
                self.screen = [[' '] * self.cols for _ in range(self.rows)]
            elif n == 1:
                for r in range(self.cr + 1):
                    self.screen[r] = [' '] * self.cols
            else:
                for r in range(self.cr, self.rows):
                    for c in range(self.cols):
                        self.screen[r][c] = ' '
        elif cmd == 'K':
            n = pts[0]
            if n == 2:
                self.screen[self.cr] = [' '] * self.cols
            elif n == 1:
                for c in range(self.cc + 1):
                    self.screen[self.cr][c] = ' '
            else:
                for c in range(self.cc, self.cols):
                    self.screen[self.cr][c] = ' '

    def dump(self):
        return '\n'.join(''.join(row) for row in self.screen)


def parse_and_dump(raw_log_path, output_path, rows, cols):
    term = VirtualTerminal(rows, cols)
    csi_buf = ''
    in_csi = False
    in_osc = False
    osc_buf = ''

    with open(raw_log_path, 'rb') as f:
        data = f.read()

    for byte in data:
        ch = chr(byte) if byte < 128 else bytes([byte]).decode('utf-8', errors='replace')

        if in_osc:
            if byte == 0x07:
                in_osc = False
            elif byte == 0x1b:
                in_osc = False
            continue

        if in_csi:
            if ch in 'ABCDEFGHJKSTfmlh':
                term.handle_csi(csi_buf, ch)
                in_csi = False
                csi_buf = ''
            elif ch in '0123456789;':
                csi_buf += ch
            else:
                in_csi = False
                csi_buf = ''
            continue

        if byte == 0x1b:
            in_csi = False
            csi_buf = ''
            continue

        if byte == 0x9b:
            in_csi = True
            csi_buf = ''
            continue

        if ch == '\x1b':
            in_csi = True
            csi_buf = ''
            continue

        if byte == 0x07:
            continue

        term.put(ch)

    with open(output_path, 'w') as f:
        f.write(term.dump())


def run_prompt(binary_path, prompt_text, output_path, fixtures_dir,
               rows=40, cols=120, timeout=15):
    test_home = None
    log_path = output_path + '.raw.log'

    try:
        test_home = tempfile.mkdtemp(prefix='tui-capture-')
        env = os.environ.copy()
        env['HOME'] = test_home

        child = pexpect.spawn(
            binary_path,
            timeout=timeout,
            encoding='utf-8',
            codec_errors='replace',
            dimensions=(rows, cols),
            env=env,
            cwd=fixtures_dir
        )

        with open(log_path, 'w', encoding='utf-8', errors='replace') as raw_log:
            child.logfile_read = raw_log

            time.sleep(2)
            if prompt_text:
                child.sendline(prompt_text)
                time.sleep(5)

            child.sendline('/exit')
            try:
                child.expect(pexpect.EOF, timeout=5)
            except pexpect.TIMEOUT:
                child.close(force=True)
            except pexpect.EOF:
                pass

        child.close(force=True)
        parse_and_dump(log_path, output_path, rows, cols)
        return True

    except Exception as e:
        print(f"  ERROR: {e}", file=sys.stderr)
        return False

    finally:
        if test_home:
            try:
                shutil.rmtree(test_home)
            except Exception:
                pass


def main():
    ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
    OPENCODE_BIN = os.environ.get('OPENCODE_BIN',
        '/home/francesco/.opencode/bin/opencode')
    OPENCOLA_BIN = os.path.join(ROOT, 'opencola')
    PROMPTS_DIR = os.path.join(ROOT, 'tests', 'prompts')
    FIXTURES_DIR = os.path.join(ROOT, 'tests', 'fixtures', 'demo-project')
    CAPTURES_DIR = os.path.join(ROOT, 'tests', 'captures')

    os.makedirs(CAPTURES_DIR, exist_ok=True)

    for name, path in [('opencode', OPENCODE_BIN), ('opencola', OPENCOLA_BIN)]:
        if not os.access(path, os.X_OK):
            print(f"ERROR: {name} not found at {path}")
            sys.exit(1)

    prompt_files = sorted(f for f in os.listdir(PROMPTS_DIR) if f.endswith('.txt'))
    if not prompt_files:
        print("ERROR: no prompt files in", PROMPTS_DIR)
        sys.exit(1)

    print("=" * 50)
    print("  Prompt Comparison Tests (pexpect)")
    print("=" * 50)
    print(f"  Fixtures: {FIXTURES_DIR}")
    print(f"  Captures: {CAPTURES_DIR}")
    print()

    total = 0
    failed = 0

    for prompt_file in prompt_files:
        prompt_path = os.path.join(PROMPTS_DIR, prompt_file)
        with open(prompt_path, 'r') as f:
            prompt_text = f.read().strip()

        basename = os.path.splitext(prompt_file)[0]
        total += 1
        print(f"[{total}] {basename}")

        for bin_name, bin_path in [('opencode', OPENCODE_BIN),
                                    ('opencola', OPENCOLA_BIN)]:
            out_file = os.path.join(CAPTURES_DIR, f"{basename}-{bin_name}.txt")
            print(f"  {bin_name} ... ", end='', flush=True)
            if run_prompt(bin_path, prompt_text, out_file, FIXTURES_DIR):
                with open(out_file, 'r') as f:
                    lines = [l for l in f.read().split('\n') if l.strip()]
                print(f"OK ({len(lines)} non-blank lines), saved: {os.path.basename(out_file)}")
            else:
                print("FAIL")
                failed += 1

        print()

    print("=" * 50)
    if failed:
        print(f"  Failed: {failed}/{total}")
        sys.exit(1)
    else:
        print(f"  All {total} tests completed.")
    print("=" * 50)


if __name__ == '__main__':
    main()
