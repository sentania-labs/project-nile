#!/usr/bin/env python3
"""Exercise the real local sidecar. All active SQLite files live in local /tmp.

This is IPC/persistence evidence for a five-field fixture, not a world benchmark.
"""
import argparse
import hashlib
import json
import os
from pathlib import Path
import selectors
import sqlite3
import statistics
import subprocess
import tempfile
import time


class Client:
    def __init__(self, binary, path):
        self.process = subprocess.Popen(
            [str(binary), "--save", str(path)], stdin=subprocess.PIPE,
            stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True, bufsize=1,
        )
        self.selector = selectors.DefaultSelector()
        self.selector.register(self.process.stdout, selectors.EVENT_READ)
        self.sequence = 0

    def request(self, op, **fields):
        self.sequence += 1
        self.process.stdin.write(json.dumps({"id": self.sequence, "op": op, **fields}) + "\n")
        self.process.stdin.flush()
        if not self.selector.select(5):
            raise AssertionError("sidecar response timeout")
        line = self.process.stdout.readline()
        if not line:
            raise AssertionError("sidecar exited: " + self.process.stderr.read())
        response = json.loads(line)
        assert response["id"] == self.sequence, response
        assert response["state"]["grain"] == 24, response
        return response

    def close(self):
        if self.process.poll() is None:
            self.process.kill()
        self.process.wait(timeout=5)
        self.selector.close()
        self.process.stdin.close()
        self.process.stdout.close()
        self.process.stderr.close()


def checked(client, op, **fields):
    result = client.request(op, **fields)
    assert result["ok"], result
    return result


def run(binary, samples):
    binary_sha256 = hashlib.sha256(binary.read_bytes()).hexdigest()
    clients = []
    with tempfile.TemporaryDirectory(prefix="nile-probe-", dir="/tmp") as temp:
        save = Path(temp) / "fixture.nile"
        def launch():
            client = Client(binary, save)
            clients.append(client)
            return client
        try:
            started = time.perf_counter_ns()
            client = launch()
            initial = checked(client, "state")
            startup_ms = (time.perf_counter_ns() - started) / 1e6
            assert initial["state"] == {"tick": 0, "seed": 17, "rng": 17, "walker": 0, "grain": 24}
            times = []
            for _ in range(samples):
                started = time.perf_counter_ns()
                checked(client, "state")
                times.append((time.perf_counter_ns() - started) / 1e6)
            saved = checked(client, "step", ticks=137)
            checked(client, "save")
            expected = checked(client, "step", ticks=863)
            assert not client.request("step", ticks=-1)["ok"]
            assert checked(client, "state")["hash"] == expected["hash"]
            duplicate = subprocess.run([str(binary), "--save", str(save)],
                                       input='', text=True, capture_output=True, timeout=5)
            assert duplicate.returncode != 0, "concurrent writer was allowed"
            checked(client, "quit")
            assert client.process.wait(timeout=5) == 0
            client = launch()
            restored = checked(client, "load")
            assert restored["hash"] == saved["hash"]
            assert checked(client, "step", ticks=863)["hash"] == expected["hash"]
            checked(client, "save")
            checked(client, "step", ticks=11)
            client.process.kill()
            client.process.wait(timeout=5)
            client = launch()
            assert checked(client, "load")["hash"] == expected["hash"]
            client.process.stdin.close()
            assert client.process.wait(timeout=5) == 0, "EOF must stop process"
            # A killed SQLite writer leaves an uncommitted revision; the last
            # committed snapshot must still be restored on the next open.
            crash_script = (
                "import os,sqlite3,sys; c=sqlite3.connect(sys.argv[1]); "
                "c.execute('BEGIN IMMEDIATE'); "
                "c.execute(\"UPDATE snapshots SET checksum='uncommitted'\"); os._exit(0)"
            )
            subprocess.run([os.sys.executable, "-c", crash_script, str(save)], check=True, timeout=5)
            client = launch()
            assert checked(client, "load")["hash"] == expected["hash"]
            checked(client, "quit")
            assert client.process.wait(timeout=5) == 0
            with sqlite3.connect(save) as db:
                assert db.execute("PRAGMA integrity_check").fetchone()[0] == "ok"
                assert db.execute("SELECT count(*) FROM snapshots").fetchone()[0] == 2
                db.execute("UPDATE snapshots SET checksum='corrupt' WHERE generation=(SELECT max(generation) FROM snapshots)")
                db.commit()
            client = launch()
            before = checked(client, "state")
            assert not client.request("load")["ok"]
            assert checked(client, "state")["hash"] == before["hash"]
            checked(client, "quit")
            assert client.process.wait(timeout=5) == 0
            assert hashlib.sha256(binary.read_bytes()).hexdigest() == binary_sha256, "binary changed during probe"
            return {
                "scope": "five-field diagnostic; no Pharaoh or large-world performance claim",
                "binary_sha256": binary_sha256,
                "samples": samples, "startup_ms": round(startup_ms, 3),
                "ipc_roundtrip_ms_p50": round(statistics.median(times), 4),
                "ipc_roundtrip_ms_p95": round(sorted(times)[int((len(times)-1)*.95)], 4),
                "replay_hash": expected["hash"], "sqlite_bytes": save.stat().st_size,
                "passed": ["initial_state", "inventory_conservation", "invalid_command_no_mutation",
                           "single_writer_exclusion", "save_restart_load_replay", "quit", "eof",
                           "crash_restart_last_commit", "interrupted_transaction_rollback",
                           "two_generations_retained", "sqlite_integrity", "corrupt_load_no_mutation"],
                "runtime_platform": os.sys.platform,
            }
        finally:
            for client in clients:
                client.close()


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("binary", type=Path)
    parser.add_argument("--samples", type=int, default=250)
    args = parser.parse_args()
    if args.samples < 1:
        parser.error("samples must be positive")
    print(json.dumps(run(args.binary.resolve(), args.samples), indent=2))
