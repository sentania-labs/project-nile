"""Remove magenta matte trapped between leaves in original Nile foliage sprites."""
import argparse
import hashlib
import json
from pathlib import Path
from PIL import Image


def clean(image):
    result = image.convert("RGBA")
    pixels = []
    for red, green, blue, alpha in result.getdata():
        spill = min(red - green, blue - green)
        if alpha and spill > 12 and blue > 72:
            # Estimate the keyed background contribution, then unmix the
            # remaining foreground in straight-alpha space.
            coverage = 1 - min(255, spill) / 255
            if coverage < 0.10:
                red, green, blue, alpha = (0, 0, 0, 0)
            else:
                red = round((red - spill) / coverage)
                green = min(255, round(green / coverage))
                blue = round((blue - spill) / coverage)
                alpha = round(alpha * coverage)
        pixels.append((red, green, blue, alpha))
    result.putdata(pixels)
    return result


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("archive", type=Path)
    parser.add_argument("kinds", nargs="+")
    args = parser.parse_args()
    art = Path(__file__).resolve().parents[2] / "client/assets/street"
    registry = art / "provenance.json"
    data = json.loads(registry.read_text())
    records = {r["id"]: r for r in data["assets"]}
    args.archive.mkdir(parents=True, exist_ok=True)
    for kind in args.kinds:
        path = art / (kind + ".png")
        source = args.archive / path.name
        record = records[kind]
        if not source.exists():
            source.write_bytes(path.read_bytes())
        clean(Image.open(source)).save(path, optimize=True)
        record["matte_cleanup_source"] = str(source)
        record["matte_cleanup_source_sha256"] = hashlib.sha256(source.read_bytes()).hexdigest()
        record["matte_cleanup_tool"] = "tools/assets/despill_foliage.py"
        record["sha256"] = hashlib.sha256(path.read_bytes()).hexdigest()
    registry.write_text(json.dumps(data, indent=2) + "\n")
