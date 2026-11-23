extends "res://scripts/street_study.gd"
## Input and presentation only. The bundled Go process owns the settlement.

const Transport = preload("res://scripts/sidecar.gd")
const TICK_SECONDS := 0.2

var sim: Node
var state: Dictionary = {}
var last_reply: Dictionary = {}
var replies := 0
var current_operation := ""
var command_queue: Array[Dictionary] = []
var paused := true
var poll_elapsed := 0.0
var smoke_mode := false
var active_tool := "inspect"
var painting_roads := false
var last_painted := Vector2i(-100, -100)
var hover_cell := Vector2i(-100, -100)
var hover_valid := false
var ghost: Sprite2D
var road_nodes: Array[Node] = []
var road_signature := ""
var building_nodes: Dictionary = {}
var walker_nodes: Dictionary = {}
var walker_targets: Dictionary = {}
var texture_cache: Dictionary = {}
var tool_buttons: Dictionary = {}
var pause_button: Button
var save_button: Button
var load_button: Button
var stats: Label
var status: Label


func _ready() -> void:
	var args := OS.get_cmdline_user_args()
	if "--diagnostic" in args or "--smoke" in args:
		palette.free()
		get_tree().change_scene_to_file.call_deferred("res://scenes/neighborhood.tscn")
		return
	if "--street-smoke" in args or "--study" in args:
		palette.free()
		get_tree().change_scene_to_file.call_deferred("res://scenes/street_study.tscn")
		return
	get_tree().auto_accept_quit = false
	get_window().title = "Project Nile · Riverbank settlement"
	smoke_mode = "--sandbox-smoke" in args
	if not _load_data():
		get_tree().quit(1)
		return
	assets["well"] = {"id": "well", "path": "well.png", "footprint": [1, 1], "draw_width": 120, "anchor": [0.5, 0.99], "label": "Well"}
	assets["large_hut"] = {"id": "large_hut", "path": "large-hut.png", "footprint": [1, 1], "draw_width": 96, "anchor": [0.49, 0.87], "label": "Large Hut"}
	layout = {"roads": [], "terrain_polygons": [
		{"kind": "silt", "points": [[-64, -64], [10, -64], [9.5, 2], [10.8, 9], [9.5, 17], [10.2, 25], [9.5, 34], [10, 96], [-64, 96]]},
		{"kind": "grass", "points": [[-64, -64], [8, -64], [8.8, 3], [8, 8], [9, 13], [8.1, 19], [9, 25], [8.4, 32], [8, 96], [-64, 96]]},
		{"kind": "water", "points": [[-64, -64], [6, -64], [6, 96], [-64, 96]]}]}
	placements.clear()
	focus = _iso(16, 15)
	neighborhood_zoom = 1.1
	RenderingServer.set_default_clear_color(Color("c8b181"))
	_build_world()
	buildings.y_sort_enabled = true
	for at: Vector2 in [Vector2(7, 5), Vector2(8.2, 7), Vector2(7.2, 9.5), Vector2(8, 12), Vector2(7.5, 19.2), Vector2(8.5, 24), Vector2(6.8, 26.5), Vector2(7.6, 29), Vector2(25, 9), Vector2(26, 12), Vector2(24, 26)]:
		var palm := Sprite2D.new()
		_skin_sprite(palm, "palm", _iso(at.x, at.y), 90)
		buildings.add_child(palm)
	_build_hud()
	_set_view(neighborhood_zoom)
	ghost = Sprite2D.new()
	ghost.name = "PlacementGhost"
	ghost.z_index = 20
	ghost.visible = false
	buildings.add_child(ghost)
	sim = Transport.new()
	add_child(sim)
	sim.response_received.connect(_on_reply)
	sim.failed.connect(_on_failure)
	if sim.start("sandbox"):
		_queue_command("state")
		if not smoke_mode:
			_queue_command("resume")
	if smoke_mode:
		_sandbox_smoke.call_deferred()


