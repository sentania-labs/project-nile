extends SceneTree
## Load a town constructed through normal sidecar commands, then exercise the UI.
var game: Node
var directory := ""

func _initialize() -> void: _run.call_deferred()

func _check(ok: bool, reason: String) -> bool:
	if not ok:
		push_error(reason)
		quit(1)
	return ok

func _step(ticks: int) -> bool:
	game._queue_command("step", {"ticks": ticks})
	return await game._wait_idle()

func _capture(name: String) -> void:
	await process_frame
	await RenderingServer.frame_post_draw
	root.get_texture().get_image().save_png(directory.path_join(name))

func _run() -> void:
	directory = OS.get_environment("NILE_MILITARY_CAPTURE")
	if directory.is_empty(): directory = "/mnt/holding/project-nile/military-review-20261123"
	DirAccess.make_dir_recursive_absolute(directory)
	root.size = Vector2i(1920,1080)
	root.set_meta("nile_start_choice", "load")
	game = load("res://scenes/sandbox.tscn").instantiate()
	root.add_child(game)
	var deadline := Time.get_ticks_msec() + 20000
	while (not game.startup_done or game.paused) and Time.get_ticks_msec() < deadline: await process_frame
	if not _check(game.startup_done and not game.paused, "Prepared military town failed to load"): return
	await game._smoke_click_button(game.pause_button)
	if not _check(await game._wait_idle() and game.paused, "Military clock did not pause"): return
	game.focus = game._iso(18,16)
	game._set_view(0.85)
	await _capture("military-city.png")
	game.focus = game._iso(20,24)
	game._set_view(1.35)
	await _capture("fort-garrison.png")
	await game._smoke_click_button(game.city_ui.overseers_button)
	await game._smoke_click_button(game.city_ui.overseer_tabs["Military"])
	var controls = game.city_ui.military_controls
	if not _check(controls.visible and controls.companies.item_count == 1, "Military company controls missing"): return
	controls.x.value = 26
	controls.y.value = 18
	await game._smoke_click_button(controls.muster)
	if not _check(await game._wait_idle() and game.state.companies[0].order == "muster", "Muster button did not issue authoritative order"): return
	game.city_ui.overseer_modal.hide()
	if not _check(await _step(50), "Company march failed"): return
	game.focus = game._iso(27,18)
	game._set_view(1.35)
	await _capture("company-muster.png")
	await game._smoke_click_button(game.city_ui.overseers_button)
	await game._smoke_click_button(game.city_ui.overseer_tabs["Military"])
	await game._smoke_click_button(controls.raid)
	if not _check(await game._wait_idle() and game.state.military.invasion_active, "Test invasion did not enter the world"): return
	game.city_ui.overseer_modal.hide()
	if not _check(await _step(8), "Battle did not advance"): return
	await _capture("invaders-intercepted.png")
	game._queue_command("save")
	if not _check(await game._wait_idle(), "Battle save failed"): return
	var saved_hash := str(game.last_reply.hash)
	if not _check(await _step(8), "Battle step after save failed"): return
	game._queue_command("load")
	if not _check(await game._wait_idle() and str(game.last_reply.hash) == saved_hash, "Battle reload changed state"): return
	for i in range(30):
		if not game.state.military.invasion_active: break
		if not _check(await _step(4), "Battle progression failed"): return
	if not _check(int(game.state.military.defeated) == 4 and int(game.state.military.breaches) == 0, "Mustered company did not repel the invasion"): return
	await game._smoke_click_button(game.city_ui.overseers_button)
	await game._smoke_click_button(game.city_ui.overseer_tabs["Military"])
	await _capture("military-victory.png")
	await game._smoke_click_button(controls.recall)
	if not _check(await game._wait_idle(), "Recall command failed"): return
	game.city_ui.overseer_modal.hide()
	if not _check(await _step(80) and game.state.companies[0].stationed == game.state.companies[0].soldiers, "Infantry failed to return home"): return
	print("NILE_MILITARY_PASS copper weapons recruits muster invasion combat save reload recall ", JSON.stringify(game.state.military))
	quit(0)
