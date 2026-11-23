extends RefCounted


static func run(game: Node) -> void:
	await game.get_tree().process_frame
	if not game._smoke_require(await game._wait_idle(), "Mining village did not open"): return
	game._queue_command("step", {"ticks": 1000})
	if not game._smoke_require(await game._wait_idle() and game.state.palace_active, "Palace did not become staffed"): return
	game.focus = game._iso(25, 7)
	game._set_view(0.85)
	await game._smoke_choose_tool("goldmine")
	await game._click_cell(Vector2i(24, 4))
	if not game._smoke_require(game.placement_notice.visible and "rock" in game.placement_notice.text, "Missing gold terrain feedback"): return
	await game._click_cell(Vector2i(28, 4))
	if not game._smoke_require(await game._wait_idle() and game._has_building_kind("goldmine"), "Could not build at the deposit"): return
	await game._drag_cells("road", Vector2i(22, 10), Vector2i(28, 10))
	if not game._smoke_require(await game._wait_idle(), "Mine approach road refused"): return
	await game._drag_cells("road", Vector2i(28, 10), Vector2i(28, 6))
	if not game._smoke_require(await game._wait_idle(), "Mine access road refused"): return
	await game._smoke_choose_tool("inspect")
	for i in 150:
		game._queue_command("step", {"ticks": 2})
		if not game._smoke_require(await game._wait_idle(), "Mine simulation failed"): return
		if game.state.gold_in_transit > 0: break
	if not game._smoke_require(game.state.gold_in_transit > 0 and game.state.gold_delivered == 0, "No physical gold shipment"): return
	var path := OS.get_environment("NILE_GOLD_CAPTURE")
	if not path.is_empty():
		game.get_window().size = Vector2i(1920, 1080)
		if not await game._save_capture(path.get_base_dir(), path.get_file(), Vector2i(1920, 1080)): return
	await game._smoke_click_button(game.save_button)
	if not game._smoke_require(await game._wait_idle(), "Could not save loaded gold cart"): return
	var saved: String = game.last_reply.hash
	game._queue_command("step", {"ticks": 100})
	if not game._smoke_require(await game._wait_idle() and game.state.gold_delivered > 0, "Gold did not arrive"): return
	var delivered: String = game.last_reply.hash
	await game._smoke_click_button(game.load_button)
	if not game._smoke_require(await game._wait_idle() and game.last_reply.hash == saved, "Gold save lost in-flight cargo"): return
	game._queue_command("step", {"ticks": 100})
	if not game._smoke_require(await game._wait_idle() and game.last_reply.hash == delivered, "Gold replay diverged"): return
	print("NILE_GOLD_PASS placement ore cart palace treasury save replay")
	game.sim.stop()
	game.get_tree().quit()