func _build_hud() -> void:
	var layer := CanvasLayer.new()
	add_child(layer)
	hud = Control.new()
	hud.set_anchors_and_offsets_preset(Control.PRESET_FULL_RECT)
	hud.mouse_filter = Control.MOUSE_FILTER_IGNORE
	layer.add_child(hud)
	var ribbon := PanelContainer.new()
	ribbon.set_anchors_and_offsets_preset(Control.PRESET_TOP_WIDE)
	ribbon.offset_bottom = RIBBON_HEIGHT
	ribbon.add_theme_stylebox_override("panel", _style(Color("3b3023"), GOLD))
	hud.add_child(ribbon)
	var row := HBoxContainer.new()
	row.add_theme_constant_override("separation", 26)
	ribbon.add_child(row)
	row.add_child(_label("N I L E", 21, Color("f2db9f")))
	row.add_child(_label("RIVERBANK SETTLEMENT", 12))
	stats = _label("Starting local settlement…", 14)
	stats.size_flags_horizontal = Control.SIZE_EXPAND_FILL
	stats.horizontal_alignment = HORIZONTAL_ALIGNMENT_RIGHT
	row.add_child(stats)
	var paper := TextureRect.new()
	paper.texture = load(ASSET_ROOT + "sidebar.png")
	paper.expand_mode = TextureRect.EXPAND_IGNORE_SIZE
	paper.stretch_mode = TextureRect.STRETCH_SCALE
	paper.mouse_filter = Control.MOUSE_FILTER_IGNORE
	paper.set_anchors_and_offsets_preset(Control.PRESET_RIGHT_WIDE)
	paper.offset_left = -PANEL_WIDTH
	paper.offset_top = RIBBON_HEIGHT
	hud.add_child(paper)
	var panel := PanelContainer.new()
	panel.set_anchors_and_offsets_preset(Control.PRESET_RIGHT_WIDE)
	panel.offset_left = -PANEL_WIDTH
	panel.offset_top = RIBBON_HEIGHT
	var style := _style(Color.TRANSPARENT, Color.TRANSPARENT, 0)
	style.content_margin_left = 14
	style.content_margin_right = 14
	style.content_margin_top = 86
	style.content_margin_bottom = 42
	panel.add_theme_stylebox_override("panel", style)
	hud.add_child(panel)
	var column := VBoxContainer.new()
	column.add_theme_constant_override("separation", 6)
	panel.add_child(column)
	column.add_child(_label("BUILD", 12, INK))
	var grid := palette
	grid.columns = 2
	column.add_child(grid)
	for entry: Array in [["inspect", "Inspect"], ["road", "Road"], ["hut", "Housing"], ["well", "Well"]]:
		var tool := str(entry[0])
		var button := _button(str(entry[1]), _set_tool.bind(tool))
		button.toggle_mode = true
		button.custom_minimum_size.x = 86
		if tool in ["hut", "well"]:
			button.text = ""
			button.icon = _texture_for(tool)
			button.expand_icon = true
			button.icon_alignment = HORIZONTAL_ALIGNMENT_CENTER
			button.custom_minimum_size.y = 62
			button.tooltip_text = str(entry[1])
			button.accessibility_name = str(entry[1])
		tool_buttons[tool] = button
		grid.add_child(button)
	var clear_button := _button("Clear land", _set_tool.bind("clear"))
	clear_button.toggle_mode = true
	tool_buttons["clear"] = clear_button
	column.add_child(clear_button)
	tool_buttons.inspect.set_pressed_no_signal(true)
	title_label = _label("Begin a settlement", 18, INK)
	title_label.autowrap_mode = TextServer.AUTOWRAP_WORD_SMART
	column.add_child(title_label)
	kind_label = _label("Housing and water", 12, Color("735936"))
	kind_label.autowrap_mode = TextServer.AUTOWRAP_WORD_SMART
	column.add_child(kind_label)
	detail_label = _label("Connect housing to the arrival road. Add a nearby well, then watch people settle in.", 13, INK)
	detail_label.autowrap_mode = TextServer.AUTOWRAP_WORD_SMART
	column.add_child(detail_label)
	var spacer := Control.new()
	spacer.size_flags_vertical = Control.SIZE_EXPAND_FILL
	column.add_child(spacer)
	status = _label("Roads connect homes to arriving settlers.", 12, Color("735936"))
	status.autowrap_mode = TextServer.AUTOWRAP_WORD_SMART
	status.max_lines_visible = 4
	status.text_overrun_behavior = TextServer.OVERRUN_TRIM_ELLIPSIS
	column.add_child(status)
	pause_button = _button("Resume" if paused else "Pause", _toggle_pause)
	column.add_child(pause_button)
	var saves := HBoxContainer.new()
	column.add_child(saves)
	save_button = _button("Save", func() -> void: _queue_command("save"))
	load_button = _button("Load", _load_save)
	save_button.size_flags_horizontal = Control.SIZE_EXPAND_FILL
	load_button.size_flags_horizontal = Control.SIZE_EXPAND_FILL
	saves.add_child(save_button)
	saves.add_child(load_button)
	neighborhood_button = _button("Center village", func() -> void: _set_view(neighborhood_zoom))
	column.add_child(neighborhood_button)
	footprints_button = _button("Footprints", _toggle_footprints)
	footprints_button.toggle_mode = true
	column.add_child(footprints_button)
	column.add_child(_label("Drag to lay roads\nRight-drag to pan · wheel zoom\nSpace: pause · Esc: inspect", 11, Color("735936")))


