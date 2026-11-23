# Mechanics registry: research before implementation

Original Pharaoh is authoritative unless Scott accepts a recorded deviation.
No original executable code or extracted commercial assets enter this repository.
This registry describes behavior and evidence locators, not copied implementation.

The requested `/mnt/holdings` is absent on this host. The verified corpus is
`/mnt/holding/frontier-research/decompilation/pharaoh-d93c8edc2617` (R below).
It contains 469 files / 37,548,944 bytes: passes 1–7, 275 C extracts representing
210 unique addresses, model tables, indexes/logs and a Ghidra project. These
totals include 43 UI interaction review extracts. The executable fingerprint is
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
| H01 | `R/models/Pharaoh_Model_Normal.txt`, `housing-table.json`: 20 tiers, 22 named fields; early capacities 5, 7, 9 | High for extracted data; runtime modifiers unverified | Crude Hut (5), Sturdy Hut (7), Meager Shanty (9) implemented |
| H02 | `R/pass7/00446be0.c`: current/next tier checks service and inventory requirements | Static control-flow evidence; map all service fields | Not implemented / none approved |
| H03 | `R/pass7/00445ac0.c`: maintenance before advancement; two failed-check increments precede downgrade | Static evidence; calendar timing unresolved | Not implemented / none approved |
| S01 | `D/selected2/00474d50.c`: bounded tile scan refreshes eligible field to 0x60 | Exact walker/service identity and elapsed-time meaning unresolved; not evidence for radial municipal coverage | Not implemented / none approved |
| T01 | `R/pass6/004450b0.c`: decay, desirability, then housing; 51 dispatcher states | Scheduling order supported; tick/calendar rate unknown | Diagnostic clock only / not a substitute |
| W01 | Original manual describes destination vs roaming walkers and roadblocks allowing destination travel | Documented behavior; original routing, junction tie rules, spawn/return timing need observation or tracing | Destination migrants, freight and buyers plus roaming sellers; roadblock rules and junction parity pending |
| L01 | Original manual describes granary → buyer → bazaar → seller → household | Documented chain; quantities, reservations and dispatch timing unresolved | Working grain chain with road walkers; dispatch interval, load size and household consumption are provisional |
| U01 | `R/interaction-review-20261103/pass1/004b8220.c`: cursor/button transitions | Partial static UI traces, not complete interaction semantics | Original diagnostic controls only |
| P01 | Model tables, existing inventory and targeted goods extracts | Production ratios/timing, seasonal yields and full economic calculations not verified | Not implemented / none approved |

