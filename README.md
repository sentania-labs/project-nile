# Project Nile

Build an early Egyptian neighborhood on a local 32×32 Nile map. Extend the
entry road, place housing and watch immigrants travel to connected homes. Build
a well near a Small Hut to provide water and support its first upgrade. Inspect
the house to see its population, water and capacity. Save and reload the town
locally, including migrants in transit.

This first free-build sandbox exercises two housing tiers. Their capacities
of five and seven come from Pharaoh's housing model. Migration cadence and
water-only evolution are provisional. Desirability, labor, food distribution,
treasury and historical unlocks are future gameplay increments.

## Run

With Godot 4.7.2, Go 1.26.5+ and Python 3:

```sh
make run
make check
```

The game starts its bundled Go simulation over private pipes. The first source
build downloads Go dependencies. The finished application runs offline and
uses a local SQLite save. Keep active saves on local storage.

The Linux bundle is at `/mnt/holding/project-nile/sandbox-01/linux-portable/nile`.
To package another copy in a new directory:

```sh
python3 tools/package_linux.py --templates /path/to/Godot_v4.7.2-stable_export_templates.tpz --output /path/to/new/nile
```

`make diagnostic` opens the earlier persistence probe. The illustrated street
study is still available at `res://scenes/street_study.tscn`. Windows and macOS
native application packaging and performance remain unverified.

## Working references

- [Product direction and next playable increment](docs/product-spec.md)
- [Local architecture evidence and recommendation](docs/architecture.md)
- [Pharaoh mechanics evidence registry](docs/pharaoh-mechanics.md)
- [Frontier reuse and shared graphics pipeline](docs/graphics-pipeline.md)
- [Decisions and remaining questions](docs/decisions.md)

Development is directed by Scott through short playable iterations. Frontier
remains independent. No paid asset-generation budget has been authorized.