func _queue_command(op: String, fields: Dictionary = {}) -> void:
	if sim.child.is_empty(): return
	command_queue.append({"op": op, "fields": fields})


func _load_save() -> void:
	if sim.child.is_empty():
		stats.text = "Starting local settlement…"
		if not sim.start("sandbox"): return
		_queue_command("state")
		if state.is_empty():
			_queue_command("resume")
			return
	_queue_command("pause")
	_queue_command("load")


func _toggle_pause() -> void:
	if sim.child.is_empty(): return
	pause_button.disabled = true
	_queue_command("resume" if paused else "pause")


func _process(delta: float) -> void:
	if sim == null or camera == null:
		return
	poll_elapsed += delta
	if sim.pending == 0 and not sim.child.is_empty():
		if not command_queue.is_empty():
			var command: Dictionary = command_queue.pop_front()
			current_operation = command.op
			sim.request(command.op, int(command.fields.get("ticks", 0)), command.fields)
		elif not paused and poll_elapsed >= 0.1:
			# Poll frequency affects only presentation. Go owns elapsed time and ticks.
			poll_elapsed = 0.0
			current_operation = "state"
			sim.request("state")
	for id: String in walker_nodes:
		var sprite := walker_nodes[id] as Sprite2D
		sprite.position = sprite.position.lerp(walker_targets[id], minf(delta / TICK_SECONDS, 1.0))
	_update_ghost()


func _on_reply(reply: Dictionary) -> void:
	if stats.text == "Starting local settlement…":
		status.text = "Settlement ready."
	status.tooltip_text = ""
	load_button.text = "Load"
	save_button.disabled = false
	for button: Button in tool_buttons.values(): button.disabled = false
	last_reply = reply
	replies += 1
	paused = not bool(reply.get("running", false))
	pause_button.text = "Resume" if paused else "Pause"
	if current_operation in ["resume", "pause"]:
		pause_button.disabled = false
	state = reply.get("state", {})
	_sync_world()
	stats.text = "Population %d   ·   Tick %d%s" % [int(state.get("population", 0)), int(state.get("tick", 0)), "   ·   Paused" if paused else ""]
	if not reply.get("ok", false):
		status.text = str(reply.get("error", "Command failed"))
	elif current_operation == "save":
		status.text = "Settlement saved."
	elif current_operation == "load":
		status.text = "Saved settlement restored."
	elif current_operation == "build":
		status.text = "Placed. Choose Inspect to see details."
	_refresh_inspector()


