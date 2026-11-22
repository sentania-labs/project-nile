# Project Nile

Pharaoh at street level. One continuous Nile civilization at regional level.

This is an **interactive visual and architecture study**, not the first gameplay
milestone. The default scene is an illustrated Egyptian street with separate
housing, granary, bazaar, palms and terrain assets. Zoom, pan, inspect buildings
and toggle their footprints. People are static scale references; housing,
services, farming and trade are not simulated. No original Pharaoh code or
artwork is distributed here. Scott's visual acceptance is pending.

The rejected primitive 3D scene remains a separate Go/SQLite diagnostic.

## Run the study

On Linux with Go 1.26.5+, Godot 4.7.2 and Python 3:

```sh
make run         # illustrated street; wheel zoom, right-drag pan, click inspect
make diagnostic  # earlier 3D scene, local Go sidecar and SQLite controls
make check       # lint, race tests, IPC/crash probe, both actual Godot scenes
make capture     # five resolutions plus street detail, requires xvfb-run
```

The diagnostic starts a bundled Go executable over private stdio pipes, with no
server, listening socket, account or database installation. First build downloads
Go modules; running the finished bundle is offline. Advance, save, advance again,
then restore. Restart recovers only explicitly saved state. The visual scene is
independent of this diagnostic and does not launch a simulation.

`CAPTURES=/local/artifacts make capture` changes the image destination. Active
test databases use local `/tmp`; `/mnt/holding` is NFS and is for retained
artifacts, never active SQLite saves. `NILE_SIM` and `NILE_SAVE` override the
diagnostic executable and local save path. Closing its window stops the sidecar.
CI calls the same `make check` target.

Windows x64 and macOS arm64 sidecars cross-compile. Native application operation,
packaging, signing and performance on those systems remain untested. The diagnostic
save format is versioned but is **not** a stable released civilization format.

## Portable Linux study

The local Linux bundle is at
`/mnt/holding/project-nile/street-v2/linux-portable/nile`.
It includes the Godot runtime, resource pack, Go sidecar and dependency notices;
launch `nile` directly from that directory. It is an unsigned visual study,
not a tagged game release. To build another bundle into a **new** directory:

```sh
python3 tools/package_linux.py --templates /path/to/Godot_v4.7.2-stable_export_templates.tpz --output /path/to/new/nile-study
```

The script verifies the official template archive's SHA256. Packaging fetches
dependency notices; running the bundle is offline.

## Working references

- [Product direction and next playable increment](docs/product-spec.md)
- [Local architecture evidence and recommendation](docs/architecture.md)
- [Pharaoh mechanics evidence registry](docs/pharaoh-mechanics.md)
- [Frontier reuse and shared graphics pipeline](docs/graphics-pipeline.md)
- [Decisions and remaining questions](docs/decisions.md)

Development is directed by Scott through short playable iterations. Frontier
remains independent. No paid asset-generation budget has been authorized.
