extends "res://scripts/street_study.gd"
## Input and presentation only. The bundled Go process owns the settlement.

const Transport = preload("res://scripts/sidecar.gd")
const CityInterface = preload("res://scripts/city_interface.gd")
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
var road_view := Transform2D.IDENTITY
var painting_roads := false
var road_start := Vector2i.ZERO
var road_preview: Array[Vector2i] = []
var road_tiles: Dictionary = {}
var right_press_position := Vector2.ZERO
var right_pointer_down := false
var right_pointer_moved := false
var hover_cell := Vector2i(-100, -100)
var hover_valid := false
var ghost: Sprite2D
var road_nodes: Array[Node] = []
var placement_notice: Label
var notice_timer: Timer
var road_signature := ""
var building_nodes: Dictionary = {}
var walker_nodes: Dictionary = {}
var walker_targets: Dictionary = {}
var walker_data: Dictionary = {}
var resources := preload("res://scripts/resource_terrain.gd").new()
var river_material := ShaderMaterial.new()
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
var city_ui: RefCounted
var active_overlay := ""
var city_smoke_mode := false
var monument_announced := false
var reset_in_progress := false
var startup_done := false
var startup_action := ""
var startup_choice := "village"


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
	city_smoke_mode = "--city-smoke" in args
	smoke_mode = "--sandbox-smoke" in args or "--gold-smoke" in args or city_smoke_mode
	startup_choice = str(get_tree().root.get_meta("nile_start_choice", "village"))
	get_tree().root.remove_meta("nile_start_choice")
	if not _load_data():
		get_tree().quit(1)
		return
	assets["well"] = {"id": "well", "path": "well.png", "footprint": [1, 1], "draw_width": 120, "anchor": [0.5, 0.99], "label": "Well"}
	assets["large_hut"] = {"id": "large_hut", "path": "large-hut.png", "footprint": [1, 1], "draw_width": 96, "anchor": [0.49, 0.87], "label": "Large Hut"}
	assets["small_shanty"] = {"id": "small_shanty", "path": "small-shanty.png", "footprint": [1, 1], "draw_width": 148, "anchor": [0.49, 0.98], "label": "Small Shanty"}
	assets["farm"] = {"id": "farm", "path": "farm.png", "footprint": [3, 3], "draw_width": 320, "anchor": [0.49, 0.93], "vertical_scale": 0.77, "label": "Grain farm"}
	assets["cart"] = {"id": "cart", "path": "cart.png", "footprint": [0.2, 0.2], "draw_width": 80, "anchor": [0.58, 0.96], "label": "Grain cart"}
	var city_assets: Variant = _read_json(ASSET_ROOT + "city-assets.json")
	if city_assets is Dictionary:
		for asset_id: String in city_assets: assets[asset_id] = city_assets[asset_id]
	assets["bazaar"]["footprint"] = [2, 2]
	assets["bazaar"]["draw_width"] = 190
	assets["granary"]["anchor"] = [0.6244, 0.9051]
	assets["granary"]["ground_shear"] = 0.2122
	assets["granary"]["vertical_scale"] = 1.0932
	for frame in range(4):
		var spec: Dictionary = assets.resident.duplicate(true)
		spec.region = [frame * 543, 0, 543, 724]
		assets["resident_%d" % frame] = spec
	layout = {"roads": [], "terrain_polygons": [
		{"kind": "silt", "points": [[-64, -64], [10, -64], [9.5, 2], [10.8, 9], [9.5, 17], [10.2, 25], [9.5, 34], [10, 96], [-64, 96]]},
		{"kind": "grass", "points": [[-64, -64], [8, -64], [8.8, 3], [8, 8], [9, 13], [8.1, 19], [9, 25], [8.4, 32], [8, 96], [-64, 96]]},
		{"kind": "water", "points": [[-64, -64], [10.3, -64], [10.3, 96], [-64, 96]]}]}
	placements.clear()
	focus = _iso(16, 15)
	neighborhood_zoom = 1.1
	RenderingServer.set_default_clear_color(Color("c8b181"))
	_build_world()
	var river := ground.get_child(-1) as Polygon2D
	river_material.shader = preload("res://shaders/river.gdshader")
	var water_region: Array = terrain.water.region
	var atlas_size := river.texture.get_size()
	river_material.set_shader_parameter("atlas_region", Vector4(
		water_region[0] / atlas_size.x, water_region[1] / atlas_size.y,
		water_region[2] / atlas_size.x, water_region[3] / atlas_size.y))
	river_material.set_shader_parameter("tile_size", Vector2(tile_width, tile_height))
	river.material = river_material
	buildings.y_sort_enabled = true
	for at: Vector2 in [Vector2(7, 5), Vector2(8.2, 7), Vector2(7.2, 9.5), Vector2(8, 12), Vector2(7.5, 19.2), Vector2(8.5, 24), Vector2(6.8, 26.5), Vector2(7.6, 29), Vector2(25, 9), Vector2(26, 12), Vector2(24, 26)]:
		var palm := Sprite2D.new()
		_skin_sprite(palm, "palm", _iso(at.x, at.y), 90)
		palm.set_meta("tile", Vector2i(at.floor()))
		buildings.add_child(palm)
		landscape_nodes.append(palm)
	_build_hud()
	placement_notice = _label("", 15, Color("392617"))
	placement_notice.add_theme_stylebox_override("normal", _style(Color("f0d8a9"), Color("79542e"), 1))
	placement_notice.mouse_filter = Control.MOUSE_FILTER_IGNORE
	hud.add_child(placement_notice)
	placement_notice.hide()
	notice_timer = Timer.new()
	notice_timer.one_shot = true
	notice_timer.timeout.connect(placement_notice.hide)
	add_child(notice_timer)
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
	if "--gold-smoke" in args:
		preload("res://tests/mine_smoke.gd").run.call_deferred(self)
	elif city_smoke_mode:
		_city_smoke.call_deferred()
	elif smoke_mode:
		_sandbox_smoke.call_deferred()


func _build_hud() -> void:
	city_ui = CityInterface.new()
	city_ui.setup(self)


func _button(text: String, callback: Callable) -> Button:
	var button := Button.new()
	button.text = text
	button.custom_minimum_size.y = 31
	button.add_theme_font_size_override("font_size", 13)
	for name in ["font_color", "font_hover_color", "font_pressed_color", "font_focus_color"]:
		button.add_theme_color_override(name, Color("392617"))
	for state_name: String in ["normal", "hover", "pressed"]:
		var fill: Color = {"normal": Color("d5a56c"), "hover": Color("edc78f"), "pressed": Color("ba824b")}[state_name]
		var style := _style(fill, Color("8a5b30"), 2)
		style.content_margin_left = 8
		style.content_margin_right = 8
		style.content_margin_top = 4
		style.content_margin_bottom = 4
		style.shadow_color = Color(0.25, 0.13, 0.04, 0.2)
		style.shadow_size = 1
		style.shadow_offset = Vector2(0, 1)
		button.add_theme_stylebox_override(state_name, style)
	button.add_theme_stylebox_override("focus", _style(Color.TRANSPARENT, Color("8b4023"), 2))
	button.pressed.connect(callback)
	button.pressed.connect(func() -> void:
		if not _modal_open(): button.release_focus())
	return button