Manual reference: [original Pharaoh manual](https://cdn.akamai.steamstatic.com/steam/apps/564530/manuals/Pharaoh_-_manual.pdf?t=1481822384),
printed pages 50–52 and 86–88 for walkers/bazaars, page 23 for the interface.
These are manual statements, not newly observed game behavior. Original disc play was subsequently observed under Wine (below). Prior static extraction must not become
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

The player designates vacant plots. Immigrants travel to them from the entry
road. A watered hut can advance to Sturdy Hut. Access to a bazaar and a type of
food can support a Meager Shanty. The original table also has desirability
thresholds. Those thresholds, original migration cadence, exact agricultural
yields and original walker dispatch intervals remain unverified in play.


## Original disc comparison, November 22, 2026

Ran the archived Pharaoh disc under Wine 10, used the Normal difficulty
Sandbox and Alexandria custom missions, and started the Nubt campaign.
In Nubt one drag placed six roadside housing markers. Thirty residents moved in. The occupied Crude Hut inspector reported five people and identified water as its next requirement. The campaign objective named the third tier Meager Shanty.
The municipal menu showed 30 debens each for the firehouse and architect’s
post and 75 for the tax collector. Distribution showed 250 for the granary,
40 for the bazaar and 70 for the storage yard. Compare the Normal building
table for the supported construction prices.

Alexandria offered monuments through the religious buildings menu. After a
large pyramid site was placed, the construction foreman reported that workers
could not reach its staging area. The monument overseer showed zero progress
and described the labor and materials required. This observation establishes
the access feedback and progression interface. It does not establish the
small mastaba’s footprint, quantities or timing. Nile uses a compact 3×3
mastaba and 64 brick units.

The private, original-window captures are under
`/mnt/holding/project-nile/original-wine-VwPQ/`, including
`og-housing-tutorial.png`, `og-municipal-services.png`,
`og-construction-foreman.png` and `og-monument-overseer.png`.
Only separately created artwork ships in Nile.

## Capability coverage and campaign verification

This is a coverage register, not a claim that the sandbox implements the original
game. **Documented** means an original manual or shipped data names the capability;
**static** means a traced predicate; **observed** means a recorded original-game
session. None alone establishes complete behavioral parity. Current prototype
tests verify Nile's behavior, not agreement with Pharaoh. No new deviation is
approved by this inventory.

Sources used below:

- **PM:** [original Pharaoh manual](https://cdn.akamai.steamstatic.com/steam/apps/564530/manuals/Pharaoh_-_manual.pdf?t=1481822384),
  printed page numbers. Local text: `/mnt/holding/frontier/settlement-loop-20261103/research/pharaoh-manual.txt`.
- **PR:** original disc `readme.txt` in `/mnt/holding/frontier-research/work/pharaoh-d93c8edc2617/files/`,
  version 1.0.0.0. Its corrections take precedence over PM.
- **CM:** [original Cleopatra manual, third-party reproduction](https://manualmachine.com/gamespc/pharaohcleopatra/1118224-user-manual/),
  sections named below, inspected November 22, 2026. This documents expansion
  capabilities; it is not a verified match to a local expansion executable.
- **OC:** `/mnt/holding/project-nile/original-wine-VwPQ/Executable/`, the original
  disc installation used for the observed comparison above.

| ID | Capability named by original evidence | Evidence | Nile coverage / next verification |
|---|---|---|---|
| C01 | Housing, desirability, goods, migration and labor | PM pp. 40–57; R model and H01–H03 | Three-tier prototype; verify all thresholds, consumption, merge/split and recruitment. |
| C02 | Floodplain/meadow farms, work camps, irrigation, hunting, fishing, cattle | PM pp. 59–67 | Grain prototype; trace fertility, seasons, crews, yields and other food chains. |
| C03 | Quarries, mines, reeds, wood, crops; manufacturers and shipwrights | PM pp. 69–79, appendix pp. 222–234 | Clay/straw/bricks prototype; verify recipes, staffing, output and resource eligibility. |
| C04 | Granaries, yards, bazaars, docks, land/river trade | PM pp. 81–95; PR storage corrections | Local cargo prototype; verify orders, capacities, buyers, sellers and trade limits. |
| C05 | Crime, fire, collapse, palaces, taxes, wages, roadblocks, ferries, bridges | PM pp. 97–115 | Partial patrol/fiscal model; quantify original rules and routing. |
| C06 | Gods, shrines, temples, complexes, festivals; health, education, entertainment | PM pp. 117–127, 151–166 | Unimplemented beyond well; verify each walker and dependency before housing expansion. |
| M01 | Infantry, archers, chariots; recruiters, academies, forts; morale, formations | PM pp. 169–177 | Unimplemented; establish recruitment, equipment, combat and replenishment fixtures. |
| M02 | Warships, troop transports, walls, towers, gatehouses; kingdom deployments | PM pp. 177–189; PR troop-access and transport corrections | Unimplemented; verify embarkation, naval orders and off-map service. |
| N01 | Mastabas; stepped, bent, true and brick-core pyramids; sphinxes, obelisks, sun temples, mausoleums | PM pp. 129–149; PR monument corrections | One provisional mastaba. Verify each variant's footprint, resources, crews, stages and burial provisions. |
| E01 | Ratings, requests, gifts, debt, invasions; mission-specific availability | PM pp. 17–19, 191–195; PR control-panel/messages corrections | No campaign/event engine; verify triggers and consequences per mission. |
| X01 | Henna, oil, lamps, paint, white marble; henna farms, paint/lamp makers, artisans | CM “Commodities”, “Industries”, production diagram | Cleopatra only; unimplemented. Verify recipe quantities and delivery contracts in expansion data. |
| X02 | Four royal-tomb sizes, Abu Simbel, Pharos, Great Library, Caesareum; cliff placement | CM “Monuments”, “Cliffs” | Cleopatra only; unimplemented. Do not reuse mastaba stages without evidence. |
| X03 | Zoo supplied with game meat/straw; additional original-mission availability | CM “Entertainment”, “Changes to Original Missions” | Cleopatra only; verify rating/unlock changes separately from base Pharaoh. |
| X04 | Blood/frog/hail/locust plagues; tomb robbery; optional monument acceleration | CM “Major Plagues”, introduction, “Faster Pyramid Construction” | Cleopatra only; trace option state, event damage, prevention and recovery. |
| X05 | Survival/deadline missions; campaign savings, troops and monument carry-over | CM “Survival & Time Limit Missions”, “Family Savings, Troop & Monument Carry-Over” | Unimplemented. Verify each mission's deadline, completion check and carried state. |

For existing prototype coverage, see `internal/sandbox/{world,economy,city,builders}_test.go`:
grain/material conservation, supply-loss devolution, patrol effects, construction
access and save/replay have tests. The 3×3/64-brick mastaba, worker counts, recipe
rates and simulation calendar remain provisional; they must not silently become
the original-game contract. A menu entry or a completed prototype animation is
not acceptance of an entire capability.

### Original campaign catalog now located

`OC/campaign.txt` supplies **38 base-game mission names and five period graphs**.
The earlier `nile-reference/mission-inventory.json` absence finding applied to
the partially extracted work tree; it does not describe this installed corpus.
The file's comments define branch selection, no backtracking, and path 0 as a
merge. The following is a factual index of that data, not copied mission scripts.
Numbers are original mission IDs; braces separate alternatives.

| Period; source lines | Mission progression from shipped data |
|---|---|
| Predynastic; 67–71 | 0 Nubt → 1 Thinis → 2 Perwadjyt |
| Archaic; 72–84 | 3 Nekhen → 4 Men-nefer → 5 Timna → {6 Behdet / 7 Abedju} |
| Old Kingdom; 85–138 | {8 Selima Oasis / 9 Abu} → 10 Saqqara → {11 Serabit Khadim / 12 Meidum} → {13 Buhen / 14 South Dahshur} → 15 North Dahshur → {16 Iunet / 17 On} → 18 Rostja → {19 Bahariya Oasis / 20 Djedu} → {21 Dunqul Oasis / 22 Dakhla Oasis} |
| Middle Kingdom; 139–178 | {23 Thinis / 24 Waset} → {25 Kebet / 26 Menat Khufu} → 27 Itjtawy → {28 Iken / 29 Sawu} → {30 Heh / 31 Bubastis} |
| New Kingdom; 179–end | {32 Khmun / 33 Sauty} → {34 Byblos → 36 Rowarty / 35 Baki → 37 Hetepsenusret} |

The duplicate Thinis names denote different mission IDs. Shipped file semantics
support this graph; complete runtime branch selection has not been observed.
`OC/MISSION1.PAK` is present (12,195,626 bytes), so original per-mission data is
available for targeted inspection rather than inferred solely from Akhenaten.
This pass did not decode that archive. OC also has 12 custom `.map` files,
including Sandbox, Alexandria and Warfare. Custom maps are not campaign nodes.

Source fingerprints:

- `campaign.txt`: `28df4bb13015ec8ed19e1b8a004547dbfc60f301dff10866f338fcdc051856f2`
- `MISSION1.PAK`: `42fc2104f66bf7a5b23dada1830a40bbe4ef0582d8d9c1cf3de4f6e5bc725b19`
- `OC/Pharaoh.exe` matches the executable fingerprint at the top of this document.

CM documents four expansion campaigns and distinguishes survival from deadline
missions. It names Migdol, Tanis, Maritis and Actium as survival missions, and Tut
in the Valley, Qadesh and Alexandria as deadline missions. Their exact durations,
objectives and runtime boundary conditions are still unverified.
The local Akhenaten scripts `m_038` through `m_052` provide a **secondary candidate
catalog**: Thutmose in the Valley, Tut in the Valley, Seti in the Valley, Sumur,
Qadesh, Abu Simbel, Ramses in the Valley, Pi-Yer, Migdol, Tanis, Alexandria,
Ptolemy's Alexandria, Maritis, Cleopatra's Alexandria and Actium. Their IDs,
grouping, ordering and scripted values require an original Cleopatra installation
and matching patch evidence; do not promote them to original data by association.

### Missing primary evidence and next mission checks

1. Retain the OC fingerprints, difficulty, executable version and game options
   with each observation. Obtain the original Cleopatra executable, campaign,
   mission archive and patch readme before treating expansion values as verified.
   Base Pharaoh and Cleopatra can change the same original mission.
2. Inspect original archive records and in-game briefings for each mission:
   starting date/resources, terrain eligibility, building availability, tutorial
   unlocks, ratings/population/housing/monument goals, burial provisions, trade
   partners, event conditions, deadlines and carry-over. Preserve raw evidence
   privately under `/mnt/holding`; put only independently written facts and
   source locators in Nile. Akhenaten is a comparison aid, not an oracle.
3. Start with Nubt → Thinis → Perwadjyt to test early unlocks and housing, then
   inspect Timna's two branches and the New Kingdom's linked branch endings.
   For original monuments inspect a mastaba mission and a pyramid mission;
   use custom Warfare only for isolated military experiments, not proof of
   campaign objectives. Choose expansion probes after matching its catalog.
4. For each trigger, capture the state immediately before, at and after the
   threshold, including loss/recovery and a save/reload. Record expected state
   changes separately from the presentation. Test at multiple game speeds;
   one successful screenshot cannot establish cadence or deterministic rules.
5. Promote a capability only when its source/observation, implementation location
   and a meaningful behavioral fixture agree. Keep unmeasured constants marked
   provisional. Translating mutually exclusive missions into one persistent
   civilization remains a product decision, separate from recovering the graph.
