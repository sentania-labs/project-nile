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
var zoning := false
var zone_start := Vector2i.ZERO
var blocked_tiles: Dictionary = {}
var vacant_plots: Array[Vector2i] = []
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
var walker_data: Dictionary = {}
var landscape_nodes: Array[Node2D] = []
var texture_cache: Dictionary = {}
var tool_buttons: Dictionary = {}
var pause_button: Button
var speed_button: Button
var simulation_speed := 1
var save_button: Button
var load_button: Button
var stats: Label
var status: Label
var production_bar: ProgressBar


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
	assets["small_shanty"] = {"id": "small_shanty", "path": "small-shanty.png", "footprint": [1, 1], "draw_width": 148, "anchor": [0.49, 0.98], "label": "Small Shanty"}
	assets["farm"] = {"id": "farm", "path": "farm.png", "footprint": [3, 3], "draw_width": 320, "anchor": [0.49, 0.93], "vertical_scale": 0.77, "label": "Grain farm"}
	assets["cart"] = {"id": "cart", "path": "cart.png", "footprint": [0.2, 0.2], "draw_width": 80, "anchor": [0.58, 0.96], "label": "Grain cart"}
	assets["bazaar"]["footprint"] = [2, 2]
	assets["bazaar"]["draw_width"] = 190
	for frame in range(4):
		var spec: Dictionary = assets.resident.duplicate(true)
		spec.region = [frame * 543, 0, 543, 724]
		assets["resident_%d" % frame] = spec
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
		palm.set_meta("tile", Vector2i(at.floor()))
		buildings.add_child(palm)
		landscape_nodes.append(palm)
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
	var commands := HBoxContainer.new()
	column.add_child(commands)
	var grid := palette
	grid.columns = 3
	column.add_child(grid)
	for entry: Array in [["inspect", "Inspect"], ["road", "Road"], ["hut", "Housing"], ["well", "Well"], ["farm", "Grain farm"], ["granary", "Granary"], ["bazaar", "Bazaar"], ["clear", "Clear"]]:
		var tool := str(entry[0])
		var button := _button(str(entry[1]), _set_tool.bind(tool))
		button.toggle_mode = true
		button.custom_minimum_size.x = 86 if tool in ["inspect", "road"] else 56
		if tool in ["hut", "well", "farm", "granary", "bazaar"]:
			button.text = ""
			button.icon = _texture_for(tool)
			button.expand_icon = true
			button.icon_alignment = HORIZONTAL_ALIGNMENT_CENTER
			button.custom_minimum_size.y = 52
			button.tooltip_text = str(entry[1])
			button.accessibility_name = str(entry[1])
		tool_buttons[tool] = button
		if tool in ["inspect", "road"]:
			commands.add_child(button)
		else:
			grid.add_child(button)
	tool_buttons.inspect.set_pressed_no_signal(true)
	title_label = _label("Feed the neighborhood", 18, INK)
	title_label.autowrap_mode = TextServer.AUTOWRAP_WORD_SMART
	column.add_child(title_label)
	kind_label = _label("Your first harvest is near", 12, Color("735936"))
	kind_label.autowrap_mode = TextServer.AUTOWRAP_WORD_SMART
	column.add_child(kind_label)
	detail_label = _label("Follow grain from the farm to the granary and bazaar. Extend the roads and mark housing to grow your neighborhood.", 13, INK)
	detail_label.autowrap_mode = TextServer.AUTOWRAP_WORD_SMART
	detail_label.max_lines_visible = 9
	detail_label.text_overrun_behavior = TextServer.OVERRUN_TRIM_ELLIPSIS
	column.add_child(detail_label)
	production_bar = ProgressBar.new()
	production_bar.custom_minimum_size.y = 13
	production_bar.show_percentage = false
	production_bar.add_theme_stylebox_override("background", _style(Color("85734e"), Color("85734e"), 0))
	production_bar.add_theme_stylebox_override("fill", _style(Color("a9ab54"), Color("a9ab54"), 0))
	production_bar.visible = false
	column.add_child(production_bar)
	var spacer := Control.new()
	spacer.size_flags_vertical = Control.SIZE_EXPAND_FILL
	column.add_child(spacer)
	status = _label("Roads connect homes to arriving settlers.", 12, Color("735936"))
	status.autowrap_mode = TextServer.AUTOWRAP_WORD_SMART
	status.max_lines_visible = 4
	status.text_overrun_behavior = TextServer.OVERRUN_TRIM_ELLIPSIS
	column.add_child(status)
	pause_button = _button("Resume" if paused else "Pause", _toggle_pause)
	var time_controls := HBoxContainer.new()
	column.add_child(time_controls)
	pause_button.size_flags_horizontal = Control.SIZE_EXPAND_FILL
	time_controls.add_child(pause_button)
	speed_button = _button("1×", _cycle_speed)
	speed_button.custom_minimum_size.x = 56
	time_controls.add_child(speed_button)
	var saves := HBoxContainer.new()
	column.add_child(saves)
	save_button = _button("Save", func() -> void: _queue_command("save"))
	load_button = _button("Load", _load_save)
	save_button.size_flags_horizontal = Control.SIZE_EXPAND_FILL
	load_button.size_flags_horizontal = Control.SIZE_EXPAND_FILL
	saves.add_child(save_button)
	saves.add_child(load_button)
	var view_controls := HBoxContainer.new()
	column.add_child(view_controls)
	neighborhood_button = _button("Center", func() -> void: _set_view(neighborhood_zoom))
	neighborhood_button.size_flags_horizontal = Control.SIZE_EXPAND_FILL
	view_controls.add_child(neighborhood_button)
	footprints_button = _button("Plots", _toggle_footprints)
	footprints_button.toggle_mode = true
	footprints_button.size_flags_horizontal = Control.SIZE_EXPAND_FILL
	view_controls.add_child(footprints_button)
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