func _queue_command(op: String, fields: Dictionary = {}) -> void:
	if sim.child.is_empty(): return
	command_queue.append({"op": op, "fields": fields})


func _new_town(scenario: String) -> void:
	startup_done = true
	startup_action = ""
	command_queue.clear()
	selected = {}
	reset_in_progress = true
	city_ui.completion_modal.hide()
	zoning = false
	painting_roads = false
	_set_tool("inspect")
	_queue_command("pause")
	_queue_command("new_town", {"scenario": scenario})
	if not smoke_mode: _queue_command("resume")


func _repair_selected() -> void:
	if selected.has("building"):
		_queue_command("repair", {"x": int(selected.building.x), "y": int(selected.building.y)})


func _turn_city(step: int) -> void:
	_cancel_gesture()
	# Reproject ground vertices while keeping painted facades upright and the
	# same geographic point in the middle of the playable viewport.
	var center_screen := (get_viewport_rect().size - Vector2(PANEL_WIDTH, -RIBBON_HEIGHT)) / 2
	var center_tile := _tile_at(get_canvas_transform().affine_inverse() * center_screen)
	var focus_tile := _tile_at(focus)
	var vertices := {}
	for polygon: Polygon2D in ground.get_children():
		if polygon in road_nodes: continue
		var tiles := PackedVector2Array()
		for vertex: Vector2 in polygon.polygon:
			tiles.append(_tile_at(vertex))
		vertices[polygon] = tiles
	var palm_tiles := PackedVector2Array()
	for palm: Node2D in landscape_nodes: palm_tiles.append(_tile_at(palm.position))
	view_turns = posmod(view_turns + step, 4)
	for polygon: Polygon2D in vertices:
		var points := PackedVector2Array()
		for tile: Vector2 in vertices[polygon]: points.append(_iso(tile.x, tile.y))
		polygon.polygon = points
		if polygon.material != river_material and polygon.material.get_shader_parameter("boundary_count") != null:
			var boundary := points.duplicate()
			boundary.resize(128)
			polygon.material.set_shader_parameter("boundary_points", boundary)
	for index in landscape_nodes.size():
		var tile := palm_tiles[index]
		landscape_nodes[index].position = _iso(tile.x, tile.y)
	road_signature = ""
	_sync_world()
	for id: String in walker_nodes: walker_nodes[id].position = walker_targets[id]
	focus = _iso(focus_tile.x, focus_tile.y)
	var center_offset := (center_screen - get_viewport_rect().size / 2) / camera.zoom
	camera.position = _iso(center_tile.x, center_tile.y) - center_offset
	camera.force_update_scroll()
	_update_ghost()


func _navigate_minimap(tile: Vector2) -> void:
	focus = _iso(tile.x, tile.y)
	_set_view(camera.zoom.x)


func _set_overlay(kind: String) -> void:
	active_overlay = kind
	overlay.queue_redraw()


func _catalog_entry(kind: String) -> Dictionary:
	for item: Dictionary in state.get("building_catalog", []):
		if item.kind == kind: return item
	return {}


func _load_save() -> void:
	_cancel_gesture()
	if sim.child.is_empty():
		stats.text = "Starting local settlement…"
		if not sim.start("sandbox"): return
		_queue_command("state")
		if not startup_done: return
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
			if sprite.get_meta("actor_asset", "resident") == "resident":
				sprite.texture = _texture_for("resident_%d" % (int(Time.get_ticks_msec() / 140) % 4))
		sprite.position = sprite.position.lerp(target, minf(delta * simulation_speed / TICK_SECONDS, 1.0))
	_update_ghost()
	if city_ui != null: city_ui.update_city()


func _on_reply(reply: Dictionary) -> void:
	if stats.text in ["Starting local settlement…", "Settlement stopped"]:
		city_ui.policy_apply.disabled = false
	if stats.text == "Starting local settlement…":
		status.text = ""
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
	stats.text = "Db  %d         Pop  %d         %s   ·   Year %d%s" % [int(state.get("treasury", 0)), int(state.get("population", 0)), str(state.get("season", "Growing")).capitalize(), int(state.get("year", 1)), "   ·   Paused" if paused else ""]
	if not reply.get("ok", false):
		status.text = str(reply.get("error", "Command failed"))
		if current_operation in ["build", "zone", "road"]: _show_placement_error(status.text)
	elif current_operation == "save":
		status.text = "Settlement saved."
	elif current_operation == "load":
		reset_in_progress = false
		status.text = "Saved settlement restored."
	elif current_operation in ["build", "road", "zone"]:
		status.text = ""
	elif current_operation == "repair":
		status.text = "Building repaired. Keep a patrol nearby to protect it."
	elif current_operation == "new_town":
		reset_in_progress = false
		monument_announced = false
		city_ui.completion_modal.hide()
		focus = _iso(11, 15) if state.get("buildings", []).is_empty() else _iso(16, 15)
		_set_view(neighborhood_zoom)
		status.text = ""
	if current_operation == "set_policy": city_ui.policy_acknowledged(reply)
	if not startup_done and reply.get("ok", false):
		if smoke_mode:
			startup_done = true
		elif not startup_action.is_empty():
			if current_operation == startup_action:
				startup_done = true
				startup_action = ""
				_queue_command("resume")
		elif current_operation == "state":
			if startup_choice == "empty":
				reset_in_progress = true
				startup_action = "new_town"
				_queue_command("new_town", {"scenario": "empty"})
			elif startup_choice == "load":
				startup_action = "load"
				_queue_command("load")
			else:
				startup_done = true
				_queue_command("resume")
	_refresh_inspector()
	var goal := int(state.get("monument_goal", 64))
	if goal > 0 and int(state.get("monument_progress", 0)) >= goal and not monument_announced and not reset_in_progress:
		monument_announced = true
		city_ui.show_completion()


func _on_failure(reason: String) -> void:
	if not startup_done: startup_action = ""
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
	city_ui.policy_apply.disabled = true


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
	var catalog := _catalog_entry(kind)
	if not catalog.is_empty():
		return Vector2i(int(catalog.get("width", 1)), int(catalog.get("height", 1)))
	if assets.has(kind):
		var size: Array = assets[kind].get("footprint", [1, 1])
		return Vector2i(int(size[0]), int(size[1]))
	return Vector2i.ONE


