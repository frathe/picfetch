"""Exercise the actual signed broker/service and transferred process pipes."""
import errno
import json
import os
from pathlib import Path
import selectors
import signal
import subprocess
import sys

broker = str(Path(sys.argv[1]).resolve())


def command(mode):
    return [broker, mode]


def verify_worker(record, mode):
    assert record["tcp"] in (errno.EPERM, errno.EACCES), record
    assert record["udp"] in (errno.EPERM, errno.EACCES), record
    assert record["heic"] == (mode == "heic"), record
    assert record["similarity"] == (mode == "similarity"), record


def assert_reaped(pid):
    try:
        os.kill(pid, 0)
    except ProcessLookupError:
        return
    raise AssertionError(f"worker {pid} survived broker completion")


for mode in ("heic", "similarity"):
    result = subprocess.run(command(mode), input="echo\n", capture_output=True,
                            text=True, timeout=15)
    assert result.returncode == 0, (result.returncode, result.stdout, result.stderr)
    record = json.loads(result.stdout)
    verify_worker(record, mode)
    assert_reaped(record["pid"])
    print(f"PASS {mode}: transferred pipes, mode, TCP/UDP denial and reaped exit")

result = subprocess.run(command("invalid"), input="echo\n", capture_output=True,
                        text=True, timeout=15)
assert result.returncode == 2, result
assert not result.stdout, result
print("PASS invalid mode refused before worker launch")

for action, termination in (("hold", signal.SIGTERM), ("spawn", signal.SIGTERM),
                            ("spawn", signal.SIGKILL)):
    process = subprocess.Popen(command("similarity"), stdin=subprocess.PIPE,
                               stdout=subprocess.PIPE, stderr=subprocess.PIPE,
                               text=True)
    try:
        process.stdin.write(action + "\n")
        process.stdin.flush()
        with selectors.DefaultSelector() as ready:
            ready.register(process.stdout, selectors.EVENT_READ)
            assert ready.select(timeout=15), "worker did not publish readiness"
        record = json.loads(process.stdout.readline())
        verify_worker(record, "similarity")
        process.send_signal(termination)
        output, error = process.communicate(timeout=15)
        expected = -signal.SIGKILL if termination == signal.SIGKILL else 128 + signal.SIGKILL
        assert process.returncode == expected, (output, error, process.returncode)
        assert_reaped(record["pid"])
        # communicate observes EOF on both inherited pipes as well as broker
        # exit; a live descendant holding those handles would time out.
        print(f"PASS {action}/{termination.name}: leader reaped and descendant pipes closed")
    finally:
        if process.poll() is None:
            process.kill()
            process.communicate(timeout=15)

result = subprocess.run(command("similarity"), input="spawnexit\n",
                        capture_output=True, text=True, timeout=15, check=True)
record = json.loads(result.stdout)
verify_worker(record, "similarity")
assert_reaped(record["pid"])
print("PASS normal leader exit retires descendants before broker completion")

result = subprocess.run(command("grant"), input="", capture_output=True,
                        text=True, timeout=15)
assert result.returncode == 0, (result.returncode, result.stdout, result.stderr)
record = json.loads(result.stdout)
verify_worker(record, "similarity")
assert record["before"] in (errno.EPERM, errno.EACCES), record
assert record["granted"] == 1, record
assert record["sibling"] in (errno.EPERM, errno.EACCES), record
assert_reaped(record["pid"])
print("PASS bookmark transfer: private source denied before resolution, allowed after; sibling denied")
