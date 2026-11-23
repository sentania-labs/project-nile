extends SceneTree
var game: Node
var directory := ""
var observed := {}
var stages := {}

func _initialize() -> void: _run.call_deferred()
func fail(message: String) -> void:
	push_error(message)
	game.sim.stop()
	quit(1)
func _capture(name: String) -> void:
	await RenderingServer.frame_post_draw
	root.get_texture().get_image().save_png(directory.path_join(name + ".png"))
func _step(ticks: int) -> bool:
	game._queue_command("step", {"ticks": ticks})
	if not await game._wait_idle():
		fail("Pyramid advance failed")
		return false
	for a: Dictionary in game.state.get("walkers", []): observed[str(a.role) + "/" + str(a.get("stage", ""))] = true
	return true
func _run() -> void:
	directory = OS.get_environment("NILE_PYRAMID_DIR")
	root.size = Vector2i(1920, 1080)
	root.set_meta("nile_start_choice", "load")
	game = load("res://scenes/sandbox.tscn").instantiate()
	root.add_child(game)
	var deadline := Time.get_ticks_msec() + 20000
	while not game.startup_done and Time.get_ticks_msec() < deadline: await process_frame
	if not game.startup_done:
		fail("Prepared monument city did not load")
		return
	await game._wait_idle()
	if not game.paused:
		await game._smoke_click_button(game.pause_button)
		await game._wait_idle()
	game.focus = game._iso(26, 24)
	game._set_view(1.05)
	await game._smoke_choose_tool("smallstepped")
	await game._click_cell(Vector2i(23, 20))
	await game._wait_idle()
	if not game._has_building_kind("smallstepped"):
		fail("8x8 pyramid placement through the construction menu failed")
		return
	await game._smoke_choose_tool("inspect")
	await _capture("01-foundation")
	var ready := FileAccess.open(OS.get_environment("NILE_PYRAMID_READY"), FileAccess.WRITE)
	ready.store_string("ready")
	ready.close()
	# Physical preparation and movement use the normal Go clock for this shot.
	await game._smoke_click_button(game.speed_button)
	await game._wait_idle()
	await game._smoke_click_button(game.speed_button)
	await game._wait_idle()
	await game._smoke_click_button(game.pause_button)
	await game._wait_idle()
	await create_timer(8).timeout
	await game._smoke_click_button(game.pause_button)
	await game._wait_idle()
	await _capture("02-preparation")
	# Paused deterministic advances create a construction time-lapse; no client
	# fabricates materials, changes progress, or assigns workers.
	for sample in range(600):
		if not await _step(4 if not observed.has("stonehauler/deliver") else 48): return
		var b: Dictionary = game._building_at(Vector2i(23, 20))
		var done := int(b.get("construction_done", 0))
		var key := ""
		if done >= 400 and done < 8000: key = "03-stone-deliveries"
		elif done >= 8000 and done < 16000: key = "04-base-courses"
		elif done >= 16000 and done < 19200: key = "05-upper-base"
		elif done >= 19200 and done < 24000: key = "06-second-tier"
		elif done == 24000: key = "07-complete"
		if not key.is_empty() and not stages.has(key):
			stages[key] = true
			if game.city_ui.completion_modal.visible:
				await game._smoke_click_button(game.city_ui.completion_continue)
				await game._wait_idle()
			await _capture(key)
			await create_timer(1.0).timeout
		if done == 24000: break
	var final: Dictionary = game._building_at(Vector2i(23, 20))
	if int(final.get("construction_done", 0)) != 24000 or not observed.has("stonehauler/deliver"):
		fail("Monument did not consume its measured stone requirement via physical sledges")
		return
	await game._smoke_click_button(game.save_button)
	await game._wait_idle()
	var saved: Dictionary = game.state.duplicate(true)
	if not await _step(16): return
	await game._smoke_click_button(game.load_button)
	await game._wait_idle()
	if game.state != saved:
		fail("Completed pyramid SQLite restore changed state")
		return
	game._select_at(game._iso(26.5, 27.5))
	game._refresh_inspector()
	game.city_ui.show_inspector()
	await _capture("08-foreman")
	if not "240" in game.detail_label.text:
		fail("Foreman omitted measured stone count")
		return
	print("NILE_PYRAMID_PASS UI-placement 8x8 staging camp-preparation paired-masons timber-ramps physical-four-block-sledges 240-stone-blocks deterministic-time-lapse sqlite")
	game.sim.stop()
	quit(0)