func _cycle_speed() -> void:
	if sim.child.is_empty(): return
	speed_button.disabled = true
	_queue_command("set_speed", {"speed": 1 if simulation_speed == 4 else simulation_speed * 2})


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
		var target: Vector2 = walker_targets[id]
		var moving := sprite.position.distance_to(target) > 0.4
		if moving:
			sprite.flip_h = target.x < sprite.position.x
			if str(walker_data.get(id, {}).get("role", "")) != "cart":
				sprite.texture = _texture_for("resident_%d" % (int(Time.get_ticks_msec() / 140) % 4))
		sprite.position = sprite.position.lerp(target, minf(delta * simulation_speed / TICK_SECONDS, 1.0))
	_update_ghost()


func _on_reply(reply: Dictionary) -> void:
	if stats.text == "Starting local settlement…":
		status.text = "Settlement ready."
	status.tooltip_text = ""
	load_button.text = "Load"
	save_button.disabled = false
	speed_button.disabled = false
	for button: Button in tool_buttons.values(): button.disabled = false
	last_reply = reply
	replies += 1
	paused = not bool(reply.get("running", false))
	simulation_speed = int(reply.get("speed", 1))
	speed_button.text = "%d×" % simulation_speed
	if current_operation == "set_speed": speed_button.disabled = false
	pause_button.text = "Resume" if paused else "Pause"
	if current_operation in ["resume", "pause"]:
		pause_button.disabled = false
	state = reply.get("state", {})
	_sync_world()
	stats.text = "Population %d   ·   Grain %d   ·   Workers %d/%d   ·   %s, year %d%s" % [int(state.get("population", 0)), int(state.get("grain", 0)), int(state.get("employed", 0)), int(state.get("workforce", 0)), str(state.get("season", "Growing")).capitalize(), int(state.get("year", 1)), "   ·   Paused" if paused else ""]
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
	zoning = false
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
	speed_button.disabled = true


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


func _building_size(kind: String, data: Dictionary = {}) -> Vector2i:
	if data.has("width") and data.has("height"):
		return Vector2i(int(data.width), int(data.height))
	if assets.has(kind):
		var size: Array = assets[kind].get("footprint", [1, 1])
		return Vector2i(int(size[0]), int(size[1]))
	return Vector2i.ONE


