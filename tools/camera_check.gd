extends SceneTree

var failed := false
func _initialize() -> void:
	run.call_deferred()

func check(result: bool, description: String) -> void:
	if not result:
		printerr(description)
		failed = true

func run() -> void:
	var settings := root.get_node("CitySettings")
	settings.open_settings()
	await process_frame
	check(settings.is_open(), "settings failed to open")
	settings._listen_for_key("Rotate right")
	var event := InputEventKey.new()
	event.pressed = true
	event.keycode = KEY_Q
	settings._settings_input(event)
	check(settings.notice.text.contains("Already assigned"), "duplicate key accepted")
	event.keycode = KEY_T
	settings._settings_input(event)
	var disk := ConfigFile.new()
	check(disk.load(settings.SAVE_PATH) == OK and disk.get_value("camera", "Rotate right") == KEY_T, "binding not saved")
	settings.settings_window.hide()
	change_scene_to_file("res://scenes/sandbox.tscn")
	await scene_changed
	var game := current_scene
	for frame in range(200):
		await process_frame
		if not game.state.is_empty(): break
	check(not game.state.is_empty(), "sidecar not ready")
	var origin: Vector2 = game.camera.position
	game._queue_command("pause")
	await game._wait_idle()
	var snapshot_hash: String = str(game.last_reply.hash)
	game._set_tool("road")
	await game._click_cell(Vector2i(13, 12), false)
	check(game.painting_roads, "drag did not start")
	for i in range(4):
		root.push_input(event, true)
		await process_frame
		check(not game.painting_roads and game.road_preview.is_empty(), "turn left a stale road preview")
		var world: Vector2 = game._iso(17.5, 11.5)
		check(game._tile_at(world).is_equal_approx(Vector2(17.5, 11.5)), "projection round trip")
		check(game.view_turns == (i + 1) % 4, "turn ignored")
		for item: Dictionary in game.placements:
			var footprint: PackedVector2Array = game._footprint(item.rect)
			for corner: Vector2 in footprint:
				check(item.sprite.position.y >= corner.y, "wrong sprite corner")
		await process_frame
	check(game.camera.position.distance_to(origin) < 0.1, "camera drift after four turns")
	game._queue_command("state")
	await game._wait_idle()
	check(str(game.last_reply.hash) == snapshot_hash, "rotation changed clock or map")
	if not failed: print("NILE_CAMERA_PASS")
	quit(int(failed))
