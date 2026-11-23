"""Promote a processed Project Nile prop pack and update Godot art provenance."""
import argparse
import hashlib
import json
from pathlib import Path
from PIL import Image

ROOT = Path(__file__).resolve().parents[2]
ART = ROOT / "client/assets/street"


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def promote(pack, archive):
    plan = json.loads(pack.read_text())
    name = plan["name"]
    processed = archive / (name + "-out")
    source = archive / (name + ".png")
    prompt = ART / "prompts" / (name + ".txt")
    manifest = ART / "city-assets.json"
    registry = ART / "provenance.json"
    specs = json.loads(manifest.read_text())
    provenance = json.loads(registry.read_text())
    assets = {item["id"]: item for item in provenance["assets"]}
    qc = json.loads((processed / "pipeline-meta.json").read_text())
    if qc.get("qc_errors"):
        raise ValueError(qc["qc_errors"])
    for i, (kind, label, size, width) in enumerate(plan["assets"], 1):
        image = Image.open(processed / f"prop-{i}.png").convert("RGBA")
        if not image.getbbox():
            raise ValueError(f"Empty frame for {kind}")
        path = ART / (kind + ".png")
        image = image.crop(image.getbbox())
        image.save(path, optimize=True)
        specs[kind] = dict(id=kind, label=label, path=path.name,
                           footprint=[size, size], draw_width=width, anchor=[0.5, 1.0])
        assets[kind] = dict(id=kind, runtime_path=str(path.relative_to(ROOT)),
            source_path=str(source), source="Original built-in image_gen multi-prop pack",
            source_sha256=sha(source), sha256=sha(path), dimensions=list(image.size),
            mode="RGBA", prompt=str(prompt.relative_to(ROOT)), prompt_sha256=sha(prompt),
            transparent_background=True, validation="Chroma isolation and alpha bounds checked. See pack QC in source archive.")
    provenance["assets"] = list(assets.values())
    manifest.write_text(json.dumps(specs, indent=2) + "\n")
    registry.write_text(json.dumps(provenance, indent=2) + "\n")


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("pack", type=Path)
    parser.add_argument("--archive", type=Path, required=True)
    args = parser.parse_args()
    promote(args.pack, args.archive)