func _on_failure(reason: String) -> void:
	paused = true
	command_queue.clear()
	stats.text = "Settlement stopped"
	active_tool = "inspect"
	painting_roads = false
	status.text = reason.replace("\n", " ").replace("\r", " ").left(140)
	status.tooltip_text = reason
	load_button.text = "Retry"
	save_button.disabled = true
	for button: Button in tool_buttons.values(): button.disabled = true
	pause_button.text = "Resume"
	pause_button.disabled = true


func _texture_for(asset_id: String) -> Texture2D:
	if texture_cache.has(asset_id):
		return texture_cache[asset_id]
	var spec: Dictionary = assets[asset_id]
	var texture := load(_asset_path(spec.path)) as Texture2D
	if spec.has("region"):
		var atlas := AtlasTexture.new()
		atlas.atlas = texture
		var region: Array = spec.region
		atlas.region = Rect2(float(region[0]), float(region[1]), float(region[2]), float(region[3]))
		atlas.filter_clip = true
		texture = atlas
	texture_cache[asset_id] = texture
	return texture


func _skin_sprite(sprite: Sprite2D, asset_id: String, at: Vector2, width: float = 0.0) -> void:
	var spec: Dictionary = assets[asset_id]
	sprite.texture = _texture_for(asset_id)
	sprite.centered = false
	var anchor: Array = spec.get("anchor", [0.5, 1.0])
	sprite.offset = -sprite.texture.get_size() * Vector2(float(anchor[0]), float(anchor[1]))
	var unit := (width if width > 0 else float(spec.draw_width)) / sprite.texture.get_width()
	sprite.transform = Transform2D(Vector2(1, spec.get("ground_shear", 0.0)) * unit, Vector2(0, spec.get("vertical_scale", 1.0)) * unit, at)
	sprite.texture_filter = CanvasItem.TEXTURE_FILTER_LINEAR_WITH_MIPMAPS


func _sync_world() -> void:
	var roads: Array = state.get("roads", [])
	var signature := JSON.stringify(roads)
	if signature != road_signature:
		road_signature = signature
		for node: Node in road_nodes:
			node.free()
		road_nodes.clear()
		var cells: Dictionary = {}
		for tile: Dictionary in roads:
			cells[Vector2i(int(tile.x), int(tile.y))] = true
		for cell: Vector2i in cells:
			var edges := Vector4(float(not cells.has(cell + Vector2i.LEFT)), float(not cells.has(cell + Vector2i.RIGHT)), float(not cells.has(cell + Vector2i.UP)), float(not cells.has(cell + Vector2i.DOWN)))
			_add_terrain_polygon("road", _footprint(Rect2(Vector2(cell), Vector2.ONE)), 3, Vector2(cell), edges)
			road_nodes.append(ground.get_child(ground.get_child_count() - 1))
	var selected_id := str(selected.get("id", ""))
	placements.clear()
	selected = {}
	var seen: Dictionary = {}
	for building: Dictionary in state.get("buildings", []):
		var id := str(building.get("id", "%s:%s" % [building.x, building.y]))
		seen[id] = true
		var asset_id := "well" if building.kind == "well" else ("large_hut" if int(building.get("tier", 0)) > 0 else "hut")
		if not building_nodes.has(id):
			var new_sprite := Sprite2D.new()
			buildings.add_child(new_sprite)
			building_nodes[id] = new_sprite
		var sprite := building_nodes[id] as Sprite2D
		_skin_sprite(sprite, asset_id, _iso(float(building.x) + 1, float(building.y) + 1))
		sprite.modulate.a = 0.55 if building.kind == "hut" and int(building.get("pop", 0)) == 0 else 1.0
		var item := {"id": id, "asset": asset_id, "spec": assets[asset_id], "rect": Rect2(float(building.x), float(building.y), 1, 1), "sprite": sprite, "building": building}
		placements.append(item)
		if id == selected_id:
			selected = item
	for id: String in building_nodes.keys():
		if not seen.has(id):
			building_nodes[id].free()
			building_nodes.erase(id)
	placements.sort_custom(func(a: Dictionary, b: Dictionary) -> bool:
		var ad: float = a.rect.end.x + a.rect.end.y
		var bd: float = b.rect.end.x + b.rect.end.y
		return str(a.id) < str(b.id) if is_equal_approx(ad, bd) else ad < bd)
	seen.clear()
	for walker: Dictionary in state.get("walkers", []):
		var id := str(walker.id)
		seen[id] = true
		var target := _iso(float(walker.x) + 0.5, float(walker.y) + 0.5)
		if not walker_nodes.has(id):
			var sprite := Sprite2D.new()
			_skin_sprite(sprite, "resident", target)
			buildings.add_child(sprite)
			walker_nodes[id] = sprite
		walker_targets[id] = target
	for id: String in walker_nodes.keys():
		if not seen.has(id):
			walker_nodes[id].free()
			walker_nodes.erase(id)
			walker_targets.erase(id)
	overlay.queue_redraw()