func _building_rect(data: Dictionary) -> Rect2:
	return Rect2(Vector2(float(data.x), float(data.y)), Vector2(_building_size(str(data.kind), data)))


func _sync_world() -> void:
	resources.sync(self, state)
	var flood_level := float(state.get("flood_level", 0.0))
	river_material.set_shader_parameter("flood_level", flood_level)
	river_material.set_shader_parameter("orientation", view_turns)
	var roads: Array = state.get("roads", [])
	road_tiles.clear()
	for tile: Dictionary in roads: road_tiles[Vector2i(int(tile.x), int(tile.y))] = true
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
	blocked_tiles = resources.rock.duplicate()
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
			var tier := clampi(int(building.get("tier", 0)), 0, 2)
			asset_id = ["hut", "large_hut", "small_shanty"][tier]
			var variant: String = ["crude_hut", "sturdy_hut", "meager_shanty"][tier] + ("_a" if int(building.id) % 2 == 0 else "_b")
			if assets.has(variant): asset_id = variant
			if vacant and assets.has("vacant_plot"): asset_id = "vacant_plot"
		elif asset_id == "mastaba":
			var stage := str(building.get("stage", "foundation"))
			asset_id = "mastaba" if stage == "complete" else ("mastaba_foundation" if stage == "foundation" else "mastaba_building")
		if not building_nodes.has(id):
			var new_sprite := Sprite2D.new()
			buildings.add_child(new_sprite)
			building_nodes[id] = new_sprite
		var sprite := building_nodes[id] as Sprite2D
		sprite.visible = not vacant or asset_id == "vacant_plot"
		_skin_sprite(sprite, asset_id, _front_corner(rect))
		if asset_id == "farm":
			if sprite.material == null:
				var crops := ShaderMaterial.new()
				crops.shader = preload("res://shaders/grain_field.gdshader")
				crops.set_shader_parameter("tile_size", Vector2(tile_width, tile_height))
				sprite.material = crops
			sprite.material.set_shader_parameter("growth", float(building.get("work", 0)) / 288.0)
			sprite.material.set_shader_parameter("flood_level", flood_level)
			sprite.material.set_shader_parameter("orientation", view_turns)
		else:
			sprite.material = null
		sprite.modulate = Color(0.4, 0.32, 0.27) if str(building.get("damaged", "")) != "" else Color.WHITE
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
		var ad: float = a.sprite.position.y
		var bd: float = b.sprite.position.y
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
			var actor_asset := _walker_asset(walker)
			_skin_sprite(sprite, actor_asset, target)
			sprite.set_meta("actor_asset", actor_asset)
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


func _walker_asset(walker: Dictionary) -> String:
	var role := str(walker.get("role", "immigrant"))
	var result := "resident"
	if role == "goldcart": result = "gold_sledge"
	elif role == "cart": result = "cart"
	elif role == "porter": result = {"clay": "clay_cart", "straw": "straw_cart", "bricks": "brick_sledge"}.get(str(walker.get("resource", "")), "cart")
	elif role in ["firefighter", "architect", "taxcollector", "builder"]:
		result = {"firefighter": "fire_marshal", "architect": "city_architect", "taxcollector": "tax_collector", "builder": "builder"}[role]
	return result if assets.has(result) else ("cart" if role == "porter" else "resident")


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
	city_ui.repair_button.visible = false
	if active_tool != "inspect" and selected.is_empty():
		var catalog := _catalog_entry(active_tool)
		title_label.text = str(catalog.get("label", active_tool.capitalize()))
		var size := _building_size(active_tool)
		kind_label.text = "%d × %d tiles · %d debens" % [size.x, size.y, int(catalog.get("cost", 0))]
		detail_label.text = _tool_guidance(active_tool)
		if int(catalog.get("cost", 0)) > int(state.get("treasury", 0)):
			detail_label.text += "\n\nThe city needs more debens before this can be built."
	elif selected.is_empty():
		title_label.text = ""
		kind_label.text = ""
		detail_label.text = ""
	elif selected.has("walker"):
		var walker: Dictionary = selected.walker
		var role := str(walker.get("role", "immigrant"))
		title_label.text = {"immigrant": "Arriving resident", "goldcart": "Palace gold cart", "cart": "Grain cart", "buyer": "Bazaar buyer", "seller": "Bazaar seller", "porter": "Material porter", "firefighter": "Firehouse patrol", "architect": "Architect", "taxcollector": "Tax collector", "builder": "Monument builder"}.get(role, role.capitalize())
		var resource := str(walker.get("resource", ""))
		if resource.is_empty(): resource = "grain"
		kind_label.text = "%d %s in transit" % [int(walker.get("cargo", 0)), resource] if int(walker.get("cargo", 0)) > 0 else "Following the city's roads"
		if role == "builder":
			kind_label.text = {"commute": "Travelling to the staging area", "work": "Working at the monument", "return": "Returning home"}.get(str(walker.get("stage", "")), "Monument construction crew")
		detail_label.text = {"immigrant": "Following connected roads to a home with room.", "goldcart": "Delivering gold directly to the village palace.", "cart": "Hauling the farm's harvest to a granary.", "buyer": "Collecting stored grain and carrying it back to the bazaar.", "seller": "Walking local roads and delivering food to nearby homes.", "porter": "Moving physical materials between workshops, storage and the monument.", "firefighter": "Inspecting buildings beside the patrol route to reduce fire risk.", "architect": "Checking structures beside the road to prevent collapse.", "taxcollector": "Visiting households to collect the city's taxes.", "builder": "Walking between home and the monument staging area for a construction shift."}.get(role, "Following the road network.")
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
		elif building.kind == "hut":
			title_label.text = str(building.get("tier_name", "Large Hut" if int(building.get("tier", 0)) > 0 else "Small Hut"))
			if int(building.get("pop", 0)) == 0:
				title_label.text = "Vacant housing"
			kind_label.text = "%d / %d residents" % [int(building.get("pop", 0)), int(building.get("capacity", 7 if int(building.get("tier", 0)) > 0 else 5))]
			detail_label.text = "Road: %s\nWater: %s\nFood: %d grain\nBazaar: %s" % ["connected" if building.get("connected", false) else "not connected", "available" if building.get("water", false) else "needed", int(building.get("stock", 0)), "recent visit" if building.get("market", false) else "needs a seller"]
			detail_label.text += "\nLast tax: %d Db\nTax collector: %s" % [int(building.get("last_tax", 0)), "recent visit" if building.get("tax_covered", false) else "not reached"]
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
		else:
			var catalog := _catalog_entry(str(building.kind))
			title_label.text = str(catalog.get("label", str(building.kind).capitalize()))
			kind_label.text = "%d / %d workers" % [int(building.get("workers", 0)), int(building.get("workers_needed", 0))]
			detail_label.text = "Road: %s" % ("connected" if building.get("connected", false) else "not connected")
			if building.kind == "goldmine":
				detail_label.text += "\nGold ready for collection: %d Db" % int(building.get("gold", 0))
			if building.kind == "palace":
				detail_label.text += "\nGold received: %d Db" % int(state.get("gold_delivered", 0))
			if building.kind in ["claypit", "brickworks", "storageyard", "mastaba"]:
				detail_label.text += "\nClay: %d\nStraw: %d\nBricks: %d" % [int(building.get("clay", 0)), int(building.get("straw", 0)), int(building.get("bricks", 0))]
			if building.kind == "mastaba":
				kind_label.text = str(building.get("stage", "foundation")).capitalize()
				detail_label.text += "\nCompleted: %d / %d\nCrew on site: %d\nCrew en route: %d\nReturning home: %d" % [int(building.get("construction_done", 0)), int(building.get("construction_required", 64)), int(building.get("crew_on_site", 0)), int(building.get("crew_en_route", 0)), int(building.get("crew_returning", 0))]
				if not building.get("staging_access", false):
					detail_label.text = "Workers cannot reach the staging area. Connect an unobstructed road to the site.\n\n" + detail_label.text
				elif int(building.get("crew_on_site", 0)) == 0 and int(building.get("construction_done", 0)) < 64:
					detail_label.text = "Waiting for builders to reach the staging area.\n\n" + detail_label.text
				elif int(building.get("bricks", 0)) == 0 and int(building.get("construction_done", 0)) < 64:
					detail_label.text = "Builders are waiting for delivered bricks.\n\n" + detail_label.text
				production_bar.visible = true
				production_bar.max_value = maxf(1, float(building.get("construction_required", 64)))
				production_bar.value = int(building.get("construction_done", 0))
			else:
				detail_label.text += "\n\n" + _tool_guidance(str(building.kind))
		if building.kind != "mastaba":
			detail_label.text += "\n\nFire risk: %d\nCollapse risk: %d" % [int(building.get("fire_risk", 0)), int(building.get("collapse_risk", 0))]
		if str(building.get("damaged", "")) != "":
			kind_label.text = "Damaged by " + str(building.damaged)
			detail_label.text = "This building cannot operate until it is repaired.\n\n" + detail_label.text
			city_ui.repair_button.visible = true
			city_ui.repair_button.text = "Repair · %d debens" % int(building.get("repair_cost", 0))
	overlay.queue_redraw()


