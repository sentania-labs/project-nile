# Frontier reuse and shared graphics

Read-only reference: Frontier `main` at
`851986020854970275f820f737b100391d124535`, verified against the remote on November
22, 2026. Snapshot: `/mnt/holding/project-nile/gate0/nile-foundation/frontier`.
The older `/home/scott/devel/frontier` checkout is dirty and was not modified.

Reuse its pure deterministic simulation boundary, canonical snapshot/invariant
patterns, deterministic pathfinding tie order, reservation/cargo accounting,
orthographic camera/picking, dirty chunks and MultiMesh concepts. Do not transplant
64×64 assumptions, per-settlement session ownership, medieval housing/seasons,
Crown orders, authentication, PostgreSQL or remote deployment machinery.
`server/internal/sim/village_services.go` uses road-reach BFS rather than Pharaoh
service walkers; it is specifically unsuitable as the mechanical implementation.

All requested graphics files were located in the pinned current snapshot:

| Source | Reuse |
|---|---|
| `tools/assets/README.md`, `assets/README.md`, `docs/development-environment.md` | Shared contract, tooling and environment |
| `tools/assets/prepare_camp_asset.py` | Normalize axes, scale, pivot, textures and provenance; generalize names before porting |
| `tools/assets/normalize_motion.py` | Hips-based in-place movement/loop closure; rig-specific |
| `tools/assets/make_shadow_assets.py` | Simplified shadow geometry; replace hard-coded species paths |
| `tools/assets/smoke_blender.py`, `verify_import.gd` | Real Blender/glTF/Godot interchange verification |
| `client/scenes/asset_review.tscn`, `client/scripts/asset_review.gd` | Actual engine asset review pattern |
| `client/tests/capture_art.py`, `asset_capture.gd`, `test_work_assets.py` | Rendered capture, clip/loop/rig checks |
| `assets/production-023.json` | Provenance structure; not a Nile credit allowance |

## Illustrated street candidate

`client/scenes/street_study.tscn` is the current visual proposal. It uses original
built-in image generation for separate 2D buildings, palms, people, material atlas
and Egyptian sidebar. The model identifier and per-image cost are not exposed by
the tool; no Meshy credits or paid API fallback were used. No copied reference
sprites are included. This candidate does not settle 2D versus 3D for the game.

Shared contract: `client/assets/street/manifest.json` defines each asset's tile
footprint, whole-image display width, normalized front-corner anchor, optional
atlas region, label and description. `layout.json` places independent objects and
roads on a 96×48 isometric grid. Decorations are nonselectable. Source PNGs remain
unmodified; Godot uses atlas regions, mipmaps and terrain masks at runtime.

`prompts/` retains the exact generation briefs; `provenance.json` records sources,
hashes, dimensions, tool disclosure, references and approval status. Large raw
copies and rendered captures live in `/mnt/holding/project-nile/street-v2/`.
`make capture` imports into real Godot and exercises selection, zoom, pan and HUD
controls before five resolution captures and a separate detail capture. This
follows Frontier's engine-review and provenance contract without importing its
3D-only rig processor into a 2D sprite pass.

Acceptance is pending Scott's review. Current limits: two repeated housing
silhouettes, static people, provisional footprints and some generated perspective
variation. These are visual candidates, not verified original housing tiers or
mechanical building dimensions. There are no construction or walker animations.

## Current original reference

**Rejected as art direction by Scott on November 22, 2026.** This scene and granary
remain engineering test fixtures only. The implementation was not grounded in
the available gameplay images before it was proposed, and does not meet the
recognizable-Pharaoh requirement. See `decisions.md` for the inspected references
and the user's correction. Structural validity does not imply artistic fitness.

`client/assets/granary-reference.glb` is the earlier Nile procedural candidate,
not a Pharaoh asset or a finished production asset. SHA256:
`c7e65d254cd654ad253b55ca6eb82c0da20666e81a4736c8b6167bf4093350f8`.
Author: Codex Gate 0 worker; generator: original Blender Python, no paid service.
Its retained source is `tools/granary_reference.py`. The Blender master, normalized
master, source hashes and complete provenance remain under
`/mnt/holding/project-nile/gate0/nile-foundation/cloud-granary/` and
`/home/scott/agents/firstmate/data/nile-foundation/candidate-provenance.json`.
Rights: original task-authored geometry/materials; no third-party art, textures,
audio or fonts were imported. No new public source license is implied.

Earlier processing used the pinned Frontier `prepare_camp_asset.py`, Blender
5.2.3 LTS, glTF Transform 4.5.1 and Godot 4.7.2. It produced one mesh, four matte
surfaces, 2,500 triangles, ground-centered pivot and `[6,3,4]` meter bounds.
The 3×2 tile / 2 meter tile footprint is a test contract, not approved Pharaoh
dimensions. Structural validation and original import/captures passed; this
kickoff reimported it and rendered it in the new neighborhood. Exact binary
regeneration is not claimed: earlier repeated exports found tiny unused UV changes.

Housing, palms, market, walker, terrain and HUD in `neighborhood.gd` are original
procedural reference geometry. They are visibly provisional, with no finished
rigs, construction stages, LODs, collision/picking or historical/artistic approval.
The style proposal uses warm silt/mudbrick, green floodplains, blue-green water,
clear silhouettes and restrained detail. The right-side panel references the
original game's spatial organization; its diagnostic controls are not final UI.

## Shared operations and promotion

Current repeatable interfaces: `blender --background --factory-startup
--python tools/granary_reference.py -- --output /path/to/isolated/output`,
`make run`, `make capture`, and `make check`. Source creation and inspection are
usable by any agent. Keep each asset job isolated and retain bulky masters on
holding. The future generalized processor should retain Frontier's validation
contract, not become a separate agent-specific script family.

Production sequence: brief, reference, generate, process, animate, validate,
import, render, evaluate, approve, integrate. Record identity, initiating agent,
generator/model, source/hash/tool versions, settings, rights, credits, runtime
destination, structural/render results and human art decision. Static reference
geometry is not evidence for animated workers/boats or production asset acceptance.

## DGX Spark

Earlier same-day discovery/execution is preserved in
`/home/scott/agents/firstmate/data/nile-foundation/local-llm-integration.md`.
It demonstrated structured output and supervised tool arguments through the
existing OpenAI-compatible gateway. Generated geometry/code failed some contracts;
repair needed feedback. Vision remains unverified. Use bounded extraction,
validation interpretation and tests with executable checks. Rediscover endpoint,
model and resource availability before new use. No inference service or agent
orchestrator is a runtime dependency. No new inference/generation credits were
spent in this kickoff.