func _refresh_inspector() -> void:
	if title_label == null:
		return
	if selected.is_empty():
		title_label.text = "Begin a settlement" if state.get("population", 0) == 0 else "Riverbank settlement"
		kind_label.text = "Housing and water"
		detail_label.text = "Connect housing to the arrival road. Add a nearby well, then watch people settle in."
	else:
		var building: Dictionary = selected.building
		if building.kind == "well":
			title_label.text = "Well"
			kind_label.text = "Local water access"
			detail_label.text = "Supplies housing within two tiles in each direction."
		else:
			title_label.text = str(building.get("tier_name", "Large Hut" if int(building.get("tier", 0)) > 0 else "Small Hut"))
			if int(building.get("pop", 0)) == 0:
				title_label.text = "Vacant housing"
			kind_label.text = "%d / %d residents" % [int(building.get("pop", 0)), int(building.get("capacity", 7 if int(building.get("tier", 0)) > 0 else 5))]
			detail_label.text = "Road: %s\nWater: %s" % ["connected" if building.get("connected", false) else "not connected", "available" if building.get("water", false) else "needed"]
			if not building.get("connected", false):
				detail_label.text += "\n\nConnect to the arrival road."
			elif not building.get("water", false):
				detail_label.text += "\n\nNeeds a nearby well."
			elif int(building.get("pop", 0)) >= int(building.get("capacity", 7)):
				detail_label.text += "\n\nHousehold is full."
			else:
				detail_label.text += "\n\nWaiting for immigrants."
	overlay.queue_redraw()


func _set_tool(tool: String) -> void:
	if sim != null and sim.child.is_empty(): return
	active_tool = tool
	painting_roads = false
	for key: String in tool_buttons:
		tool_buttons[key].set_pressed_no_signal(key == tool)
	status.text = {"inspect": "Click a building to inspect.", "road": "Drag across empty land to lay roads.", "hut": "Place housing beside a connected road.", "well": "Place a well near housing.", "clear": "Remove a road, empty home or well. Residents are protected."}[tool]
	_update_ghost()


func _can_place(cell: Vector2i) -> bool:
	if cell.x < 6 or cell.x >= 32 or cell.y < 0 or cell.y >= 32:
		return false
	for item: Dictionary in state.get("buildings", []):
		if int(item.x) == cell.x and int(item.y) == cell.y:
			return active_tool == "clear" and int(item.get("pop", 0)) == 0
	for tile: Dictionary in state.get("roads", []):
		if int(tile.x) == cell.x and int(tile.y) == cell.y:
			return active_tool == "clear" and cell != Vector2i(6, 15)
	return active_tool != "clear"


func _update_ghost() -> void:
	if ghost == null:
		return
	var mouse := get_viewport().get_mouse_position()
	var over_map := mouse.x < get_viewport_rect().size.x - PANEL_WIDTH and mouse.y > RIBBON_HEIGHT
	var tile := _tile_at(get_canvas_transform().affine_inverse() * mouse)
	hover_cell = Vector2i(floori(tile.x), floori(tile.y))
	hover_valid = _can_place(hover_cell)
	ghost.visible = active_tool in ["hut", "well"] and over_map
	if ghost.visible:
		_skin_sprite(ghost, active_tool, _iso(hover_cell.x + 1, hover_cell.y + 1))
		ghost.modulate = Color(0.75, 1, 0.65, 0.6) if hover_valid else Color(1, 0.35, 0.25, 0.55)
	overlay.queue_redraw()