func _set_tool(tool: String) -> void:
	if sim != null and sim.child.is_empty(): return
	_cancel_gesture()
	active_tool = tool
	if tool != "inspect": selected = {}
	city_ui.construction_menu.hide()
	for key: String in tool_buttons:
		tool_buttons[key].set_pressed_no_signal(key == tool)
	status.text = ""
	_update_ghost()
	_refresh_inspector()


func _tool_guidance(tool: String) -> String:
	return {"inspect": "Inspect buildings or follow a walker.", "road": "Drag across empty land to lay roads.", "hut": "Drag to mark housing plots beside a road.", "well": "Place a well near housing.", "farm": "Place a 3×3 farm on the dark floodplain beside a road.", "granary": "Store harvests for bazaar buyers beside a connected road.", "bazaar": "Sellers distribute food along the roads serving your homes.", "goldmine": "Build beside gold-bearing rock and connect the mine to your palace by road.", "palace": "Place one village palace on grassland to receive gold and administer taxes.", "claypit": "Extract clay from the floodplain. Porters carry it to brickworks.", "brickworks": "Combine delivered clay and farm straw into bricks.", "storageyard": "Hold clay, straw and bricks for the city's workshops and builders.", "firehouse": "Patrol connected streets to reduce fire risk.", "architect": "Inspect structures along connected roads to prevent collapse.", "taxoffice": "Send tax collectors through the streets to reach households.", "mastaba": "Supply bricks and workers to complete the monument in stages.", "clear": "Remove empty structures or roads. Residents and goods are protected."}.get(tool, "Place beside a connected road.")


func _can_place(cell: Vector2i) -> bool:
	return _placement_error(cell).is_empty()


func _placement_error(cell: Vector2i) -> String:
	var size := _building_size(active_tool)
	if cell.y < 0 or cell.x < 0 or cell.x + size.x > 32 or cell.y + size.y > 32:
		return "Outside the city site"
	if cell.x < 6: return "This site is in the Nile"
	if active_tool == "clear":
		for item: Dictionary in state.get("buildings", []):
			if _building_rect(item).has_point(Vector2(cell)):
				if int(item.get("pop", 0)) > 0: return "Residents still live here"
				for resource in ["stock", "clay", "straw", "bricks", "gold"]:
					if int(item.get(resource, 0)) > 0: return "Stored goods remain here"
				for job: Dictionary in state.get("walkers", []):
					if item.id in [job.target, job.origin, job.source]: return "A worker is still using this building"
				return ""
		var entry: Dictionary = state.get("entry", {"x": 6, "y": 15})
		var exit_tile: Dictionary = state.get("exit", entry)
		if cell in [Vector2i(int(entry.x), int(entry.y)), Vector2i(int(exit_tile.x), int(exit_tile.y))]:
			return "The city entrance must stay open"
		return "" if road_tiles.has(cell) else "Nothing here to clear"
	if active_tool in ["farm", "claypit"] and cell.x + size.x > 10:
		return "Place the whole site on the floodplain"
	if active_tool == "goldmine" and not resources.near_deposit(cell, size):
		return "Place the mine beside gold-bearing rock"
	if active_tool == "palace":
		if _has_building_kind("palace"): return "The city already has a palace"
		if not resources.has_grass(cell, size): return "The palace needs grass beneath its grounds"
	if int(_catalog_entry(active_tool).get("cost", 0)) > int(state.get("treasury", 0)):
		return "The treasury cannot afford this site"
	for y in range(cell.y, cell.y + size.y):
		for x in range(cell.x, cell.x + size.x):
			var tile := Vector2i(x, y)
			if resources.rock.has(tile): return "Rock occupies part of the site"
			if road_tiles.has(tile): return "A road crosses the site"
			if blocked_tiles.has(tile): return "Another building occupies the site"
	return ""


