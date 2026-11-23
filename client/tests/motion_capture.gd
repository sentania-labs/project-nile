extends SceneTree
## Run against a real prepared save. The game uses its normal Go clock throughout.
var game: Node
var seen_frames: Dictionary = {}
var seen_roles: Dictionary = {}
var seen_work_frames: Dictionary = {}
var seen_builder_work := false
var saw_courses := false
var directory := ""

func _initialize() -> void: _run.call_deferred()

func _observe() -> void:
	for id: String in game.walker_nodes:
		var sprite: Sprite2D = game.walker_nodes[id]
		var data: Dictionary = game.walker_data[id]
		seen_roles[str(data.role)] = true
		var clip := str(sprite.get_meta("motion_clip", ""))
		seen_frames[clip + ":" + str(sprite.get_meta("motion_frame", -1))] = true
		if data.role == "builder" and str(data.get("stage", "")) == "work": seen_builder_work = true
	for id: String in game.motion.monuments:
		if game.motion.monuments[id].visible: saw_courses = true
	for id: String in game.motion.sites:
		var site: Dictionary = game.motion.sites[id]
		if not site.group.visible or not site.active: continue
		for sprite: Sprite2D in site.actors:
			if not sprite.visible: continue
			var clip := str(sprite.get_meta("motion_clip", ""))
			if not seen_work_frames.has(clip): seen_work_frames[clip] = {}
			seen_work_frames[clip][int(sprite.get_meta("motion_frame", -1))] = true

func _capture(filename: String) -> void:
	await RenderingServer.frame_post_draw
	root.get_texture().get_image().save_png(directory.path_join(filename))

func _watch(seconds: float, filename: String) -> void:
	var until := Time.get_ticks_msec() + int(seconds * 1000)
	var captured := false
	while Time.get_ticks_msec() < until:
		await process_frame
		_observe()
		if not captured and Time.get_ticks_msec() > until - int(seconds * 500):
			captured = true
			await _capture(filename)

func _run() -> void:
	directory = OS.get_environment("NILE_MOTION_DIR")
	if directory.is_empty(): directory = "/mnt/holding/project-nile/motion-review"
	DirAccess.make_dir_recursive_absolute(directory)
	root.size = Vector2i(1920, 1080)
	root.set_meta("nile_start_choice", "load")
	game = load("res://scenes/sandbox.tscn").instantiate()
	root.add_child(game)
	var deadline := Time.get_ticks_msec() + 20000
	while (not game.startup_done or game.paused) and Time.get_ticks_msec() < deadline: await process_frame
	if not game.startup_done or game.paused:
		push_error("Motion capture could not open its prepared city")
		quit(1)
		return
	await game._smoke_click_button(game.pause_button)
	await game._wait_idle()
	var paused_clock: float = game.motion.clock
	await create_timer(0.25).timeout
	if game.motion.clock != paused_clock:
		push_error("Work animation advanced while paused")
		quit(1)
		return
	game.focus = game._iso(17, 13)
	game._set_view(1.55)
	await game._smoke_click_button(game.pause_button)
	await game._wait_idle()
	var ready := OS.get_environment("NILE_MOTION_READY")
	if not ready.is_empty():
		var file := FileAccess.open(ready, FileAccess.WRITE)
		file.store_string("ready")
		file.close()
	await _watch(8, "market.png")
	game.focus = game._iso(8, 13)
	game._set_view(1.65)
	await _watch(6, "farm.png")
	game.focus = game._iso(27, 6)
	game._set_view(1.85)
	await _watch(8, "mine.png")
	game.focus = game._iso(25, 14)
	game._set_view(1.55)
	await _watch(8, "industry.png")
	game.focus = game._iso(25, 18)
	game._set_view(1.9)
	await game._smoke_click_button(game.speed_button)
	await game._wait_idle()
	await game._smoke_click_button(game.speed_button)
	await game._wait_idle()
	await _watch(12, "mastaba-courses.png")
	var finish_deadline := Time.get_ticks_msec() + 50000
	while int(game.state.get("monument_progress", 0)) < 64 and Time.get_ticks_msec() < finish_deadline:
		await process_frame
		_observe()
	if game.city_ui.completion_modal.visible:
		await game._smoke_click_button(game.city_ui.completion_continue)
		await game._wait_idle()
	await _capture("mastaba-complete.png")
	var work_varied := true
	for clip: String in ["worker-dig", "worker-brick", "vendor-work"]:
		if seen_work_frames.get(clip, {}).size() < 2: work_varied = false
	if not seen_roles.has_all(["goldcart", "porter", "builder", "seller"]) or not seen_builder_work or not saw_courses or seen_frames.size() < 20 or not work_varied or int(game.state.get("monument_progress", 0)) != 64:
		push_error("Motion capture missed physical work or varied animation: " + JSON.stringify({"roles": seen_roles.keys(), "frames": seen_frames.size(), "builder": seen_builder_work, "courses": saw_courses}))
		quit(1)
		return
	print("NILE_MOTION_PASS real-clock directional-feet loaded-empty-haul staff-work builder-work authoritative-courses paused-freeze frames=" + str(seen_frames.size()))
	game.sim.stop()
	quit(0)