func _building_rect(data: Dictionary) -> Rect2:
	return Rect2(Vector2(float(data.x), float(data.y)), Vector2(_building_size(str(data.kind), data)))


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
	blocked_tiles.clear()
	vacant_plots.clear()
	for tile: Dictionary in roads:
		blocked_tiles[Vector2i(int(tile.x), int(tile.y))] = true
	var selected_id := str(selected.get("id", ""))
	placements.clear()
	selected = {}
	var seen: Dictionary = {}
	for building: Dictionary in state.get("buildings", []):
		var id := str(building.get("id", "%s:%s" % [building.x, building.y]))
		seen[id] = true
		var cell := Vector2i(int(building.x), int(building.y))
		var rect := _building_rect(building)
		for y in range(int(rect.position.y), int(rect.end.y)):
			for x in range(int(rect.position.x), int(rect.end.x)):
				blocked_tiles[Vector2i(x, y)] = true
		var vacant: bool = building.kind == "hut" and int(building.pop) == 0
		if vacant: vacant_plots.append(cell)
		var asset_id := str(building.kind)
		if asset_id == "hut":
			asset_id = ["hut", "large_hut", "small_shanty"][clampi(int(building.get("tier", 0)), 0, 2)]
		if not building_nodes.has(id):
			var new_sprite := Sprite2D.new()
			buildings.add_child(new_sprite)
			building_nodes[id] = new_sprite
		var sprite := building_nodes[id] as Sprite2D
		sprite.visible = not vacant
		_skin_sprite(sprite, asset_id, _iso(rect.end.x, rect.end.y))
		sprite.modulate.a = 0.55 if building.kind == "hut" and int(building.get("pop", 0)) == 0 else 1.0
		var item := {"id": id, "asset": asset_id, "spec": assets[asset_id], "rect": rect, "sprite": sprite, "building": building}
		placements.append(item)
		if id == selected_id:
			selected = item
	for id: String in building_nodes.keys():
		if not seen.has(id):
			building_nodes[id].free()
			building_nodes.erase(id)
	for palm: Node2D in landscape_nodes:
		palm.visible = not blocked_tiles.has(palm.get_meta("tile"))
	placements.sort_custom(func(a: Dictionary, b: Dictionary) -> bool:
		var ad: float = a.rect.end.x + a.rect.end.y
		var bd: float = b.rect.end.x + b.rect.end.y
		return str(a.id) < str(b.id) if is_equal_approx(ad, bd) else ad < bd)
	seen.clear()
	walker_data.clear()
	for walker: Dictionary in state.get("walkers", []):
		var id := str(walker.id)
		seen[id] = true
		walker_data[id] = walker
		var target := _iso(float(walker.x) + 0.5, float(walker.y) + 0.5)
		if not walker_nodes.has(id):
			var sprite := Sprite2D.new()
			_skin_sprite(sprite, "cart" if walker.role == "cart" else "resident", target)
			buildings.add_child(sprite)
			walker_nodes[id] = sprite
		walker_targets[id] = target
		if selected_id == "walker:" + id:
			selected = _walker_selection(id)
	for id: String in walker_nodes.keys():
		if not seen.has(id):
			walker_nodes[id].free()
			walker_nodes.erase(id)
			walker_targets.erase(id)
	overlay.queue_redraw()


func _walker_selection(id: String) -> Dictionary:
	var data: Dictionary = walker_data[id]
	return {"id": "walker:" + id, "walker": data, "sprite": walker_nodes[id], "rect": Rect2(float(data.x), float(data.y), 1, 1)}


func _select_at(world_point: Vector2) -> void:
	super._select_at(world_point)
	var best_depth := float(selected.sprite.position.y) if not selected.is_empty() else -INF
	for id: String in walker_nodes:
		var sprite := walker_nodes[id] as Sprite2D
		var point := sprite.to_local(world_point)
		if sprite.position.y >= best_depth and sprite.get_rect().has_point(point) and sprite.is_pixel_opaque(point):
			best_depth = sprite.position.y
			selected = _walker_selection(id)
	_refresh_inspector()


