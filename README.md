# Project Nile

Grow a grain-producing village on a local 32×32 Nile map. Drag out housing
plots along a road. Immigrants walk in and settle them. Wells supply water,
farmers grow grain, and the granary supplies a bazaar whose sellers carry food
to nearby homes. Inspect a house to see whether it can grow.

The first three housing capacities are 5, 7 and 9. The farm calendar, staffing
and delivery quantities are sandbox rules. Desirability, treasury and historical
unlocks remain future work.

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
