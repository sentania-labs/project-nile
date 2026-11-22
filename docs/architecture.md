# Local application: tested candidate

**Recommendation for the next increment:** Godot + a bundled Go process using
private stdin/stdout pipes, with SQLite owned exclusively by that process. This
is a Linux-tested candidate, not final cross-platform architecture acceptance.

| Option | Evidence and tradeoff |
|---|---|
| Go sidecar, private pipes | Working Godot launch, nonblocking replies, crash/restart and SQLite restore. Preserves Go rules/tests without authentication or a network listener. Requires bundled executable and lifecycle supervision. |
| In-process native extension | Avoids process/message overhead, but Go runtime, ABI bindings and per-platform extension packaging need new engineering. Not benchmarked; no current measured need justifies it. |
| Godot-only simulation | Simplifies packaging but discards Go reuse and moves deterministic rules into a new implementation. Not prototyped; retained as a fallback if native platform probes fail. |

Godot's [`OS.execute_with_pipe`](https://docs.godotengine.org/en/stable/classes/class_os.html#class-os-method-execute-with-pipe)
supports redirected IO on the target platforms, including nonblocking mode.
API support is not proof of packaged macOS/Windows operation. Sandboxed macOS
helper placement/signing requires a native test. No socket or HTTP stack exists
in this probe.

## What is implemented

One Go owner holds a five-field diagnostic state: tick, seed, integer RNG state,
walker index on a 32-position ring, and constant grain inventory. UI commands
explicitly step the fixture; rendering never advances it. This is not the game
clock or Pharaoh routing. A future fixed-step Go scheduler owns wall-clock
accumulation, pause/speed and command ordering, independently of rendered frames.

Requests and responses are newline-delimited JSON with matching IDs. Godot permits
one in-flight command, retains partial input, caps response buffering at 64 KiB
and times out after five seconds. The server rejects oversized/unknown input.
JSON is adequate for this experiment; it is not a decision to serialize full
large worlds every frame. Measure chunk/delta exchange before expanding it.

Closing the pipe exits Go. The client detects failure and offers restart from
the last explicit save. Unsaved work is not silently reconstructed. Graceful
shutdown has a bounded grace period and kills a stuck owned child. The automated
Godot test moves the rendered walker, saves, moves again, restores, kills the
actual child and recovers the same hash and position after restart.

## SQLite proof

Schema 1 uses SQLite application ID `NLPR` (diagnostic), a snapshot BLOB and
SHA256 checksum, retaining two generations in one transaction. JSON payloads
are deliberately small diagnostic fixtures. Load validates fields, checksum,
schema and application identity before replacing active state. Future versions
are rejected; no migration is needed yet and none is invented.

SQLite exclusive locking holds the save's single-writer lease for the process
lifetime. DELETE journal with FULL synchronization keeps this proof simple.
WAL has not demonstrated a benefit here. See SQLite's
[atomic commit explanation](https://www.sqlite.org/atomiccommit.html).
Do not copy an open database. After clean closure this rollback-journal file is
portable; a future live-export feature should use the supported
[backup API or VACUUM INTO](https://www.sqlite.org/backup.html).
The retained previous generation is available for investigation, but automatic
fallback and a recovery UI are not implemented. Neither autosave nor a released
`.nile` compatibility guarantee is claimed.

## Evidence and remaining tests

November 22, 2026, America/Chicago: actual Linux process tests cover deterministic
save/restart/replay, second-writer exclusion, invalid commands, corrupt snapshots,
unsupported format, EOF/quit, forced process crash, interrupted transaction,
integrity check and two retained revisions. Unit/race tests complement them.
`tools/probe.py` emits measured latency with the exact binary SHA. Preliminary
250-request fixture results were about 0.03 ms median and 0.04 ms p95, not city
simulation or large-world persistence measurements. Re-run on the final binary.

Artifacts are under `/mnt/holding/project-nile/interactive-kickoff/`; active saves
use local disk. Actual Godot captures exist at all five requested resolutions,
using Mesa llvmpipe software rendering. This establishes layout, IPC and restore
behavior, not 60 FPS, historical accuracy or finished art. Windows x64/macOS arm64
cross-compilation is weaker than native launch, signing and crash-recovery proof.
Large-world saves, chunk streaming, real workloads and native platform packaging
remain future tests before dependent systems rely on this candidate architecture.

The local Linux bundle additionally uses the official hash-verified 4.7.2 release
template and a resource pack, with the static Go executable beside it. Its default
lookup uses the executable directory, not a development checkout path. The bundle
is for local review, without release tagging or publication. `package_linux.py`
retains engine/dependency notices and hashes its three runtime components.