func _refresh_inspector() -> void:
	if title_label == null:
		return
	production_bar.visible = false
	if selected.is_empty():
		var fed_homes := 0
		for home: Dictionary in state.get("buildings", []):
			if home.kind == "hut" and int(home.pop) > 0 and bool(home.get("food", false)):
				fed_homes += 1
		title_label.text = "Feed six households" if fed_homes < 6 else "A growing neighborhood"
		kind_label.text = "%d homes supplied with food" % fed_homes
		if int(state.get("grain_produced", 0)) == 0:
			detail_label.text = "The first harvest is near. Watch the farm cart carry grain to the granary, then follow the bazaar's buyers and sellers."
		elif fed_homes < 6:
			detail_label.text = "Extend the roads and drag out housing plots. New families need water and a bazaar seller passing their street."
		else:
			detail_label.text = "Keep grain reserves through the next flood. Add floodplain farms as your neighborhood grows."
	elif selected.has("walker"):
		var walker: Dictionary = selected.walker
		var role := str(walker.get("role", "immigrant"))
		title_label.text = {"immigrant": "Arriving resident", "cart": "Grain cart", "buyer": "Bazaar buyer", "seller": "Bazaar seller"}.get(role, role.capitalize())
		kind_label.text = "%d grain in transit" % int(walker.get("cargo", 0)) if role != "immigrant" else "Moving into the neighborhood"
		detail_label.text = {"immigrant": "Following connected roads to a home with room.", "cart": "Hauling the farm's harvest to a granary.", "buyer": "Collecting stored grain and carrying it back to the bazaar.", "seller": "Walking local roads and delivering food to nearby homes."}.get(role, "Following the road network.")
	else:
		var building: Dictionary = selected.building
		if building.kind == "well":
			title_label.text = "Well"
			kind_label.text = "Local water access"
			detail_label.text = "Supplies housing within two tiles in each direction."
		elif building.kind in ["farm", "granary", "bazaar"]:
			title_label.text = {"farm": "Grain farm", "granary": "Granary", "bazaar": "Bazaar"}[building.kind]
			kind_label.text = "%d / %d workers" % [int(building.get("workers", 0)), int(building.get("workers_needed", 0))]
			detail_label.text = "Grain: %d / %d\nRoad: %s" % [int(building.get("stock", 0)), int(building.get("stock_capacity", 0)), "connected" if building.get("connected", false) else "not connected"]
			if building.kind == "farm":
				var seconds := ceili(float(state.get("harvest_in", 0)) * TICK_SECONDS / simulation_speed)
				detail_label.text += "\nHarvest in %ds at %d×" % [seconds, simulation_speed]
				production_bar.visible = true
				production_bar.max_value = maxf(float(building.get("work_capacity", 1)), 1)
				production_bar.value = float(building.get("work", 0))
				production_bar.tooltip_text = "Preparation for the next harvest"
			elif building.kind == "granary":
				detail_label.text += "\n\nFarm carts deliver here. Bazaar buyers collect grain for their stalls."
			else:
				detail_label.text += "\n\nBuyers fetch grain. Sellers carry it along local streets to homes."
			if not building.get("connected", false):
				detail_label.text += "\nConnect to the road network."
			elif int(building.get("workers", 0)) == 0:
				detail_label.text += "\nMore residents are needed to work here."
		else:
			title_label.text = str(building.get("tier_name", "Large Hut" if int(building.get("tier", 0)) > 0 else "Small Hut"))
			if int(building.get("pop", 0)) == 0:
				title_label.text = "Vacant housing"
			kind_label.text = "%d / %d residents" % [int(building.get("pop", 0)), int(building.get("capacity", 7 if int(building.get("tier", 0)) > 0 else 5))]
			detail_label.text = "Road: %s\nWater: %s\nFood: %d grain\nBazaar: %s" % ["connected" if building.get("connected", false) else "not connected", "available" if building.get("water", false) else "needed", int(building.get("stock", 0)), "recent visit" if building.get("market", false) else "needs a seller"]
			if not building.get("connected", false):
				detail_label.text += "\n\nConnect to the arrival road."
			elif not building.get("water", false):
				detail_label.text += "\n\nNeeds a nearby well."
			elif not building.get("food", false) or not building.get("market", false):
				detail_label.text += "\n\nNeeds food delivered by a bazaar seller."
			elif int(building.get("pop", 0)) >= int(building.get("capacity", 7)):
				detail_label.text += "\n\nHousehold is full."
			else:
				detail_label.text += "\n\nWaiting for immigrants."
	overlay.queue_redraw()


