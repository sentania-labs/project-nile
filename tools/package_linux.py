"""Bundle a local Linux study using official Godot templates, without publishing."""
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
    args = parser.parse_args()
    if digest(args.templates) != TEMPLATE_SHA:
        raise SystemExit("Expected official Godot 4.7.2 standard export templates")
    output = args.output.resolve()
    output.mkdir(parents=True, exist_ok=False)
    with ZipFile(args.templates) as archive:
        if archive.read("templates/version.txt").decode().strip() != "4.7.2.stable":
            raise SystemExit("Template version mismatch")
        with archive.open("templates/linux_release.x86_64") as source:
            with (output / "nile").open("wb") as destination:
                shutil.copyfileobj(source, destination)
    (output / "nile").chmod(0o755)
    env = os.environ.copy()
    env.update(CGO_ENABLED="0", GOOS="linux", GOARCH="amd64")
    subprocess.run(["go", "build", "-trimpath", "-o", str(output / "nile-sim"),
                    "./cmd/nile-sim"], cwd=ROOT, env=env, check=True)
    for flags in [["--editor", "--import", "--quit"],
                  ["--export-pack", "Linux study", str(output / "nile.pck")]]:
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
        "Project Nile illustrated street study, not a game release.\n"
        "Run ./nile on Linux x64. No Godot, Go, SQLite server or network installation needed.\n"
        "Wheel to zoom, right-drag to pan, click a building to inspect, F for footprints.\n"
        "Original generated visual candidates. People are static; gameplay systems are not implemented.\n"
        "For the Go/SQLite diagnostic: ./nile -- --diagnostic\n"
        "Diagnostic saves use your local Godot user-data directory. Restart restores the last explicit save.\n"
        "Godot and dependency notices are in licenses/. No new Project Nile license declared.\n")
    manifest = {name: digest(output / name) for name in ["nile", "nile.pck", "nile-sim"]}
    (output / "sha256.json").write_text(json.dumps(manifest, indent=2) + "\n")
    print(output / "nile")


if __name__ == "__main__":
    main()
