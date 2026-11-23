#!/usr/bin/env python3
"""Record real Godot play in a private display, with a city built through Go commands.

python3 client/tests/capture_motion.py --output /mnt/holding/project-nile/motion-review
Runtime art is in assets/motion; original sheets and QC remain in /mnt/holding.
"""
from __future__ import annotations

import argparse
import json
import os
from pathlib import Path
import signal
import subprocess
import sys
import tempfile
import time

ROOT = Path(__file__).resolve().parents[2]


def prepare_city(binary: Path, save: Path) -> None:
    process = subprocess.Popen(
        [str(binary), "--sandbox", "--save", str(save)],
        stdin=subprocess.PIPE, stdout=subprocess.PIPE, text=True,
    )
    request_id = 0

    def request(op: str, **fields):
        nonlocal request_id
        request_id += 1
        process.stdin.write(json.dumps({"id": request_id, "op": op, **fields}) + "\n")
        process.stdin.flush()
        response = json.loads(process.stdout.readline())
        if not response.get("ok"):
            raise RuntimeError(f"{op}: {response.get('error')}")
        return response["state"]

    def road(start, end):
        x, y = start
        tiles = [{"x": x, "y": y}]
        while x != end[0]:
            x += 1 if x < end[0] else -1
            tiles.append({"x": x, "y": y})
        while y != end[1]:
            y += 1 if y < end[1] else -1
            tiles.append({"x": x, "y": y})
        request("road", tiles=tiles)

    try:
        request("step", ticks=1000)
        request("build", kind="farm", x=7, y=14)
        request("build", kind="well", x=15, y=17)
        request("zone", x=14, y=19, x2=17, y2=19)
        request("step", ticks=1000)
        for kind, x, y in [("goldmine", 28, 4), ("claypit", 6, 5),
                           ("brickworks", 24, 12), ("storageyard", 25, 6), ("mastaba", 24, 17)]:
            request("build", kind=kind, x=x, y=y)
        for start, end in [((22, 10), (28, 10)), ((28, 10), (28, 6)),
                           ((8, 5), (10, 5)), ((25, 10), (25, 9)),
                           ((22, 12), (23, 12)), ((22, 18), (23, 18))]:
            road(start, end)
        request("set_policy", tax_rate=2, wage_rate=1, labor_priority="construction")
        for _ in range(500):
            state = request("step", ticks=4)
            if state["monument_progress"] >= 12:
                break
        else:
            raise RuntimeError("Prepared city never delivered and built its first 12 bricks")
        request("save")
    finally:
        process.stdin.close()
        process.wait(timeout=10)


def record(args) -> None:
    output = args.output.resolve()
    output.mkdir(parents=True, exist_ok=True)
    with tempfile.TemporaryDirectory(prefix="nile-motion-") as scratch:
        temp = Path(scratch)
        save = temp / "city.nile"
        ready = temp / "ready"
        prepare_city(args.sim.resolve(), save)
        env = dict(os.environ, NILE_SIM=str(args.sim.resolve()), NILE_SANDBOX_SAVE=str(save),
                   NILE_MOTION_READY=str(ready), NILE_MOTION_DIR=str(output))
        with (output / "godot.log").open("w") as log:
            game = subprocess.Popen(
                [args.godot, "--audio-driver", "Dummy", "--path", str(ROOT / "client"),
                 "--script", "res://tests/motion_capture.gd"],
                env=env, stdout=log, stderr=subprocess.STDOUT,
            )
            recorder = None
            try:
                deadline = time.monotonic() + 25
                while not ready.exists() and game.poll() is None and time.monotonic() < deadline:
                    time.sleep(0.1)
                if not ready.exists():
                    raise RuntimeError("Godot did not open the prepared city; inspect godot.log")
                recorder = subprocess.Popen([
                    "ffmpeg", "-y", "-loglevel", "error", "-f", "x11grab", "-framerate", "20",
                    "-video_size", "1920x1080", "-i", os.environ["DISPLAY"], "-c:v", "libx264",
                    "-preset", "veryfast", "-crf", "20", "-pix_fmt", "yuv420p", str(output / "city-motion.mp4"),
                ], stdin=subprocess.DEVNULL)
                result = game.wait(timeout=125)
                if result:
                    raise RuntimeError(f"Godot motion playthrough failed ({result}); inspect godot.log")
            finally:
                if game.poll() is None:
                    game.terminate()
                    game.wait(timeout=10)
                if recorder is not None and recorder.poll() is None:
                    recorder.send_signal(signal.SIGINT)
                    recorder.wait(timeout=15)
        log_text = (output / "godot.log").read_text()
        if "NILE_MOTION_PASS" not in log_text or "ERROR:" in log_text:
            raise RuntimeError(log_text)
        print(log_text)
        print(output / "city-motion.mp4")


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--godot", default="godot")
    parser.add_argument("--sim", type=Path, default=ROOT / "bin/nile-sim")
    parser.add_argument("--output", type=Path, default=Path("/mnt/holding/project-nile/motion-review"))
    parser.add_argument("--virtual-display", action="store_true", help=argparse.SUPPRESS)
    args = parser.parse_args()
    if not args.virtual_display:
        subprocess.run(["xvfb-run", "-a", "-s", "-screen 0 1920x1080x24", sys.executable,
                        str(Path(__file__).resolve()), *sys.argv[1:], "--virtual-display"], check=True)
    else:
        record(args)


if __name__ == "__main__":
    main()