func _show_placement_error(reason: String) -> void:
	placement_notice.text = reason
	placement_notice.reset_size()
	var mouse := get_viewport().get_mouse_position()
	var bounds := get_viewport_rect().size - placement_notice.size - Vector2(PANEL_WIDTH + 4, 4)
	placement_notice.position = (mouse + Vector2(14, 16)).clamp(Vector2(4, RIBBON_HEIGHT), bounds)
	placement_notice.show()
	notice_timer.start(2.2)


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
	if painting_roads and not road_view.is_equal_approx(get_canvas_transform()): _cancel_gesture()
	var mouse := get_viewport().get_mouse_position()
	var over_map := _over_map(mouse)
	var tile := _tile_at(get_canvas_transform().affine_inverse() * mouse)
	hover_cell = Vector2i(floori(tile.x), floori(tile.y))
	hover_valid = _can_place(hover_cell)
	ghost.visible = active_tool not in ["inspect", "road", "clear", "hut"] and assets.has(active_tool) and over_map
	if ghost.visible:
		var rect := _preview_rect()
		_skin_sprite(ghost, active_tool, _front_corner(rect))
		ghost.modulate = Color(0.75, 1, 0.65, 0.6) if hover_valid else Color(1, 0.35, 0.25, 0.55)
	overlay.queue_redraw()


func _draw_overlay() -> void:
	if not state.is_empty() and state.get("buildings", []).is_empty():
		var entry: Dictionary = state.get("entry", {"x": 6, "y": 15})
		var at := _iso(float(entry.x) + 0.5, float(entry.y) + 0.5)
		overlay.draw_circle(at, 12, Color("f2d078"), false, 2, true)
		overlay.draw_line(at + Vector2(0, -12), at + Vector2(0, -39), Color("664824"), 2, true)
		overlay.draw_string(ThemeDB.fallback_font, at + Vector2(10, -40), "Kingdom Road" if not state.get("kingdom_road", []).is_empty() else "Entry road", HORIZONTAL_ALIGNMENT_LEFT, -1, 18, Color("35281d"))
	if not active_overlay.is_empty():
		for item: Dictionary in placements:
			var data: Dictionary = item.building
			if active_overlay in ["water", "food", "tax"] and data.kind != "hut": continue
			if active_overlay == "labor" and int(data.get("workers_needed", 0)) == 0: continue
			var need := 0.0
			match active_overlay:
				"water": need = 0.0 if data.get("water", false) else 1.0
				"food": need = 0.0 if data.get("food", false) else 1.0
				"tax": need = 0.0 if data.get("tax_covered", false) else 1.0
				"fire": need = float(data.get("fire_risk", 0)) / 100.0
				"collapse": need = float(data.get("collapse_risk", 0)) / 100.0
				"labor": need = 1.0 - float(data.get("workers", 0)) / maxf(1, float(data.get("workers_needed", 1)))
			var color := Color("6bab42").lerp(Color("d14b2b"), clampf(need, 0, 1))
			color.a = 0.4
			overlay.draw_colored_polygon(_footprint(item.rect), color)
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
	if painting_roads:
		var valid := _road_preview_valid()
		for cell: Vector2i in road_preview:
			if road_tiles.has(cell): continue
			var road := _footprint(Rect2(Vector2(cell), Vector2.ONE))
			overlay.draw_colored_polygon(road, Color(0.94, 0.79, 0.48, 0.65) if valid else Color(0.8, 0.2, 0.12, 0.55))
			road.append(road[0])
			overlay.draw_polyline(road, Color("f1d19a") if valid else Color("ce4930"), 1.5, true)
		if not road_preview.is_empty():
			var label_at := _iso(road_preview.back().x + 0.5, road_preview.back().y + 0.5) + Vector2(16, -32)
			var price := "%d Db" % _road_preview_cost() if valid else "Cannot build here"
			var text_size := ThemeDB.fallback_font.get_string_size(price, HORIZONTAL_ALIGNMENT_LEFT, -1, 16)
			overlay.draw_rect(Rect2(label_at - Vector2(5, 18), text_size + Vector2(10, 5)), Color("ead4a8"))
			overlay.draw_string(ThemeDB.fallback_font, label_at, price, HORIZONTAL_ALIGNMENT_LEFT, -1, 16, Color("35281d"))
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
	var reason := _placement_error(cell)
	if not reason.is_empty():
		_show_placement_error(reason)
		return
	placement_notice.hide()
	_queue_command("build", {"kind": active_tool, "x": cell.x, "y": cell.y})


func _preview_road_to(cell: Vector2i) -> void:
	road_preview.assign([road_start])
	var cursor := road_start
	while cursor.x != cell.x and road_preview.size() < 128:
		cursor.x += signi(cell.x - cursor.x)
		road_preview.append(cursor)
	while cursor.y != cell.y and road_preview.size() < 128:
		cursor.y += signi(cell.y - cursor.y)
		road_preview.append(cursor)
	overlay.queue_redraw()


func _road_preview_cost() -> int:
	var count := 0
	for cell: Vector2i in road_preview:
		if not road_tiles.has(cell): count += 1
	return count * int(_catalog_entry("road").get("cost", 2))


func _road_preview_valid() -> bool:
	if road_preview.is_empty() or _road_preview_cost() > int(state.get("treasury", 0)): return false
	for cell: Vector2i in road_preview:
		if cell.x < 6 or cell.x >= 32 or cell.y < 0 or cell.y >= 32: return false
		if blocked_tiles.has(cell) and not road_tiles.has(cell): return false
	return true


func _cancel_gesture() -> void:
	zoning = false
	painting_roads = false
	road_preview.clear()
	right_pointer_down = false
	right_pointer_moved = false
	dragging = false
	if overlay != null: overlay.queue_redraw()


func _modal_open() -> bool:
	var settings := get_node_or_null("/root/CitySettings")
	return (city_ui != null and city_ui.has_open_modal()) or (settings != null and settings.is_open())


func _over_map(point: Vector2) -> bool:
	return get_viewport_rect().has_point(point) and point.x < get_viewport_rect().size.x - PANEL_WIDTH and point.y > RIBBON_HEIGHT and not _modal_open() and not city_ui.construction_menu.visible


func _input(event: InputEvent) -> void:
	if _modal_open():
		_cancel_gesture()
		var settings := get_node_or_null("/root/CitySettings")
		if event is InputEventMouseButton and event.pressed and event.button_index == MOUSE_BUTTON_RIGHT and (settings == null or not settings.is_open()):
			city_ui.dismiss_modal()
			get_viewport().set_input_as_handled()
		return
	if event is InputEventMouseButton and event.button_index == MOUSE_BUTTON_RIGHT and not event.pressed and right_pointer_down:
		var inspect_click := not right_pointer_moved and _over_map(event.position)
		_cancel_gesture()
		if inspect_click:
			_select_at(get_canvas_transform().affine_inverse() * event.position)
			city_ui.show_inspector()
		get_viewport().set_input_as_handled()
	if event is InputEventMouseButton and event.button_index == MOUSE_BUTTON_LEFT and not event.pressed:
		if painting_roads:
			if _over_map(event.position) and road_view.is_equal_approx(get_canvas_transform()):
				_preview_road_to(Vector2i(_tile_at(get_canvas_transform().affine_inverse() * event.position).floor()))
				if _road_preview_valid():
					var tiles: Array[Dictionary] = []
					for cell: Vector2i in road_preview: tiles.append({"x": cell.x, "y": cell.y})
					_queue_command("road", {"tiles": tiles})
				else: status.text = "Road cannot be built here."
			_cancel_gesture()
			get_viewport().set_input_as_handled()
		if zoning:
			zoning = false
			if _over_map(event.position):
				var end := Vector2i(_tile_at(get_canvas_transform().affine_inverse() * event.position).floor())
				_queue_command("zone", {"x": zone_start.x, "y": zone_start.y, "x2": end.x, "y2": end.y})
			get_viewport().set_input_as_handled()


