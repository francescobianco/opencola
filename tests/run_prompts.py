#!/usr/bin/env python3
"""Capture prompt output from opencode and opencola using tmux."""

import os
import re
import shlex
import shutil
import subprocess
import sys
import tempfile
import time
import uuid


def env_int(name, default):
    value = os.environ.get(name)
    if value is None:
        return default
    try:
        return int(value)
    except ValueError:
        print(f"ERROR: {name} must be an integer, got {value!r}", file=sys.stderr)
        sys.exit(1)


def env_float(name, default):
    value = os.environ.get(name)
    if value is None:
        return default
    try:
        return float(value)
    except ValueError:
        print(f"ERROR: {name} must be a number, got {value!r}", file=sys.stderr)
        sys.exit(1)


def truthy_env(name):
    return os.environ.get(name, '').lower() in ('1', 'true', 'yes', 'on')


def safe_name(value):
    value = re.sub(r'[^A-Za-z0-9_.-]+', '-', value).strip('-')
    return value or 'prompt'


def run_tmux(*args, check=True):
    proc = subprocess.run(
        ['tmux', *args],
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        text=True,
    )
    if check and proc.returncode != 0:
        cmd = ' '.join(shlex.quote(part) for part in ('tmux', *args))
        raise RuntimeError(f"{cmd} failed: {proc.stderr.strip()}")
    return proc


def wait_for_tmux_session_exit(session, timeout):
    deadline = time.time() + timeout
    while time.time() < deadline:
        if run_tmux('has-session', '-t', session, check=False).returncode != 0:
            return
        time.sleep(0.1)


def capture_pane(session):
    return run_tmux('capture-pane', '-t', session, '-p').stdout


def wait_for_capture(session, timeout):
    deadline = time.time() + timeout
    last = None
    stable_since = None
    best = ''

    while time.time() < deadline:
        captured = capture_pane(session)
        if captured.strip():
            best = captured
            if captured == last:
                if stable_since is not None and time.time() - stable_since >= 1:
                    return captured
            else:
                stable_since = time.time()
                last = captured
        time.sleep(0.25)

    return best if best.strip() else capture_pane(session)


def send_prompt(session, prompt_text):
    for line_number, line in enumerate(prompt_text.splitlines()):
        if line_number:
            run_tmux('send-keys', '-t', session, 'C-j')
        if line:
            run_tmux('send-keys', '-t', session, '-l', line)
    run_tmux('send-keys', '-t', session, 'Enter')


def remove_tree(path):
    if not path:
        return
    for _ in range(5):
        try:
            shutil.rmtree(path)
            return
        except FileNotFoundError:
            return
        except OSError:
            time.sleep(0.2)


def copy_fixture(fixtures_dir, tmp_root, prompt_name, bin_name):
    prefix = f"{safe_name(prompt_name)}-{safe_name(bin_name)}-"
    workdir = tempfile.mkdtemp(prefix=prefix, dir=tmp_root)
    shutil.copytree(fixtures_dir, workdir, dirs_exist_ok=True)
    return workdir


def capture_prompt(binary_path, bin_name, prompt_name, prompt_text, output_path,
                   fixtures_dir, tmp_root, rows, cols, wait_start,
                   wait_prompt, wait_exit, keep_workdirs):
    session = f"opencola-capture-{os.getpid()}-{uuid.uuid4().hex[:10]}"
    workdir = None
    test_home = None

    try:
        workdir = copy_fixture(fixtures_dir, tmp_root, prompt_name, bin_name)
        test_home = tempfile.mkdtemp(prefix=f"home-{safe_name(bin_name)}-", dir=tmp_root)

        command = f"env HOME={shlex.quote(test_home)} {shlex.quote(binary_path)}"
        run_tmux(
            'new-session',
            '-d',
            '-s', session,
            '-x', str(cols),
            '-y', str(rows),
            '-c', workdir,
            command,
        )

        time.sleep(wait_start)

        if prompt_text:
            send_prompt(session, prompt_text)

        captured = wait_for_capture(session, wait_prompt)
        with open(output_path, 'w', encoding='utf-8') as f:
            f.write(captured)

        run_tmux('send-keys', '-t', session, '/exit', 'Enter', check=False)
        wait_for_tmux_session_exit(session, wait_exit)
        return True, workdir

    except Exception as exc:
        print(f"  ERROR: {exc}", file=sys.stderr)
        return False, workdir

    finally:
        run_tmux('kill-session', '-t', session, check=False)

        if not keep_workdirs:
            for path in (test_home, workdir):
                remove_tree(path)


