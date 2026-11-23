# Project Nile

Build a town along its overland Kingdom Road, or expand the starter village. Zone
housing, connect homes to wells and a bazaar, and keep grain moving through
the granary. Staff a firehouse and an architect’s post to protect the town.
Use the city overlays and overseers to inspect labor, risk and tax coverage.

To build a small mastaba, connect a clay pit and grain farm to a brickworks.
The farm supplies straw at harvest. Porters carry materials along roads, and
bricklayers need road access to the monument. The sandbox tracks each
delivery, construction progress and money spent. Save and Load preserve the
town and active jobs. On grassland, a staffed village palace can receive taxes
and gold delivered from a mine beside a metallic outcrop. The treasury credits
gold when its cart arrives. Floodwater spreads across the fields and recedes
before growth resumes.

Drag to preview a road and release to pay and build. Right-click a building
to inspect it. File handles saves, and Options lets you adjust the original
soundtrack or remap pan, zoom and rotation. Q/E turn the map by default.

The first three housing capacities are 5, 7 and 9. Desirability, higher
housing tiers and the campaign remain open. Farm yields, staffing, hazard
timing and the 64-brick mastaba are sandbox rules.

## Run

With Godot 4.7.2, Go 1.26.5+ and Python 3:

```sh
make run
make check
```

The game starts its bundled Go simulation over private pipes. The first source
build downloads Go dependencies. The finished application runs offline and
uses a local SQLite save. Keep active saves on local storage.

To package Windows x64 in a new directory (omit `--platform windows` for Linux x64):

```sh
python3 tools/package_linux.py --platform windows --templates /path/to/Godot_v4.7.2-stable_export_templates.tpz --output /path/to/new/nile
```

`make diagnostic` opens the earlier persistence probe. The illustrated street
study is still available at `res://scenes/street_study.tscn`. The Windows bundle opens with `Nile.exe` after extraction. macOS packaging and
native hardware performance remain unverified.

## Working references

- [Product direction and next playable increment](docs/product-spec.md)
- [Local architecture evidence and recommendation](docs/architecture.md)
- [Pharaoh mechanics evidence registry](docs/pharaoh-mechanics.md)
- [Frontier reuse and shared graphics pipeline](docs/graphics-pipeline.md)
- [Decisions and remaining questions](docs/decisions.md)

Development is directed by Scott through short playable iterations. Frontier
remains independent. No paid asset-generation budget has been authorized.