func _set_tool(tool: String) -> void:
	if sim != null and sim.child.is_empty(): return
	zoning = false
	active_tool = tool
	painting_roads = false
	for key: String in tool_buttons:
		tool_buttons[key].set_pressed_no_signal(key == tool)
	status.text = {"inspect": "Inspect buildings or follow a walker.", "road": "Drag across empty land to lay roads.", "hut": "Drag to mark housing plots beside a road.", "well": "Place a well near housing.", "farm": "Place a 3×3 farm on the dark floodplain beside a road.", "granary": "Place a 4×4 granary beside a connected road.", "bazaar": "Place a 2×2 bazaar on roads serving your homes.", "clear": "Remove empty structures or roads. Residents and goods are protected."}[tool]
	_update_ghost()


func _can_place(cell: Vector2i) -> bool:
	if cell.x < 6 or cell.x >= 32 or cell.y < 0 or cell.y >= 32:
		return false
	if active_tool == "clear":
		for item: Dictionary in state.get("buildings", []):
			if _building_rect(item).has_point(Vector2(cell)):
				return int(item.get("pop", 0)) == 0 and int(item.get("stock", 0)) == 0
		return blocked_tiles.has(cell) and cell != Vector2i(6, 15)
	var size := _building_size(active_tool)
	if cell.x + size.x > 32 or cell.y + size.y > 32:
		return false
	if active_tool == "farm" and cell.x + size.x > 10:
		return false
	for y in range(cell.y, cell.y + size.y):
		for x in range(cell.x, cell.x + size.x):
			if blocked_tiles.has(Vector2i(x, y)):
				return false
	return true


func _preview_rect() -> Rect2:
	if active_tool == "clear":
		for item: Dictionary in state.get("buildings", []):
			var rect := _building_rect(item)
			if rect.has_point(Vector2(hover_cell)):
				return rect
	return Rect2(Vector2(hover_cell), Vector2(_building_size(active_tool)))


func _update_ghost() -> void:
	if ghost == null:
		return
	var mouse := get_viewport().get_mouse_position()
	var over_map := mouse.x < get_viewport_rect().size.x - PANEL_WIDTH and mouse.y > RIBBON_HEIGHT
	var tile := _tile_at(get_canvas_transform().affine_inverse() * mouse)
	hover_cell = Vector2i(floori(tile.x), floori(tile.y))
	hover_valid = _can_place(hover_cell)
	ghost.visible = active_tool in ["well", "farm", "granary", "bazaar"] and over_map
	if ghost.visible:
		var rect := _preview_rect()
		_skin_sprite(ghost, active_tool, _iso(rect.end.x, rect.end.y))
		ghost.modulate = Color(0.75, 1, 0.65, 0.6) if hover_valid else Color(1, 0.35, 0.25, 0.55)
	overlay.queue_redraw()


func _draw_overlay() -> void:
	for cell in vacant_plots:
		var lot := _footprint(Rect2(Vector2(cell) + Vector2(0.08, 0.08), Vector2(0.84, 0.84)))
		overlay.draw_colored_polygon(lot, Color(0.43, 0.28, 0.13, 0.24))
		lot.append(lot[0])
		overlay.draw_polyline(lot, Color("ac8249"), 1.5, true)
	super._draw_overlay()
	if active_tool == "inspect" or hud == null:
		return
	var mouse := get_viewport().get_mouse_position()
	if mouse.x >= get_viewport_rect().size.x - PANEL_WIDTH or mouse.y < RIBBON_HEIGHT:
		return
	if zoning:
		for y in range(maxi(0, mini(zone_start.y, hover_cell.y)), mini(31, maxi(zone_start.y, hover_cell.y)) + 1):
			for x in range(maxi(6, mini(zone_start.x, hover_cell.x)), mini(31, maxi(zone_start.x, hover_cell.x)) + 1):
				if not blocked_tiles.has(Vector2i(x, y)):
					overlay.draw_colored_polygon(_footprint(Rect2(x+0.04, y+0.04, 0.92, 0.92)), Color(0.58, 0.85, 0.35, 0.55))
		return
	var points := _footprint(_preview_rect())
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
		if zoning:
			zoning = false
			if get_viewport_rect().has_point(event.position) and event.position.x < get_viewport_rect().size.x - PANEL_WIDTH and event.position.y > RIBBON_HEIGHT:
				var end := Vector2i(_tile_at(get_canvas_transform().affine_inverse() * event.position).floor())
				_queue_command("zone", {"x": zone_start.x, "y": zone_start.y, "x2": end.x, "y2": end.y})
			get_viewport().set_input_as_handled()


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
		if active_tool == "hut":
			zoning = true
			zone_start = cell
		elif active_tool == "road":
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
	if what == NOTIFICATION_APPLICATION_FOCUS_OUT:
		zoning = false
		painting_roads = false
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


