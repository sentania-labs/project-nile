# Mechanics registry: research before implementation

Original Pharaoh is authoritative unless Scott accepts a recorded deviation.
No original executable code or extracted commercial assets enter this repository.
This registry describes behavior and evidence locators, not copied implementation.

The requested `/mnt/holdings` is absent on this host. The verified corpus is
`/mnt/holding/frontier-research/decompilation/pharaoh-d93c8edc2617` (R below).
It contains 469 files / 37,548,944 bytes: passes 1–7, 275 C extracts representing
210 unique addresses, model tables, indexes/logs and a Ghidra project. The UI
interaction review has 43 additional extracts. The executable fingerprint is
`b6169404b9e22f228d07d5cabf889d25b8cd46d53023af9c9d0d2d6034cb741a`.
The adjacent readme says 1.0.0.0; executable/version equivalence is not verified.

Additional targeted research (D):
`/mnt/holding/frontier/settlement-loop-20261103/research/deeper-static/pharaoh-d93c8edc2617`.
Earlier Nile inventory and extraction results:
`/home/scott/agents/firstmate/data/nile-reference/` (`mechanics-registry.json`,
`housing-model.csv`, `mission-inventory.json`, `inventory-manifest.json`). These
are useful secondary indexes; their conclusions remain subject to original evidence.

| ID | Behavior/evidence | Confidence and open question | Nile tests / deviation |
|---|---|---|---|
| H01 | `R/models/Pharaoh_Model_Normal.txt`, `housing-table.json`: 20 tiers, 22 named fields; early capacities 5, 7, 9 | High for extracted data; runtime modifiers unverified | Sandbox implements Small Hut (5) and Large Hut (7); higher tiers pending |
| H02 | `R/pass7/00446be0.c`: current/next tier checks service and inventory requirements | Static control-flow evidence; map all service fields | Not implemented / none approved |
| H03 | `R/pass7/00445ac0.c`: maintenance before advancement; two failed-check increments precede downgrade | Static evidence; calendar timing unresolved | Not implemented / none approved |
| S01 | `D/selected2/00474d50.c`: bounded tile scan refreshes eligible field to 0x60 | Exact walker/service identity and elapsed-time meaning unresolved; not evidence for radial municipal coverage | Not implemented / none approved |
| T01 | `R/pass6/004450b0.c`: decay, desirability, then housing; 51 dispatcher states | Scheduling order supported; tick/calendar rate unknown | Diagnostic clock only / not a substitute |
| W01 | Original manual describes destination vs roaming walkers and roadblocks allowing destination travel | Documented behavior; original routing, junction tie rules, spawn/return timing need observation or tracing | Not implemented / ring walker is explicitly a test fixture |
| L01 | Original manual describes granary → buyer → bazaar → seller → household | Documented chain; quantities, reservations and dispatch timing unresolved | Not implemented / finite grain is test input only |
| U01 | `R/interaction-review-20261103/pass1/004b8220.c`: cursor/button transitions | Partial static UI traces, not complete interaction semantics | Original diagnostic controls only |
| P01 | Model tables, existing inventory and targeted goods extracts | Production ratios/timing, seasonal yields and full economic calculations not verified | Not implemented / none approved |

Manual reference: [original Pharaoh manual](https://cdn.akamai.steamstatic.com/steam/apps/564530/manuals/Pharaoh_-_manual.pdf?t=1481822384),
printed pages 50–52 and 86–88 for walkers/bazaars, page 23 for the interface.
These are manual statements, not newly observed game behavior. No original-game
playthrough occurred in this kickoff. Prior static extraction must not become
an unqualified claim of exact mechanical fidelity.

For the food-distribution increment, resolve the remaining early housing dependency fields,
bazaar delivery quantities and walker lifecycle; attach behavioral fixtures and
tests to these IDs. Keep confidence, tests and approved deviations current as
evidence improves. Seasonal farming and historical milestone adaptation follow
the relevant original model/campaign evidence, not invented modern-city rules.

## First sandbox

The original maintenance pass collects a building type and invokes a map scan
with distance two (R/pass7/00448440.c). The scan covers inclusive x and y
bounds and sets a building field for occupied tiles (R/pass7/004703a0.c).
Identifying the type as a well and the field as well access draws on the
existing static investigation. Runtime equivalence still needs playthrough
evidence. A well supplies nearby huts directly. A staffed water supply is
a separate future building.

Road-connected immigration, movement speed, dispatch cadence and evolution driven
by water alone are prototype rules. The extracted model includes desirability;
the sandbox currently exercises only the water dependency and first two capacities.