func _unhandled_input(event: InputEvent) -> void:
	var settings := get_node_or_null("/root/CitySettings")
	if settings != null and settings.is_open(): return
	if _modal_open():
		if (event is InputEventKey and event.pressed and event.keycode == KEY_ESCAPE) or (event is InputEventMouseButton and event.pressed and event.button_index == MOUSE_BUTTON_RIGHT):
			city_ui.dismiss_modal()
			get_viewport().set_input_as_handled()
		return
	if settings != null and settings.handle_camera_key(self, event): return
	if event is InputEventKey and event.pressed and not event.echo:
		if event.keycode == KEY_ESCAPE and city_ui != null and city_ui.dismiss_modal():
			return
		if event.keycode == KEY_SPACE:
			_toggle_pause()
			return
		if event.keycode == KEY_ESCAPE:
			_set_tool("inspect")
			selected = {}
			_refresh_inspector()
			return
	if event is InputEventMouseButton and event.button_index == MOUSE_BUTTON_RIGHT:
		if event.pressed:
			if active_tool != "inspect" or painting_roads or zoning or city_ui.construction_menu.visible:
				_set_tool("inspect")
			elif _over_map(event.position):
				right_pointer_down = true
				right_pointer_moved = false
				right_press_position = event.position
		get_viewport().set_input_as_handled()
		return
	if event is InputEventMouseMotion and right_pointer_down:
		right_pointer_moved = right_pointer_moved or event.position.distance_to(right_press_position) > 4.0
		if right_pointer_moved:
			dragging = true
			camera.position -= event.relative / camera.zoom
			_clamp_camera()
			camera.force_update_scroll()
		return
	if event is InputEventMouseButton and event.button_index == MOUSE_BUTTON_LEFT and event.pressed and active_tool != "inspect":
		var tile := _tile_at(get_canvas_transform().affine_inverse() * event.position)
		var cell := Vector2i(floori(tile.x), floori(tile.y))
		if active_tool == "hut":
			zoning = true
			zone_start = cell
		elif active_tool == "road":
			painting_roads = true
			road_view = get_canvas_transform()
			road_start = cell
			_preview_road_to(cell)
		else:
			_place_cell(cell)
		get_viewport().set_input_as_handled()
		return
	if event is InputEventMouseMotion and painting_roads:
		var tile := _tile_at(get_canvas_transform().affine_inverse() * event.position)
		_preview_road_to(Vector2i(floori(tile.x), floori(tile.y)))
		return
	super._unhandled_input(event)


func _notification(what: int) -> void:
	if what == NOTIFICATION_APPLICATION_FOCUS_OUT:
		_cancel_gesture()
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


func _smoke_choose_tool(kind: String) -> void:
	var category: String = city_ui.category_for(kind)
	if not category.is_empty():
		await _smoke_click_button(city_ui.categories[category])
	await _smoke_click_button(tool_buttons[kind])


func _drag_cells(tool: String, start: Vector2i, end: Vector2i) -> void:
	await _smoke_choose_tool(tool)
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


func _has_building_kind(kind: String) -> bool:
	for building: Dictionary in state.get("buildings", []):
		if building.kind == kind: return true
	return false


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


func _smoke_road_gesture() -> bool:
	var initial_roads: Array = state.roads.duplicate(true)
	var initial_treasury := int(state.treasury)
	await _smoke_choose_tool("road")
	await _click_cell(Vector2i(12, 12), false)
	var motion := InputEventMouseMotion.new()
	motion.position = get_canvas_transform() * _iso(14.5, 12.5)
	get_viewport().push_input(motion, true)
	await get_tree().process_frame
	if not _smoke_require(await _wait_idle() and road_preview.size() == 3 and state.roads == initial_roads and int(state.treasury) == initial_treasury, "road preview changed the city before release"): return false
	var capture_path := OS.get_environment("NILE_SANDBOX_CAPTURE")
	if not capture_path.is_empty() and DisplayServer.get_name() != "headless":
		DirAccess.make_dir_recursive_absolute(capture_path.get_base_dir())
		get_window().size = Vector2i(1920, 1080)
		if not await _save_capture(capture_path.get_base_dir(), "road-preview.png", Vector2i(1920, 1080)): return false
	var cancel := InputEventMouseButton.new()
	cancel.button_index = MOUSE_BUTTON_RIGHT
	cancel.position = motion.position
	cancel.pressed = true
	get_viewport().push_input(cancel, true)
	await get_tree().process_frame
	cancel.pressed = false
	get_viewport().push_input(cancel, true)
	var release := InputEventMouseButton.new()
	release.button_index = MOUSE_BUTTON_LEFT
	release.position = motion.position
	get_viewport().push_input(release, true)
	await get_tree().process_frame
	if not _smoke_require(await _wait_idle() and road_preview.is_empty() and active_tool == "inspect" and state.roads == initial_roads and int(state.treasury) == initial_treasury, "right-click did not cancel road spending"): return false
	await _smoke_choose_tool("road")
	await _click_cell(Vector2i(12, 12), false)
	var escape := InputEventKey.new()
	escape.keycode = KEY_ESCAPE
	escape.pressed = true
	get_viewport().push_input(escape, true)
	get_viewport().push_input(release, true)
	await get_tree().process_frame
	if not _smoke_require(await _wait_idle() and road_preview.is_empty() and state.roads == initial_roads, "Escape did not cancel road placement"): return false
	await _drag_cells("road", Vector2i(12, 12), Vector2i(14, 12))
	if not _smoke_require(await _wait_idle() and state.roads.size() == initial_roads.size() + 3 and int(state.treasury) == initial_treasury - 3 * int(_catalog_entry("road").cost), "road release did not commit exact construction cost"): return false
	var paid := int(state.treasury)
	await _drag_cells("road", Vector2i(12, 12), Vector2i(14, 12))
	if not _smoke_require(await _wait_idle() and int(state.treasury) == paid, "existing roads were charged again"): return false
	await _smoke_choose_tool("clear")
	for x in range(12, 15):
		await _click_cell(Vector2i(x, 12))
		if not _smoke_require(await _wait_idle(), "preview test road cleanup failed"): return false
	await _smoke_choose_tool("inspect")
	var inspect := InputEventMouseButton.new()
	inspect.button_index = MOUSE_BUTTON_RIGHT
	inspect.position = get_canvas_transform() * _iso(14.5, 23.5)
	inspect.pressed = true
	get_viewport().push_input(inspect, true)
	await get_tree().process_frame
	inspect.pressed = false
	get_viewport().push_input(inspect, true)
	await get_tree().process_frame
	if not _smoke_require(city_ui.inspector_modal.visible and selected.has("building") and selected.building.kind == "well", "right-click inspection did not open the well"): return false
	if not capture_path.is_empty() and DisplayServer.get_name() != "headless":
		if not await _save_capture(capture_path.get_base_dir(), "inspection.png", Vector2i(1920, 1080)): return false
	inspect.pressed = true
	get_viewport().push_input(inspect, true)
	await get_tree().process_frame
	if not _smoke_require(not city_ui.inspector_modal.visible, "right-click did not close inspection"): return false
	return true