func _drag_cells(tool: String, start: Vector2i, end: Vector2i) -> void:
	await _smoke_click_button(tool_buttons[tool])
	await _click_cell(start, false)
	var motion := InputEventMouseMotion.new()
	motion.position = get_canvas_transform() * _iso(end.x + 0.5, end.y + 0.5)
	motion.relative = get_canvas_transform().basis_xform(_iso(end.x - start.x, end.y - start.y))
	get_viewport().push_input(motion, true)
	await get_tree().process_frame
	var release := InputEventMouseButton.new()
	release.button_index = MOUSE_BUTTON_LEFT
	release.position = motion.position
	get_viewport().push_input(release, true)
	await get_tree().process_frame


func _building_at(cell: Vector2i) -> Dictionary:
	for item: Dictionary in state.get("buildings", []):
		if _building_rect(item).has_point(Vector2(cell)):
			return item
	return {}


func _record_economy(roles: Dictionary, stocks: Dictionary) -> bool:
	for walker: Dictionary in state.get("walkers", []):
		roles[str(walker.role)] = true
	for building: Dictionary in state.get("buildings", []):
		if int(building.get("stock", 0)) > 0:
			stocks[str(building.kind)] = true
		if building.kind == "hut" and int(building.get("tier", 0)) == 2:
			stocks["small_shanty"] = true
	return _smoke_require(int(state.get("grain", 0)) + int(state.get("grain_consumed", 0)) == int(state.get("grain_produced", 0)), "grain was created or lost outside production/consumption")