func _draw_overlay() -> void:
	super._draw_overlay()
	if active_tool == "inspect" or hud == null:
		return
	var mouse := get_viewport().get_mouse_position()
	if mouse.x >= get_viewport_rect().size.x - PANEL_WIDTH or mouse.y < RIBBON_HEIGHT:
		return
	var points := _footprint(Rect2(Vector2(hover_cell), Vector2.ONE))
	overlay.draw_colored_polygon(points, Color(0.7, 0.9, 0.45, 0.3) if hover_valid else Color(0.9, 0.2, 0.1, 0.3))
	points.append(points[0])
	overlay.draw_polyline(points, Color("e5d58d") if hover_valid else Color("e7755d"), 2, true)


func _place_cell(cell: Vector2i) -> void:
	if not _can_place(cell):
		return
	_queue_command("build", {"kind": active_tool, "x": cell.x, "y": cell.y})


func _paint_to(cell: Vector2i) -> void:
	if last_painted.x == -100:
		_place_cell(cell)
	else:
		var cursor := last_painted
		while cursor.x != cell.x:
			cursor.x += signi(cell.x - cursor.x)
			_place_cell(cursor)
		while cursor.y != cell.y:
			cursor.y += signi(cell.y - cursor.y)
			_place_cell(cursor)
	last_painted = cell


func _input(event: InputEvent) -> void:
	super._input(event)
	if event is InputEventMouseButton and event.button_index == MOUSE_BUTTON_LEFT and not event.pressed:
		painting_roads = false


func _unhandled_input(event: InputEvent) -> void:
	if event is InputEventKey and event.pressed and not event.echo:
		if event.keycode == KEY_SPACE:
			_toggle_pause()
			return
		if event.keycode == KEY_ESCAPE:
			_set_tool("inspect")
			selected = {}
			_refresh_inspector()
			return
	if event is InputEventMouseButton and event.button_index == MOUSE_BUTTON_LEFT and event.pressed and active_tool != "inspect":
		var tile := _tile_at(get_canvas_transform().affine_inverse() * event.position)
		var cell := Vector2i(floori(tile.x), floori(tile.y))
		if active_tool == "road":
			painting_roads = true
			last_painted = Vector2i(-100, -100)
			_paint_to(cell)
		else:
			_place_cell(cell)
		get_viewport().set_input_as_handled()
		return
	if event is InputEventMouseMotion and painting_roads:
		var tile := _tile_at(get_canvas_transform().affine_inverse() * event.position)
		_paint_to(Vector2i(floori(tile.x), floori(tile.y)))
		return
	super._unhandled_input(event)


func _notification(what: int) -> void:
	if what == NOTIFICATION_WM_CLOSE_REQUEST:
		if sim != null:
			sim.stop()
		get_tree().quit()


func _wait_idle() -> bool:
	var deadline := Time.get_ticks_msec() + 10000
	while (sim.pending != 0 or not command_queue.is_empty()) and Time.get_ticks_msec() < deadline:
		await get_tree().process_frame
	return sim.pending == 0 and command_queue.is_empty() and bool(last_reply.get("ok", false))


func _click_cell(cell: Vector2i, release: bool = true) -> void:
	var event := InputEventMouseButton.new()
	event.button_index = MOUSE_BUTTON_LEFT
	event.pressed = true
	event.position = get_canvas_transform() * _iso(cell.x + 0.5, cell.y + 0.5)
	get_viewport().push_input(event, true)
	await get_tree().process_frame
	if release:
		event.pressed = false
		get_viewport().push_input(event, true)
		await get_tree().process_frame