func _sandbox_smoke() -> void:
	await get_tree().process_frame
	if not _smoke_require(await _wait_idle(), "sandbox initial state failed"):
		return
	_set_view(0.9)
	if not await _smoke_road_gesture(): return
	# Place and clear every economic building through the actual palette and map.
	# Test footprints inside the current view without taking starter jobs away.
	for entry: Array in [["farm", Vector2i(7, 6), Vector2i(3, 3)], ["granary", Vector2i(12, 12), Vector2i(4, 4)], ["bazaar", Vector2i(18, 12), Vector2i(2, 2)]]:
		var kind := str(entry[0])
		var cell: Vector2i = entry[1]
		var size: Vector2i = entry[2]
		await _smoke_choose_tool(kind)
		await _click_cell(cell)
		if not _smoke_require(await _wait_idle(), kind + " placement failed"):
			return
		var placed := _building_at(cell)
		if not _smoke_require(not placed.is_empty() and _building_size(kind, placed) == size, kind + " footprint differs from simulation"):
			return
		var road_count: int = state.roads.size()
		await _smoke_choose_tool("road")
		await _click_cell(cell + size - Vector2i.ONE)
		if not _smoke_require(await _wait_idle() and state.roads.size() == road_count, "road overlapped a building footprint"):
			return
		await _smoke_choose_tool("clear")
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
	await _smoke_choose_tool("well")
	await _click_cell(Vector2i(16, 18))
	if not _smoke_require(await _wait_idle(), "well placement failed"):
		return
	# A second connected farm supplies the growing population in following years.
	_set_view(0.9)
	await _smoke_choose_tool("farm")
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
	await _smoke_choose_tool("inspect")
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
	print("NILE_SANDBOX_PASS road-preview release-cost cancel right-inspection road-drag zoning footprints harvest cart granary buyer bazaar seller food evolution conservation speed sqlite-active-jobs")
	sim.stop()
	get_tree().quit(0)


func _smoke_popup_item(button: Button, index: int) -> void:
	await _smoke_click_button(button)
	await get_tree().process_frame
	var popup: PopupMenu = button.get_popup()
	popup.set_focused_item(index)
	var key := InputEventKey.new()
	key.keycode = KEY_ENTER
	key.pressed = true
	get_viewport().push_input(key, true)
	await get_tree().process_frame
	key.pressed = false
	get_viewport().push_input(key, true)
	await get_tree().process_frame


func _smoke_edit_rate(spin: SpinBox, value: int) -> void:
	var edit := spin.get_line_edit()
	var click := InputEventMouseButton.new()
	click.button_index = MOUSE_BUTTON_LEFT
	click.position = edit.get_global_rect().get_center()
	click.pressed = true
	get_viewport().push_input(click, true)
	click.pressed = false
	get_viewport().push_input(click, true)
	var select_all := InputEventKey.new()
	select_all.keycode = KEY_A
	select_all.ctrl_pressed = true
	select_all.pressed = true
	get_viewport().push_input(select_all, true)
	for character: String in str(value):
		var key := InputEventKey.new()
		key.unicode = character.unicode_at(0)
		key.pressed = true
		get_viewport().push_input(key, true)
	var enter := InputEventKey.new()
	enter.keycode = KEY_ENTER
	enter.pressed = true
	get_viewport().push_input(enter, true)
	enter.pressed = false
	get_viewport().push_input(enter, true)
	await get_tree().process_frame


func _city_capture(stage: String) -> bool:
	var path := OS.get_environment("NILE_CITY_CAPTURE")
	if path.is_empty(): return true
	if not _smoke_require(DisplayServer.get_name() != "headless", "city capture requires a rendering display"): return false
	var filename := path.get_file() if stage == "complete" else path.get_file().get_basename() + "-" + stage + ".png"
	DirAccess.make_dir_recursive_absolute(path.get_base_dir())
	get_window().size = Vector2i(1920, 1080)
	return await _save_capture(path.get_base_dir(), filename, Vector2i(1920, 1080))


