"""Run the actual Godot client and child lifecycle; optional software captures."""
import argparse
import os
from pathlib import Path
import subprocess
import tempfile

ROOT = Path(__file__).resolve().parents[1]


def run(command, env):
    result = subprocess.run(command, cwd=ROOT, env=env, text=True,
                            stdout=subprocess.PIPE, stderr=subprocess.STDOUT,
                            timeout=120)
    print(result.stdout, end="")
    if result.returncode or "SCRIPT ERROR" in result.stdout or "ERROR:" in result.stdout:
        raise SystemExit(f"Godot failed: {command}")
    return result.stdout


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--godot", default="godot")
    parser.add_argument("--capture", type=Path)
    args = parser.parse_args()
    # Active SQLite lives on the local filesystem, never in the NFS artifact path.
    with tempfile.TemporaryDirectory(prefix="nile-godot-", dir="/tmp") as scratch:
        env = os.environ.copy()
        env["XDG_DATA_HOME"] = str(Path(scratch) / "userdata")
        env["NILE_SIM"] = str(ROOT / "bin/nile-sim")
        env["NILE_SAVE"] = str(Path(scratch) / "probe.nile")
        env.pop("NILE_CAPTURE_DIR", None)
        run([args.godot, "--headless", "--path", "client", "--editor", "--import", "--quit"], env)
        command = [args.godot, "--audio-driver", "Dummy", "--path", "client",
                   "--", "--diagnostic", "--smoke"]
        if args.capture:
            args.capture.mkdir(parents=True, exist_ok=True)
            env["NILE_CAPTURE_DIR"] = str(args.capture.resolve())
            command = ["xvfb-run", "-a", "-s", "-screen 0 3456x2160x24"] + command
        else:
            command.insert(1, "--headless")
        output = run(command, env)
        if "NILE_GODOT_SMOKE_PASS" not in output:
            raise SystemExit("Godot never completed the lifecycle checks")

        camera = [args.godot, "--headless", "--path", "client", "--script", str(ROOT / "tools/camera_check.gd")]
        if "NILE_CAMERA_PASS" not in run(camera, env):
            raise SystemExit("Camera controls failed through live game input")

        env["NILE_SANDBOX_SAVE"] = str(Path(scratch) / "sandbox.nile")
        env.pop("NILE_SANDBOX_CAPTURE", None)
        sandbox = [args.godot, "--headless", "--audio-driver", "Dummy", "--path",
                   "client", "--", "--sandbox-smoke"]
        if "NILE_SANDBOX_PASS" not in run(sandbox, env):
            raise SystemExit("Sandbox never completed its gameplay checks")

        env["NILE_SANDBOX_SAVE"] = str(Path(scratch) / "city.nile")
        env.pop("NILE_CITY_CAPTURE", None)
        if "NILE_GOLD_PASS" not in run([*sandbox[:-1], "--gold-smoke"], env):
            raise SystemExit("Gold delivery and in-flight replay did not complete")
        city = [*sandbox[:-1], "--city-smoke"]
        if "NILE_CITY_PASS" not in run(city, env):
            raise SystemExit("City never completed its monument and save/load checks")


if __name__ == "__main__":
    main()
