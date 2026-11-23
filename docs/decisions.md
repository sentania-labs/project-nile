# Decisions and open questions

Updated November 22, 2026, America/Chicago.

| Status | Decision |
|---|---|
| Accepted by v0.6 | Local-first single-player, Godot rendering/input, SQLite, Go preferred subject to proof. No required home-lab service. |
| Accepted by v0.6 | One continuous world/clock; settlements and future nested districts are administrative areas inside it. Default local services, physical transport, original Pharaoh behavior. |
| Accepted by v0.6 | Interactive player-directed iterations; optional bounded delegation; no modification of Frontier or paid generation without budget. |
| Proven for this probe | Linux Godot can drive a bundled Go child over private pipes and restore validated SQLite state after restart/crash. Diagnostic commands never advance from rendered frame time. |
| Directed by Scott | Keep the Godot, Go and SQLite stack. Reproduce the original game’s buildings, unlocks and gameplay on one sandbox map first, with updated illustrated visuals. Regional features and a dedicated graphics polish pass follow sandbox gameplay. |
| Proposed next | Adopt that local integration for the first neighborhood after reviewing the evidence; retain native platform packaging as a required validation task. |
| Proposed next | Road loop/roadblock, 4–8 houses, well, finite-food granary, bazaar buyer/seller and the first three housing tiers. |
| Rejected by Scott | The kickoff visual study is not an acceptable Pharaoh art direction. Retain it only as a diagnostic for IPC/persistence. Do not extend its primitive geometry into production art. |
| Directed by Scott | Inspect original Pharaoh gameplay, local reference art, and the supplied New Era screenshot. Target detailed, illustrated isometric buildings, distinct silhouettes, textured terrain, dense street activity and Egyptian UI treatment. Original Pharaoh remains the mechanical authority. |
| Candidate for visual review | Original illustrated 2D sprites in a separate, interactive Godot street scene. No final renderer choice or production art acceptance is implied. |
| Open, evidence | Exact early walker lifecycle, service fields/timing and bazaar quantities. No deviations approved. |
| Deferred deliberately | Final `.nile` schema/encoding, chunk streaming, autosave/recovery UI, 512/1024 world benchmarks, shipping/signing, production art and GPU target. |

The earlier Gate 0 work is evidence, not permission to revive its autonomous
campaign or PostgreSQL assumptions. This kickoff ends at a reviewable foundation
and visual reference; dependent gameplay expansion follows Scott's feedback.

Visual feedback, November 22: Scott rejected the study and directed actual visual
reference inspection. Inspected all three `pharaoh_cleopatra-*-full.jpg` images
under `/mnt/holding/frontier-research/visual-references/`, early housing/granary/
bazaar sprites in `reference-engines/akhenaten/docs/wiki/player/buildings/`, and
extracted UI art in `binary-visuals/pharaoh-d93c8edc2617/`. Also inspected Scott's
original-game thumbnail and [New Era image](https://wolfsgamingblog.com/wp-content/uploads/2023/03/pharaoh_a_new_era-4.jpg).
These are study references, not assets to redistribute. The next visual proposal
must be grounded in those references, not a refinement of the rejected blockout.