func _sandbox_smoke() -> void:
	await get_tree().process_frame
	if not _smoke_require(await _wait_idle(), "sandbox initial state failed"):
		return
	_set_view(0.9)
	# Place and clear every economic building through the actual palette and map.
	# Test footprints inside the current view without taking starter jobs away.
	for entry: Array in [["farm", Vector2i(7, 6), Vector2i(3, 3)], ["granary", Vector2i(12, 12), Vector2i(4, 4)], ["bazaar", Vector2i(18, 12), Vector2i(2, 2)]]:
		var kind := str(entry[0])
		var cell: Vector2i = entry[1]
		var size: Vector2i = entry[2]
		await _smoke_click_button(tool_buttons[kind])
		await _click_cell(cell)
		if not _smoke_require(await _wait_idle(), kind + " placement failed"):
			return
		var placed := _building_at(cell)
		if not _smoke_require(not placed.is_empty() and _building_size(kind, placed) == size, kind + " footprint differs from simulation"):
			return
		var road_count: int = state.roads.size()
		await _smoke_click_button(tool_buttons.road)
		await _click_cell(cell + size - Vector2i.ONE)
		if not _smoke_require(await _wait_idle() and state.roads.size() == road_count, "road overlapped a building footprint"):
			return
		await _smoke_click_button(tool_buttons.clear)
		await _click_cell(cell + size - Vector2i.ONE)
		if not _smoke_require(await _wait_idle() and _building_at(cell).is_empty(), "clear did not remove complete " + kind + " footprint"):
			return
	_set_view(neighborhood_zoom)
	var roles: Dictionary = {}
	var stocks: Dictionary = {}
	# Observe each physical stage, not merely a final aggregate inventory.
	for sample in range(60):
		_queue_command("step", {"ticks": 4})
		if not _smoke_require(await _wait_idle(), "first harvest progression failed"):
			return
		if not _record_economy(roles, stocks):
			return
		if roles.has_all(["cart", "buyer", "seller"]) and stocks.has_all(["farm", "granary", "bazaar", "hut", "small_shanty"]):
			break
	if not _smoke_require(roles.has_all(["cart", "buyer", "seller"]) and stocks.has_all(["farm", "granary", "bazaar", "hut", "small_shanty"]), "harvest did not pass through carts, granary, buyer, bazaar, seller and housing"):
		return
	# Expand the neighborhood with roads and a reversed two-dimensional drag zone.
	await _drag_cells("road", Vector2i(13, 20), Vector2i(13, 17))
	if not _smoke_require(await _wait_idle(), "road drag failed"):
		return
	await _drag_cells("road", Vector2i(13, 17), Vector2i(16, 17))
	if not _smoke_require(await _wait_idle(), "road branch failed"):
		return
	var vacant_before := vacant_plots.size()
	await _drag_cells("hut", Vector2i(15, 19), Vector2i(14, 18))
	if not _smoke_require(await _wait_idle() and vacant_plots.size() == vacant_before + 4, "drag did not zone four additional vacant plots"):
		return
	await _smoke_click_button(tool_buttons.well)
	await _click_cell(Vector2i(16, 18))
	if not _smoke_require(await _wait_idle(), "well placement failed"):
		return
	# A second connected farm supplies the growing population in following years.
	_set_view(0.9)
	await _smoke_click_button(tool_buttons.farm)
	await _click_cell(Vector2i(7, 6))
	if not _smoke_require(await _wait_idle(), "second farm placement failed"):
		return
	await _drag_cells("road", Vector2i(10, 9), Vector2i(10, 6))
	if not _smoke_require(await _wait_idle(), "second farm road connection failed"):
		return
	_set_view(neighborhood_zoom)
	var expanded := false
	for sample in range(100):
		_queue_command("step", {"ticks": 16})
		if not _smoke_require(await _wait_idle(), "neighborhood expansion failed"):
			return
		if not _record_economy(roles, stocks):
			return
		var occupied := 0
		var fed := 0
		for y in range(18, 20):
			for x in range(14, 16):
				var home := _building_at(Vector2i(x, y))
				if int(home.get("pop", 0)) > 0: occupied += 1
				if int(home.get("tier", 0)) == 2: fed += 1
		if occupied == 4 and fed > 0:
			expanded = true
			break
	if not _smoke_require(expanded and roles.has("immigrant"), "new housing did not receive immigrants and food"):
		return
	# Save while a real shipment is in flight, then restore its complete state.
	var in_transit := false
	for sample in range(70):
		for walker: Dictionary in state.walkers:
			if int(walker.get("cargo", 0)) > 0:
				in_transit = true
		if in_transit: break
		_queue_command("step", {"ticks": 4})
		if not _smoke_require(await _wait_idle(), "shipment observation failed"):
			return
	if not _smoke_require(in_transit, "no physical cargo available for save/restore test"):
		return
	await _smoke_click_button(save_button)
	if not _smoke_require(await _wait_idle(), "SQLite save failed"):
		return
	var saved := state.duplicate(true)
	_queue_command("step", {"ticks": 17})
	if not _smoke_require(await _wait_idle(), "post-save step failed"):
		return
	await _smoke_click_button(load_button)
	if not _smoke_require(await _wait_idle() and state == saved, "SQLite restore differs for inventories or active jobs"):
		return
	await _smoke_click_button(tool_buttons.inspect)
	await _click_cell(Vector2i(20, 21))
	if not _smoke_require(not selected.is_empty() and selected.has("building") and "Food:" in detail_label.text, "housing inspector omitted live food"):
		return
	for expected in [2, 4, 1]:
		await _smoke_click_button(speed_button)
		if not _smoke_require(await _wait_idle() and simulation_speed == expected, "authoritative speed control failed"):
			return
	await _smoke_click_button(pause_button)
	if not _smoke_require(await _wait_idle() and not paused, "resume control failed"):
		return
	await _smoke_click_button(pause_button)
	if not _smoke_require(await _wait_idle() and paused, "pause control failed"):
		return
	selected = {}
	_refresh_inspector()
	status.text = "Grow your neighborhood along the bazaar's route."
	var capture := OS.get_environment("NILE_SANDBOX_CAPTURE")
	if not capture.is_empty():
		if not _smoke_require(DisplayServer.get_name() != "headless", "sandbox capture requires a display"):
			return
		DirAccess.make_dir_recursive_absolute(capture.get_base_dir())
		get_window().size = Vector2i(1920, 1080)
		if not await _save_capture(capture.get_base_dir(), capture.get_file(), Vector2i(1920, 1080)):
			return
	print("NILE_SANDBOX_PASS road-drag zoning footprints harvest cart granary buyer bazaar seller food evolution conservation speed sqlite-active-jobs")
	sim.stop()
	get_tree().quit(0)
