"""Bundle the local desktop sandbox using official Godot templates, without publishing."""
import argparse
import hashlib
import json
import os
from pathlib import Path
import shutil
import subprocess
import urllib.request
from zipfile import ZipFile

ROOT = Path(__file__).resolve().parents[1]
TEMPLATE_SHA = "f298490b8d44d934be425a5a65a51bf15f422428b229a06a6e11d9ffea248011"


def digest(path):
    with path.open("rb") as stream:
        return hashlib.file_digest(stream, "sha256").hexdigest()


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--templates", required=True, type=Path)
    parser.add_argument("--output", required=True, type=Path)
    parser.add_argument("--godot", default="godot")
    parser.add_argument("--platform", choices=("linux", "windows"), default="linux")
    args = parser.parse_args()
    windows = args.platform == "windows"
    game = "Nile.exe" if windows else "nile"
    pack = "Nile.pck" if windows else "nile.pck"
    sim = "nile-sim.exe" if windows else "nile-sim"
    template = "windows_release_x86_64.exe" if windows else "linux_release.x86_64"
    preset = "Windows sandbox" if windows else "Linux study"
    if digest(args.templates) != TEMPLATE_SHA:
        raise SystemExit("Expected official Godot 4.7.2 standard export templates")
    output = args.output.resolve()
    output.mkdir(parents=True, exist_ok=False)
    with ZipFile(args.templates) as archive:
        if archive.read("templates/version.txt").decode().strip() != "4.7.2.stable":
            raise SystemExit("Template version mismatch")
        with archive.open("templates/" + template) as source:
            with (output / game).open("wb") as destination:
                shutil.copyfileobj(source, destination)
    (output / game).chmod(0o755)
    env = os.environ.copy()
    env.update(CGO_ENABLED="0", GOOS=args.platform, GOARCH="amd64")
    subprocess.run(["go", "build", "-trimpath", "-o", str(output / sim),
                    "./cmd/nile-sim"], cwd=ROOT, env=env, check=True)
    for flags in [["--editor", "--import", "--quit"],
                  ["--export-pack", preset, str(output / pack)]]:
        result = subprocess.run([args.godot, "--headless", "--path", "client"] + flags,
                                cwd=ROOT, text=True, stdout=subprocess.PIPE,
                                stderr=subprocess.STDOUT, check=True, timeout=120)
        if "ERROR:" in result.stdout:
            raise SystemExit(result.stdout)
    notices = output / "licenses"
    notices.mkdir()
    goroot = Path(subprocess.check_output(["go", "env", "GOROOT"], cwd=ROOT, text=True).strip())
    shutil.copyfile(goroot / "LICENSE", notices / "Go-runtime-LICENSE")
    for name in ["LICENSE.txt", "COPYRIGHT.txt", "AUTHORS.md"]:
        urllib.request.urlretrieve(
            "https://raw.githubusercontent.com/godotengine/godot/4.7.2-stable/" + name,
            notices / ("Godot-" + name))
    modules = subprocess.check_output(["go", "list", "-m", "-json", "all"], cwd=ROOT, text=True)
    decoder = json.JSONDecoder()
    while modules.strip():
        module, end = decoder.raw_decode(modules.lstrip())
        modules = modules.lstrip()[end:]
        if module.get("Main") or "Dir" not in module:
            continue
        for name in Path(module["Dir"]).iterdir():
            if name.is_file() and name.name.upper().startswith(("LICENSE", "COPYING", "COPYRIGHT")):
                destination = notices / "Go" / (module["Path"] + "@" + module["Version"])
                destination.mkdir(parents=True, exist_ok=True)
                shutil.copyfile(name, destination / name.name)
    (output / "README.txt").write_text(
        "Project Nile: Nile settlement sandbox.\n"
        f"Extract all files together and run {game}. Keep {sim} alongside it.\n"
        "Drag Housing beside roads. Provide well water and distribute grain to grow a neighborhood.\n"
        "Use the in-game save and load controls for this local settlement.\n"
        f"For the Go/SQLite diagnostic: {game} -- --diagnostic\n"
        "Diagnostic saves use your local Godot user-data directory. Restart restores the last explicit save.\n"
        "Godot and dependency notices are in licenses/. No new Project Nile license declared.\n")
    manifest = {name: digest(output / name) for name in [game, pack, sim]}
    (output / "sha256.json").write_text(json.dumps(manifest, indent=2) + "\n")
    print(output / game)


if __name__ == "__main__":
    main()