func _city_smoke() -> void:
	await get_tree().process_frame
	if not _smoke_require(await _wait_idle(), "city initial connection failed"): return
	await _smoke_popup_item(city_ui.game_menu, 0)
	if not _smoke_require(city_ui.new_town_modal.visible, "Game menu did not open New Town"): return
	await _smoke_click_button(city_ui.new_town_empty)
	if not _smoke_require(await _wait_idle() and state.buildings.is_empty() and state.roads.size() == 32 and int(state.treasury) == 2500, "New Town did not create a funded Kingdom Road map"): return
	focus = _iso(18, 14)
	_set_view(0.7)
	for road: Array in [[Vector2i(10, 10), Vector2i(22, 10)], [Vector2i(10, 20), Vector2i(22, 20)], [Vector2i(10, 10), Vector2i(10, 20)], [Vector2i(22, 10), Vector2i(22, 20)]]:
		await _drag_cells("road", road[0], road[1])
		if not _smoke_require(await _wait_idle(), "city road construction failed"): return
	for row: Array in [[Vector2i(14, 21), Vector2i(15, 21)], [Vector2i(19, 21), Vector2i(20, 21)]]:
		await _drag_cells("hut", row[0], row[1])
		if not _smoke_require(await _wait_idle(), "city housing zone failed"): return
	for entry: Array in [["well", 14, 23], ["well", 20, 23], ["farm", 7, 10], ["granary", 14, 6], ["bazaar", 19, 8], ["firehouse", 12, 9], ["architect", 21, 9], ["taxoffice", 11, 17], ["palace", 11, 24], ["claypit", 6, 5], ["brickworks", 24, 12], ["storageyard", 25, 6], ["mastaba", 24, 17]]:
		await _smoke_choose_tool(str(entry[0]))
		await _click_cell(Vector2i(int(entry[1]), int(entry[2])))
		if not _smoke_require(await _wait_idle() and not _building_at(Vector2i(int(entry[1]), int(entry[2]))).is_empty(), "city placement failed: " + str(entry[0])): return
	for road: Array in [[Vector2i(8, 5), Vector2i(10, 5)], [Vector2i(10, 5), Vector2i(10, 9)], [Vector2i(22, 10), Vector2i(25, 10)], [Vector2i(25, 10), Vector2i(25, 9)], [Vector2i(22, 12), Vector2i(23, 12)], [Vector2i(22, 18), Vector2i(23, 18)]]:
		await _drag_cells("road", road[0], road[1])
		if not _smoke_require(await _wait_idle(), "material district road failed"): return
	if not _smoke_require(int(state.expenses) > 1000 and int(state.treasury) == 2500 - int(state.expenses), "construction treasury accounting failed"): return
	await _smoke_choose_tool("inspect")
	await _click_cell(Vector2i(25, 18))
	if not _smoke_require(selected.has("building") and selected.building.kind == "mastaba", "monument inspection failed"): return
	if not await _city_capture("foundation"): return
	await _smoke_popup_item(city_ui.overlay_picker, 3)
	if not _smoke_require(active_overlay == "fire", "fire overlay failed"): return
	await _smoke_popup_item(city_ui.overlay_picker, 0)
	await _smoke_click_button(city_ui.overseers_button)
	await _smoke_click_button(city_ui.overseer_tabs.Monument)
	if not _smoke_require(city_ui.overseer_modal.visible and "Mastaba progress" in city_ui.overseer_text.text, "monument overseer failed"): return
	await _smoke_click_button(city_ui.overseer_tabs.Treasury)
	await _smoke_edit_rate(city_ui.tax_rate, 6)
	await _smoke_edit_rate(city_ui.wage_rate, 0)
	await _smoke_popup_item(city_ui.labor_priority, 2)
	await _smoke_click_button(city_ui.policy_apply)
	if not _smoke_require(await _wait_idle() and int(state.tax_rate) == 6 and int(state.wage_rate) == 0 and state.labor_priority == "construction" and int(state.workforce) == 0, "fiscal controls did not change policy"): return
	await _smoke_edit_rate(city_ui.tax_rate, 2)
	await _smoke_edit_rate(city_ui.wage_rate, 1)
	await _smoke_popup_item(city_ui.labor_priority, 0)
	await _smoke_click_button(city_ui.policy_apply)
	if not _smoke_require(await _wait_idle() and int(state.tax_rate) == 2 and int(state.wage_rate) == 1 and state.labor_priority == "balanced", "fiscal controls did not restore wages and tax"): return
	var escape := InputEventKey.new()
	escape.keycode = KEY_ESCAPE
	escape.pressed = true
	get_viewport().push_input(escape, true)
	await get_tree().process_frame
	if not _smoke_require(not city_ui.overseer_modal.visible, "overseer close failed"): return
	for expected in [2, 4]:
		await _smoke_click_button(speed_button)
		if not _smoke_require(await _wait_idle() and simulation_speed == expected, "city speed control failed"): return
	var rendered := DisplayServer.get_name() != "headless"
	if rendered:
		await _smoke_click_button(pause_button)
		if not _smoke_require(await _wait_idle() and not paused, "city clock did not resume"): return
	var materials: Dictionary = {}
	var services: Dictionary = {}
	var builder_stages: Dictionary = {}
	var construction_captured := false
	var deadline := Time.get_ticks_msec() + 150000
	var observed_reply := -1
	while int(state.get("monument_progress", 0)) < 64 and Time.get_ticks_msec() < deadline:
		if not rendered:
			_queue_command("step", {"ticks": 4})
			if not _smoke_require(await _wait_idle(), "city progression failed"): return
		else:
			await get_tree().process_frame
		if observed_reply == replies: continue
		observed_reply = replies
		for walker: Dictionary in state.get("walkers", []):
			services[str(walker.role)] = true
			if walker.role == "builder": builder_stages[str(walker.get("stage", ""))] = true
			if walker.role == "porter" and int(walker.get("cargo", 0)) > 0:
				materials[str(walker.get("resource", ""))] = true
		for resource in ["grain", "clay", "straw", "bricks"]:
			if not _smoke_require(int(state.get(resource, 0)) + int(state.get(resource + "_consumed", 0)) == int(state.get(resource + "_produced", 0)), "city material conservation failed: " + resource): return
		if not _smoke_require(int(state.treasury) == 2500 + int(state.income) - int(state.expenses), "treasury conservation failed"): return
		if int(state.get("monument_progress", 0)) >= 16 and not construction_captured:
			construction_captured = true
			if not await _city_capture("construction"): return
	if not _smoke_require(int(state.get("monument_progress", 0)) == 64, "city did not finish its mastaba"): return
	if not _smoke_require(materials.has_all(["clay", "straw", "bricks"]) and services.has_all(["firefighter", "architect", "taxcollector"]), "city lacked physical materials or service patrols"): return
	if not _smoke_require(builder_stages.has_all(["commute", "work", "return"]), "builders did not physically travel to work and home"): return
	if not _smoke_require(city_ui.completion_modal.visible, "monument completion was not shown"): return
	if not await _city_capture("victory"): return
	await _smoke_click_button(city_ui.completion_continue)
	if not paused:
		await _smoke_click_button(pause_button)
		if not _smoke_require(await _wait_idle() and paused, "completed city did not pause"): return
	await _smoke_click_button(save_button)
	if not _smoke_require(await _wait_idle(), "completed city save failed"): return
	var saved := state.duplicate(true)
	await _smoke_popup_item(city_ui.game_menu, 0)
	await _smoke_click_button(city_ui.new_town_empty)
	if not _smoke_require(await _wait_idle() and state.buildings.is_empty(), "second New Town failed"): return
	await _smoke_click_button(load_button)
	var city_loaded := await _wait_idle()
	if not _smoke_require(city_loaded and state == saved, "saved city was changed by New Town"): return
	if city_ui.completion_modal.visible:
		await _smoke_click_button(city_ui.completion_continue)
	focus = _iso(18, 14)
	_set_view(0.7)
	await _smoke_choose_tool("inspect")
	await _click_cell(Vector2i(25, 18))
	if not await _city_capture("complete"): return
	print("NILE_CITY_PASS new-town categories overlays overseers treasury fiscal-controls physical-materials services builder-shifts staged-monument save-reset-load" + (" realtime-ui" if rendered else " deterministic-headless"))
	sim.stop()
	get_tree().quit(0)
