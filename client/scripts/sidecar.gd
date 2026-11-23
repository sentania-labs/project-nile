extends Node
## Nonblocking local transport shared by the diagnostic and playable sandbox.
signal response_received(reply: Dictionary)
signal failed(reason: String)

var child: Dictionary = {}
var incoming := PackedByteArray()
var pending := 0
var next_id := 1
var deadline := 0
var stopping := false
var message_limit := 65536

func start(mode: String = "") -> bool:
	var executable := OS.get_environment("NILE_SIM")
	if executable.is_empty():
		if OS.has_feature("editor"):
			executable = ProjectSettings.globalize_path("res://../bin/nile-sim")
		else:
			executable = OS.get_executable_path().get_base_dir().path_join("nile-sim")
		if OS.get_name() == "Windows": executable += ".exe"
	var save_path := OS.get_environment("NILE_SANDBOX_SAVE" if mode == "sandbox" else "NILE_SAVE")
	if save_path.is_empty():
		save_path = ProjectSettings.globalize_path("user://riverbank.nile" if mode == "sandbox" else "user://kickoff.nile")
	var arguments := PackedStringArray(["--save", save_path])
	if mode == "sandbox": arguments.append("--sandbox")
	message_limit = 2097152 if mode == "sandbox" else 65536
	child = OS.execute_with_pipe(executable, arguments, false)
	if child.is_empty():
		failed.emit("Could not start the local simulation. Run make build first.")
		return false
	incoming.clear()
	pending = 0
	stopping = false
	return true

func request(op: String, ticks: int = 0, fields: Dictionary = {}) -> bool:
	if child.is_empty() or pending != 0 or stopping: return false
	var message := {"id": next_id, "op": op}
	if op == "step": message["ticks"] = ticks
	for key: String in ["kind", "x", "y"]:
		if fields.has(key): message[key] = fields[key]
	pending = next_id
	next_id += 1
	deadline = Time.get_ticks_msec() + 5000
	var pipe: FileAccess = child.stdio
	pipe.store_string(JSON.stringify(message) + "\n")
	if pipe.get_error() != OK:
		_abort("The simulation pipe could not accept a command.")
		return false
	return true

func _process(_delta: float) -> void:
	if child.is_empty() or stopping: return
	# Bounded reads avoid blocking rendering and retain partial JSON lines.
	var pipe: FileAccess = child.stdio
	var bytes := pipe.get_buffer(8192)
	incoming.append_array(bytes)
	var diagnostics: FileAccess = child.stderr
	var errors := diagnostics.get_buffer(4096)
	if not errors.is_empty(): push_warning(errors.get_string_from_utf8())
	if incoming.size() > message_limit:
		_abort("The simulation exceeded the message limit.")
		return
	var newline := incoming.find(10)
	while newline >= 0:
		var line := incoming.slice(0, newline).get_string_from_utf8()
		incoming = incoming.slice(newline + 1)
		var parsed: Variant = JSON.parse_string(line)
		if not parsed is Dictionary or int(parsed.get("id", -1)) != pending:
			_abort("The simulation sent an invalid response.")
			return
		pending = 0
		response_received.emit(parsed)
		newline = incoming.find(10)
	if pending != 0 and Time.get_ticks_msec() > deadline:
		_abort("The simulation stopped responding. Restart to recover the last save.")
	elif not child.is_empty() and not OS.is_process_running(child.pid):
		child.pid = -1 # Already reaped; avoid querying the same dead PID again.
		_abort("The simulation exited. Restart to recover the last save.")

func _abort(reason: String) -> void:
	stop()
	failed.emit(reason)

func stop() -> void:
	if child.is_empty(): return
	stopping = true
	var pid: int = child.pid
	var pipe: FileAccess = child.stdio
	pipe.close() # EOF is the sidecar's normal shutdown path, including parent death.
	child.stderr.close()
	child.clear()
	pending = 0
	if pid < 0: return
	# Only on shutdown. Keep the grace period bounded, then reap a stuck child.
	for i in range(20):
		if not OS.is_process_running(pid): return
		OS.delay_msec(10)
	if OS.is_process_running(pid): OS.kill(pid)

func _exit_tree() -> void:
	stop()
