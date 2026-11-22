# Project Nile

Pharaoh at street level. One continuous Nile civilization at regional level.

This repository begins with an **interactive kickoff study**, not the first
gameplay milestone. It contains an original Godot neighborhood reference and a
Go/SQLite architecture probe. You can move a diagnostic walker, save, restore,
and restart its local simulation. Housing, services, farming and trade are not
implemented. No original Pharaoh code or artwork is distributed here.

## Run the study

On Linux with Go 1.26.5+, Godot 4.7.2 and Python 3:

```sh
make run
```

Godot starts the bundled Go executable automatically over private stdio pipes.
There is no server, listening socket, account or database installation. First
build downloads Go modules; subsequent gameplay does not need the internet.
The study writes `kickoff.nile` in Godot's local user-data directory. Advance the
walker, save, advance again, then restore. Wheel zooms, arrow keys pan, Home resets.
Restart recovers the last **explicitly saved** state; unsaved steps are lost.

```sh
make check       # lint, race tests, actual IPC/crash probe, actual Godot lifecycle
make capture     # five real Xvfb/Godot screenshots, requires xvfb-run
```

`CAPTURES=/local/artifacts make capture` changes the image destination. Active
test databases always use local `/tmp`; `/mnt/holding` is NFS and is for retained
artifacts, never active SQLite saves. `NILE_SIM` and `NILE_SAVE` override the
development executable and local save path. Closing the application closes its
pipe, which stops the sidecar. CI calls the same `make check` target.

Windows x64 and macOS arm64 sidecars cross-compile. Native application operation,
packaging, signing and performance on those systems remain untested. The diagnostic
save format is versioned but is **not** a stable released civilization format.

## Portable Linux study

The local Linux bundle is at
`/mnt/holding/project-nile/interactive-kickoff/linux-portable/nile`.
It includes the Godot runtime, resource pack, Go sidecar and dependency notices;
launch `nile` directly from that directory. It is an unsigned diagnostic study,
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
