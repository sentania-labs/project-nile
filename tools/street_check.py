"""Exercise the Godot street study; capture five sizes and one street detail."""
import argparse
import os
from pathlib import Path
import struct
import subprocess

ROOT = Path(__file__).resolve().parents[1]
SIZES = ((1920, 1080), (1280, 720), (2560, 1440), (3456, 2160), (3440, 1440))
CAPTURES = {f"street-{width}x{height}.png": (width, height) for width, height in SIZES}
CAPTURES["street-detail.png"] = (1920, 1080)


def run(command, env):
    result = subprocess.run(
        command, cwd=ROOT, env=env, text=True,
        stdout=subprocess.PIPE, stderr=subprocess.STDOUT, timeout=180,
    )
    print(result.stdout, end="")
    if result.returncode or "SCRIPT ERROR" in result.stdout or "ERROR:" in result.stdout:
        raise SystemExit(f"Street study failed (exit {result.returncode}): {command}")
    return result.stdout


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--godot", default="godot")
    parser.add_argument("--capture", type=Path, help="Write actual rendered PNG captures here")
    args = parser.parse_args()
    env = os.environ.copy()
    env.pop("NILE_STREET_CAPTURE_DIR", None)
    run([args.godot, "--headless", "--path", "client", "--editor", "--import", "--quit"], env)
    command = [args.godot, "--audio-driver", "Dummy", "--path", "client",
               "res://scenes/street_study.tscn", "--", "--street-smoke"]
    if args.capture:
        args.capture.mkdir(parents=True, exist_ok=True)
        env["NILE_STREET_CAPTURE_DIR"] = str(args.capture.resolve())
        command = ["xvfb-run", "-a", "-s", "-screen 0 3456x2160x24"] + command
        # A previous capture must not allow a failed new run to appear successful.
        for filename in CAPTURES:
            (args.capture / filename).unlink(missing_ok=True)
    else:
        command.insert(1, "--headless")
    output = run(command, env)
    if "NILE_STREET_PASS" not in output:
        raise SystemExit("Street study never completed its interaction checks")
    if args.capture:
        for filename, (width, height) in CAPTURES.items():
            path = args.capture / filename
            with path.open("rb") as capture:
                header = capture.read(24)
            if (header[:8] != b"\x89PNG\r\n\x1a\n"
                    or header[12:16] != b"IHDR"
                    or struct.unpack(">II", header[16:24]) != (width, height)):
                raise SystemExit(f"Invalid PNG capture dimensions: {path}")


if __name__ == "__main__":
    main()
