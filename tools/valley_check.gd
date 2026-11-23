extends SceneTree


func _initialize() -> void:
	run.call_deferred()


func run() -> void:
	root.set_meta("nile_start_choice","valley")
	change_scene_to_file("res://scenes/sandbox.tscn")
	await scene_changed
	var game := current_scene
	var until := Time.get_ticks_msec()+12000
	while (not game.startup_done or game.state.get("map_extent",0)!=96) and Time.get_ticks_msec()<until:
		await process_frame
	if game.state.get("map_extent")!=96: printerr("Valley did not open"); quit(1); return
	game._queue_command("pause")
	await game._wait_idle()
	var road_x := -1
	for tile: Dictionary in game.state.kingdom_road:
		if int(tile.y)==78: road_x=int(tile.x)
	game._navigate_minimap(Vector2(road_x,78))
	game._set_tool("hut")
	var bank := int(game.state.banks[78])
	if game._can_place(Vector2i(bank-1,78)): printerr("Allowed submerged house"); quit(1); return
	var at := Vector2i(road_x+1,78)
	await game._drag_cells("hut",at,at)
	if not await game._wait_idle() or game._building_at(at).is_empty(): printerr("Could not zone upstream"); quit(1); return
	game._queue_command("step",{"ticks":300})
	if not await game._wait_idle() or int(game.state.population)<1: printerr("Immigrants did not arrive upstream"); quit(1); return
	var path := OS.get_environment("NILE_VALLEY_CAPTURE")
	if not path.is_empty():
		game.get_window().size=Vector2i(1920,1080)
		game._navigate_minimap(Vector2(bank,63))
		game._set_view(0.44)
		await game._save_capture(path,"valley-upstream.png",Vector2i(1920,1080))
	game._turn_city(1)
	if not game._tile_at(game._iso(at.x,at.y)).is_equal_approx(Vector2(at)): printerr("Upstream rotation mismatch"); quit(1); return
	print("NILE_VALLEY_PASS 96x96 winding river dry through-road upstream housing migration rotation")
	game.sim.stop()
	quit()