func _sandbox_smoke() -> void:
	await get_tree().process_frame
	if not _smoke_require(await _wait_idle(), "sandbox initial state failed"):
		return
	await _smoke_click_button(tool_buttons.road)
	await _click_cell(Vector2i(11, 15), false)
	var drag := InputEventMouseMotion.new()
	drag.position = get_canvas_transform() * _iso(13.5, 15.5)
	drag.relative = Vector2(96, 48) * camera.zoom
	get_viewport().push_input(drag, true)
	await get_tree().process_frame
	var release := InputEventMouseButton.new()
	release.button_index = MOUSE_BUTTON_LEFT
	release.position = drag.position
	get_viewport().push_input(release, true)
	if not _smoke_require(await _wait_idle(), "road drag failed"):
		return
	await _smoke_click_button(tool_buttons.clear)
	await _click_cell(Vector2i(13, 15))
	if not _smoke_require(await _wait_idle(), "clear road failed"):
		return
	var road_removed := true
	for tile: Dictionary in state.roads:
		if int(tile.x) == 13 and int(tile.y) == 15:
			road_removed = false
	if not _smoke_require(road_removed, "clear left the road in authoritative state"):
		return
	await _smoke_click_button(tool_buttons.road)
	await _click_cell(Vector2i(13, 15))
	if not _smoke_require(await _wait_idle(), "road replacement failed"):
		return
	await _smoke_click_button(tool_buttons.hut)
	await _click_cell(Vector2i(12, 14))
	if not _smoke_require(await _wait_idle(), "housing placement failed"):
		return
	await _smoke_click_button(tool_buttons.well)
	await _click_cell(Vector2i(13, 14))
	if not _smoke_require(await _wait_idle(), "well placement failed"):
		return
	_queue_command("step", {"ticks": 8})
	if not _smoke_require(await _wait_idle(), "migration step failed"):
		return
	var previous_walkers := JSON.stringify(state.get("walkers", []))
	_queue_command("step", {"ticks": 2})
	if not _smoke_require(await _wait_idle() and JSON.stringify(state.get("walkers", [])) != previous_walkers, "migrants did not move on roads"):
		return
	_queue_command("step", {"ticks": 400})
	if not _smoke_require(await _wait_idle(), "settlement progression failed"):
		return
	var house_ok := false
	for building: Dictionary in state.buildings:
		if int(building.x) == 12 and int(building.y) == 14:
			house_ok = int(building.get("pop", 0)) > 0 and int(building.get("tier", 0)) == 1 and bool(building.get("water", false))
	if not _smoke_require(house_ok, "occupied watered housing did not reach Large Hut"):
		return
	await _smoke_click_button(save_button)
	if not _smoke_require(await _wait_idle(), "SQLite save failed"):
		return
	var saved := state.duplicate(true)
	_queue_command("step", {"ticks": 17})
	if not _smoke_require(await _wait_idle(), "post-save step failed"):
		return
	await _smoke_click_button(load_button)
	if not _smoke_require(await _wait_idle() and state == saved, "SQLite restore differs from saved state"):
		return
	await _smoke_click_button(tool_buttons.inspect)
	await _click_cell(Vector2i(12, 14))
	if not _smoke_require(not selected.is_empty() and title_label.text == "Large Hut", "inspector did not select evolved housing"):
		return
	await _smoke_click_button(pause_button)
	if not _smoke_require(await _wait_idle() and not paused, "resume control failed"):
		return
	await _smoke_click_button(pause_button)
	if not _smoke_require(await _wait_idle() and paused, "pause control failed"):
		return
	var capture := OS.get_environment("NILE_SANDBOX_CAPTURE")
	if not capture.is_empty():
		if not _smoke_require(DisplayServer.get_name() != "headless", "sandbox capture requires a display"):
			return
		DirAccess.make_dir_recursive_absolute(capture.get_base_dir())
		get_window().size = Vector2i(1920, 1080)
		if not await _save_capture(capture.get_base_dir(), capture.get_file(), Vector2i(1920, 1080)):
			return
	print("NILE_SANDBOX_PASS road-drag clear housing well migration evolution inspect pause sqlite-save-load")
	sim.stop()
	get_tree().quit(0)