def main():
    root = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
    opencode_bin = os.environ.get(
        'OPENCODE_BIN',
        shutil.which('opencode') or '/home/francesco/.opencode/bin/opencode',
    )
    opencola_bin = os.environ.get('OPENCOLA_BIN', os.path.join(root, 'opencola'))
    prompts_dir = os.environ.get('PROMPTS_DIR', os.path.join(root, 'tests', 'prompts'))
    fixtures_dir = os.environ.get(
        'FIXTURES_DIR',
        os.path.join(root, 'tests', 'fixtures', 'demo-project'),
    )
    captures_dir = os.environ.get('CAPTURES_DIR', os.path.join(root, 'tests', 'captures'))
    tmp_root = os.environ.get(
        'PROMPT_TMP_ROOT',
        os.path.join(root, 'tests', 'fixtures', '.tmp', 'run-prompts'),
    )

    rows = env_int('TMUX_HEIGHT', 40)
    cols = env_int('TMUX_WIDTH', 120)
    wait_start = env_float('WAIT_START', 5)
    wait_prompt = env_float('WAIT_PROMPT', 5)
    wait_exit = env_float('WAIT_EXIT', 1)
    keep_workdirs = truthy_env('KEEP_PROMPT_WORKDIRS')

    os.makedirs(captures_dir, exist_ok=True)
    os.makedirs(tmp_root, exist_ok=True)

    if not shutil.which('tmux'):
        print("ERROR: tmux not found in PATH")
        sys.exit(1)

    for name, path in [('opencode', opencode_bin), ('opencola', opencola_bin)]:
        if not os.access(path, os.X_OK):
            print(f"ERROR: {name} not found at {path}")
            sys.exit(1)

    if not os.path.isdir(fixtures_dir):
        print(f"ERROR: fixtures directory not found at {fixtures_dir}")
        sys.exit(1)

    prompt_files = sorted(f for f in os.listdir(prompts_dir) if f.endswith('.txt'))
    if not prompt_files:
        print("ERROR: no prompt files in", prompts_dir)
        sys.exit(1)

    print("=" * 50)
    print("  Prompt Comparison Tests (tmux)")
    print("=" * 50)
    print(f"  Fixtures: {fixtures_dir}")
    print(f"  Workdirs: {tmp_root}")
    print(f"  Captures: {captures_dir}")
    print()

    total = 0
    failed = 0

    for prompt_file in prompt_files:
        prompt_path = os.path.join(prompts_dir, prompt_file)
        with open(prompt_path, 'r', encoding='utf-8') as f:
            prompt_text = f.read().strip()

        basename = os.path.splitext(prompt_file)[0]
        total += 1
        print(f"[{total}] {basename}")

        for bin_name, bin_path in [('opencode', opencode_bin), ('opencola', opencola_bin)]:
            out_file = os.path.join(captures_dir, f"{basename}-{bin_name}.txt")
            print(f"  {bin_name} ... ", end='', flush=True)
            ok, workdir = capture_prompt(
                bin_path,
                bin_name,
                basename,
                prompt_text,
                out_file,
                fixtures_dir,
                tmp_root,
                rows,
                cols,
                wait_start,
                wait_prompt,
                wait_exit,
                keep_workdirs,
            )
            if ok:
                with open(out_file, 'r', encoding='utf-8') as f:
                    lines = [line for line in f.read().split('\n') if line.strip()]
                msg = f"OK ({len(lines)} non-blank lines), saved: {os.path.basename(out_file)}"
                if keep_workdirs and workdir:
                    msg += f", workdir: {workdir}"
                print(msg)
            else:
                print("FAIL")
                failed += 1

        print()

    print("=" * 50)
    if failed:
        print(f"  Failed: {failed}/{total}")
        sys.exit(1)

    print(f"  All {total} tests completed.")
    print("=" * 50)


if __name__ == '__main__':
    main()
