extends RefCounted

const RECENT := "user://last-civilization.cfg"
var game: Node
var chooser: FileDialog
var elapsed := 0.0
var active_path := ""
var last_hash := ""


static func last_path() -> String:
	var override := OS.get_environment("NILE_SANDBOX_SAVE")
	if not override.is_empty(): return override
	var config := ConfigFile.new()
	if config.load(RECENT)==OK:
		var path := str(config.get_value("city","path",""))
		if FileAccess.file_exists(path): return path
	return ProjectSettings.globalize_path("user://riverbank.nile")


static func new_path(label: String) -> String:
	var override := OS.get_environment("NILE_SANDBOX_SAVE")
	if not override.is_empty(): return override
	var stamp := Time.get_datetime_string_from_system().replace(":","-").replace("T"," ")
	var base := ProjectSettings.globalize_path("user://"+label+" "+stamp)
	var path := base+".nile"
	var suffix := 1
	while FileAccess.file_exists(path):
		suffix += 1
		path = "%s (%d).nile" % [base,suffix]
	return path


func setup(owner: Node) -> void:
	game=owner
	chooser=FileDialog.new()
	game.hud.add_child(chooser)
	chooser.access = FileDialog.ACCESS_FILESYSTEM
	chooser.filters = PackedStringArray(["*.nile ; Nile civilizations"])
	chooser.use_native_dialog = false
	chooser.file_selected.connect(func(path: String) -> void:
		var save := chooser.file_mode == FileDialog.FILE_MODE_SAVE_FILE
		game._queue_command("save_as" if save else "load_from",{"path":path}))


func show_dialog(save: bool) -> void:
	game._cancel_gesture()
	chooser.file_mode = FileDialog.FILE_MODE_SAVE_FILE if save else FileDialog.FILE_MODE_OPEN_FILE
	chooser.title = "Save civilization as" if save else "Open civilization"
	chooser.current_dir = active_path.get_base_dir() if not active_path.is_empty() else ProjectSettings.globalize_path("user://")
	if save: chooser.current_file = active_path.get_file()
	chooser.popup_centered_ratio(0.68)


func update(delta: float) -> void:
	if game.paused or not game.startup_done or game.sim.child.is_empty(): return
	elapsed += delta
	if elapsed < 180.0 or game.sim.pending!=0 or not game.command_queue.is_empty(): return
	elapsed = 0.0
	if game.last_reply.get("hash","")!=last_hash: game._queue_command("autosave")


func received(op: String,reply: Dictionary) -> void:
	if not reply.get("ok",false): return
	active_path = str(reply.get("save_path",active_path))
	if op not in ["save","save_as","autosave","load","load_from","load_previous","load_manual"]: return
	last_hash = str(reply.get("hash",""))
	elapsed = 0.0
	if not OS.get_environment("NILE_SANDBOX_SAVE").is_empty(): return
	var config := ConfigFile.new()
	config.set_value("city","path",active_path)
	config.save(RECENT)
