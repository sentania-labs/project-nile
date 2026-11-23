extends Node2D
## Independent art-review scene. No economy or simulation state is invented here.

const ASSET_ROOT := "res://assets/street/"
const CAPTURE_SIZES := [Vector2i(1920, 1080), Vector2i(1280, 720), Vector2i(2560, 1440), Vector2i(3456, 2160), Vector2i(3440, 1440)]
const PANEL_WIDTH := 210.0
const RIBBON_HEIGHT := 44.0
const SAND := Color("e4cf9d")
const INK := Color("35281d")
const GOLD := Color("ba9046")

var tile_width := 96.0
var tile_height := 48.0
var map_width := 32
var map_height := 32
var assets: Dictionary = {}
var terrain: Dictionary = {}
var placements: Array[Dictionary] = []
var selected: Dictionary = {}
var layout: Dictionary = {}
var camera: Camera2D
var ground: Node2D
var buildings: Node2D
var overlay: Node2D
var hud: Control
var palette := GridContainer.new()
var portrait: TextureRect
var title_label: Label
var kind_label: Label
var detail_label: Label
var footprints_button: Button
var neighborhood_button: Button
var close_button: Button
var show_footprints := false
var dragging := false
var focus := Vector2.ZERO
var neighborhood_zoom := 0.85
var terrain_shader: Shader
var view_turns := 0


func _ready() -> void:
	# Official release templates disable command-line scene path overrides.
	if "--diagnostic" in OS.get_cmdline_user_args():
		get_tree().change_scene_to_file.call_deferred("res://scenes/neighborhood.tscn")
		return
	get_window().title = "Project Nile · Street study"
	if not _load_data():
		get_tree().quit(1)
		return
	RenderingServer.set_default_clear_color(Color("c8b181"))
	_build_world()
	_build_hud()
	_set_view(neighborhood_zoom)
	if "--street-smoke" in OS.get_cmdline_user_args():
		_run_smoke.call_deferred()


func _read_json(path: String) -> Variant:
	if not FileAccess.file_exists(path):
		push_error("Street study data missing: " + path)
		return null
	var parser := JSON.new()
	if parser.parse(FileAccess.get_file_as_string(path)) != OK:
		push_error("Street study invalid JSON: %s: %s" % [path, parser.get_error_message()])
		return null
	return parser.data


func _asset_path(path: String) -> String:
	return path if path.begins_with("res://") else ASSET_ROOT + path


func _load_data() -> bool:
	var manifest: Variant = _read_json(ASSET_ROOT + "manifest.json")
	var loaded_layout: Variant = _read_json(ASSET_ROOT + "layout.json")
	if not manifest is Dictionary or not loaded_layout is Dictionary:
		return false
	layout = loaded_layout
	var entries: Variant = manifest.get("assets", [])
	if entries is Dictionary:
		for key: String in entries:
			var entry: Dictionary = entries[key].duplicate()
			entry["id"] = key
			assets[key] = entry
	else:
		for entry: Dictionary in entries:
			assets[str(entry.get("id", ""))] = entry
	terrain = manifest.get("terrain", {})
	if not terrain.has("sand") and FileAccess.file_exists(ASSET_ROOT + "terrain.png"):
		terrain["sand"] = "terrain.png"
	if assets.is_empty() or not terrain.has("sand"):
		push_error("Street study requires original building assets and a sand terrain texture")
		return false
	tile_width = float(layout.get("tile_width", 96))
	tile_height = float(layout.get("tile_height", 48))
	map_width = int(layout.get("width", 32))
	map_height = int(layout.get("height", 32))
	if tile_width <= 0 or tile_height <= 0 or map_width <= 0 or map_height <= 0:
		push_error("Street study map dimensions must be positive")
		return false
	var focus_data: Array = layout.get("focus", [map_width * 0.5, map_height * 0.5])
	focus = _iso(float(focus_data[0]), float(focus_data[1]))
	neighborhood_zoom = clampf(float(layout.get("zoom", 0.85)), 0.35, 2.3)
	var ids: Dictionary = {}
	for placement: Dictionary in layout.get("placements", []):
		var asset_id := str(placement.get("asset", ""))
		var instance_id := str(placement.get("id", ""))
		if not assets.has(asset_id) or instance_id.is_empty() or ids.has(instance_id):
			push_error("Street study placement requires a known asset and unique id: " + str(placement))
			return false
		ids[instance_id] = true
		var item: Dictionary = placement.duplicate()
		item["spec"] = assets[asset_id]
		var footprint: Array = assets[asset_id].get("footprint", [1, 1])
		item["rect"] = Rect2(float(item.get("x", 0)), float(item.get("y", 0)), float(footprint[0]), float(footprint[1]))
		item["order"] = placements.size()
		placements.append(item)
	placements.sort_custom(func(a: Dictionary, b: Dictionary) -> bool:
		var ar: Rect2 = a.rect
		var br: Rect2 = b.rect
		var ad := ar.end.x + ar.end.y
		var bd := br.end.x + br.end.y
		return int(a.order) < int(b.order) if is_equal_approx(ad, bd) else ad < bd)
	for spec: Dictionary in assets.values():
		if not ResourceLoader.exists(_asset_path(str(spec.get("path", "")))):
			push_error("Street study asset missing: " + str(spec.get("path", "")))
			return false
	for kind: String in terrain:
		var descriptor: Variant = terrain[kind]
		var path: String = str(descriptor.get("path", "")) if descriptor is Dictionary else str(descriptor)
		if not ResourceLoader.exists(_asset_path(path)):
			push_error("Street study terrain missing: " + path)
			return false
	return true


func _quarter_turn(point: Vector2, turns: int) -> Vector2:
	match posmod(turns, 4):
		1: return Vector2(-point.y, point.x)
		2: return -point
		3: return Vector2(point.y, -point.x)
	return point


func _iso(x: float, y: float) -> Vector2:
	var point := _quarter_turn(Vector2(x, y), view_turns)
	return Vector2((point.x - point.y) * tile_width * 0.5, (point.x + point.y) * tile_height * 0.5)


func _tile_at(point: Vector2) -> Vector2:
	var projected := Vector2(point.x / tile_width + point.y / tile_height, point.y / tile_height - point.x / tile_width)
	return _quarter_turn(projected, -view_turns)


func _front_corner(rect: Rect2) -> Vector2:
	match view_turns:
		1: return _iso(rect.end.x, rect.position.y)
		2: return _iso(rect.position.x, rect.position.y)
		3: return _iso(rect.position.x, rect.end.y)
	return _iso(rect.end.x, rect.end.y)


func _footprint(rect: Rect2) -> PackedVector2Array:
	return PackedVector2Array([_iso(rect.position.x, rect.position.y), _iso(rect.end.x, rect.position.y), _iso(rect.end.x, rect.end.y), _iso(rect.position.x, rect.end.y)])


func _build_world() -> void:
	ground = Node2D.new()
	ground.name = "TexturedTerrain"
	ground.z_index = -20
	add_child(ground)
	terrain_shader = Shader.new()
	terrain_shader.code = """shader_type canvas_item;
uniform vec4 atlas_region = vec4(0.0, 0.0, 1.0, 1.0);
uniform float edge_feather = 0.0;
uniform int boundary_count = 0;
uniform vec2 boundary_points[128];
uniform bool road = false;
uniform int view_turns = 0;
uniform vec2 road_origin = vec2(0.0);
uniform vec4 exposed_edges = vec4(0.0);
uniform vec2 tile_size = vec2(96.0, 48.0);
varying vec2 ground_position;
void vertex() {
    ground_position = VERTEX;
}
float segment_distance(vec2 point, vec2 a, vec2 b) {
    vec2 segment = b - a;
    float along = clamp(dot(point - a, segment) / max(dot(segment, segment), 0.0001), 0.0, 1.0);
    return length(point - a - segment * along);
}
void fragment() {
    vec2 inset = TEXTURE_PIXEL_SIZE * 0.5;
    vec2 mirrored_uv = 1.0 - abs(mod(UV, vec2(2.0)) - 1.0);
    vec2 tiled_uv = atlas_region.xy + inset + mirrored_uv * (atlas_region.zw - inset * 2.0);
    COLOR = texture(TEXTURE, tiled_uv);
    if (edge_feather > 0.0) {
        float edge_distance = 100000.0;
        if (road) {
            vec2 cell = vec2(ground_position.x / tile_size.x + ground_position.y / tile_size.y,
                             ground_position.y / tile_size.y - ground_position.x / tile_size.x);
            if (view_turns == 1) { cell = vec2(cell.y, -cell.x); }
            if (view_turns == 2) { cell = -cell; }
            if (view_turns == 3) { cell = vec2(-cell.y, cell.x); }
            cell -= road_origin;
            float perpendicular_scale = 1.0 / length(1.0 / tile_size);
            if (exposed_edges.x > 0.5) { edge_distance = min(edge_distance, cell.x * perpendicular_scale); }
            if (exposed_edges.y > 0.5) { edge_distance = min(edge_distance, (1.0 - cell.x) * perpendicular_scale); }
            if (exposed_edges.z > 0.5) { edge_distance = min(edge_distance, cell.y * perpendicular_scale); }
            if (exposed_edges.w > 0.5) { edge_distance = min(edge_distance, (1.0 - cell.y) * perpendicular_scale); }
            float irregularity = max(0.0, 0.55 + 0.4 * sin(ground_position.x * 0.23 + ground_position.y * 0.17)
                                    + 0.2 * sin(ground_position.y * 0.61 - ground_position.x * 0.13));
            edge_distance -= irregularity;
        } else {
            for (int i = 0; i < 128; i++) {
                if (i >= boundary_count) { break; }
                int next = (i + 1) % boundary_count;
                edge_distance = min(edge_distance, segment_distance(ground_position, boundary_points[i], boundary_points[next]));
            }
        }
        COLOR.a *= smoothstep(0.0, edge_feather, edge_distance);
    }
}
"""
	# Overscan keeps the study within continuous valley terrain while panning.
	_add_terrain_polygon("sand", _footprint(Rect2(-64, -64, map_width + 128, map_height + 128)))
	for region: Dictionary in layout.get("terrain_regions", []):
		var box: Array = region.get("rect", [0, 0, 1, 1])
		_add_terrain_polygon(str(region.get("kind", "sand")), _footprint(Rect2(float(box[0]), float(box[1]), float(box[2]), float(box[3]))))
	for region: Dictionary in layout.get("terrain_polygons", []):
		var points := PackedVector2Array()
		for point: Array in region.get("points", []):
			points.append(_iso(float(point[0]), float(point[1])))
		if points.size() >= 3:
			var kind := str(region.get("kind", "sand"))
			_add_terrain_polygon(kind, points, 8.0 if kind in ["grass", "water"] else 0.0)
	for tile: Dictionary in layout.get("terrain_tiles", []):
		_add_terrain_polygon(str(tile.get("kind", "sand")), _footprint(Rect2(float(tile.x), float(tile.y), 1, 1)))
	var road_cells: Dictionary = {}
	for tile: Dictionary in layout.get("roads", []):
		road_cells[Vector2i(int(tile.x), int(tile.y))] = true
	for cell: Vector2i in road_cells:
		var exposed := Vector4(float(not road_cells.has(cell + Vector2i.LEFT)), float(not road_cells.has(cell + Vector2i.RIGHT)), float(not road_cells.has(cell + Vector2i.UP)), float(not road_cells.has(cell + Vector2i.DOWN)))
		_add_terrain_polygon("road", _footprint(Rect2(Vector2(cell), Vector2.ONE)), 3.0, Vector2(cell), exposed)
	buildings = Node2D.new()
	buildings.name = "SeparateBuildingSprites"
	add_child(buildings)
	for item: Dictionary in placements:
		var spec: Dictionary = item.spec
		var rect: Rect2 = item.rect
		var texture := load(_asset_path(str(spec.path))) as Texture2D
		if spec.has("region"):
			var region: Array = spec.region
			var atlas_texture := AtlasTexture.new()
			atlas_texture.atlas = texture
			atlas_texture.region = Rect2(float(region[0]), float(region[1]), float(region[2]), float(region[3]))
			atlas_texture.filter_clip = true
			texture = atlas_texture
		var sprite := Sprite2D.new()
		sprite.name = str(item.id)
		sprite.texture = texture
		sprite.centered = false
		var anchor: Array = spec.get("anchor", [0.5, 1.0])
		sprite.offset = -texture.get_size() * Vector2(float(anchor[0]), float(anchor[1]))
		sprite.position = _iso(rect.end.x, rect.end.y)
		var draw_width := float(spec.get("draw_width", (rect.size.x + rect.size.y) * tile_width * 0.5))
		# Match the painted base edges to the road axes while keeping walls upright.
		var unit := draw_width / texture.get_width()
		sprite.transform = Transform2D(Vector2(1, spec.get("ground_shear", 0.0)) * unit,
				Vector2(0, spec.get("vertical_scale", 1.0)) * unit, sprite.position)
		sprite.texture_filter = CanvasItem.TEXTURE_FILTER_LINEAR_WITH_MIPMAPS
		buildings.add_child(sprite)
		item["sprite"] = sprite
	overlay = Node2D.new()
	overlay.name = "FootprintInspection"
	overlay.z_index = -10
	overlay.draw.connect(_draw_overlay)
	add_child(overlay)
	camera = Camera2D.new()
	camera.name = "StreetCamera"
	add_child(camera)
	camera.make_current()


func _add_terrain_polygon(kind: String, points: PackedVector2Array, feather: float = 0.0, road_origin: Vector2 = Vector2.ZERO, exposed_edges: Vector4 = Vector4.ZERO) -> void:
	if not terrain.has(kind):
		push_error("Street study terrain kind missing: " + kind)
		get_tree().quit(1)
		return
	var descriptor: Variant = terrain[kind]
	var path: String = str(descriptor.get("path", "")) if descriptor is Dictionary else str(descriptor)
	var texture := load(_asset_path(path)) as Texture2D
	var texture_size := texture.get_size()
	var atlas_rect := Rect2(Vector2.ZERO, texture_size)
	var repeat_size := 192.0
	if descriptor is Dictionary:
		var region: Array = descriptor.get("region", [0, 0, texture_size.x, texture_size.y])
		atlas_rect = Rect2(float(region[0]), float(region[1]), float(region[2]), float(region[3]))
		repeat_size = float(descriptor.get("repeat_size", 192.0))
	var polygon := Polygon2D.new()
	polygon.polygon = points
	polygon.texture = texture
	polygon.texture_filter = CanvasItem.TEXTURE_FILTER_LINEAR
	var uvs := PackedVector2Array()
	for point: Vector2 in points:
		uvs.append(point / repeat_size * texture_size)
	polygon.uv = uvs
	var material := ShaderMaterial.new()
	material.shader = terrain_shader
	material.set_shader_parameter("atlas_region", Vector4(atlas_rect.position.x / texture_size.x, atlas_rect.position.y / texture_size.y, atlas_rect.size.x / texture_size.x, atlas_rect.size.y / texture_size.y))
	material.set_shader_parameter("edge_feather", feather)
	if feather > 0.0:
		material.set_shader_parameter("road", kind == "road")
		if kind == "road":
			material.set_shader_parameter("view_turns", view_turns)
			material.set_shader_parameter("road_origin", road_origin)
			material.set_shader_parameter("exposed_edges", exposed_edges)
			material.set_shader_parameter("tile_size", Vector2(tile_width, tile_height))
		else:
			if points.size() > 128:
				push_error("Street study feathered terrain exceeds 128 boundary points")
				get_tree().quit(1)
				return
			var boundary_points := points.duplicate()
			boundary_points.resize(128)
			material.set_shader_parameter("boundary_count", points.size())
			material.set_shader_parameter("boundary_points", boundary_points)
	polygon.material = material
	ground.add_child(polygon)


func _style(fill: Color, border: Color, thickness: int = 1) -> StyleBoxFlat:
	var style := StyleBoxFlat.new()
	style.bg_color = fill
	style.border_color = border
	style.set_border_width_all(thickness)
	style.content_margin_left = 12
	style.content_margin_right = 12
	style.content_margin_top = 7
	style.content_margin_bottom = 7
	return style


func _label(text: String, size: int = 15, color: Color = SAND) -> Label:
	var label := Label.new()
	label.text = text
	label.add_theme_font_size_override("font_size", size)
	label.add_theme_color_override("font_color", color)
	return label


func _button(text: String, callback: Callable) -> Button:
	var button := Button.new()
	button.text = text
	button.custom_minimum_size.y = 34
	button.add_theme_font_size_override("font_size", 14)
	button.add_theme_color_override("font_color", SAND)
	button.add_theme_color_override("font_hover_color", Color("fff0c8"))
	button.add_theme_color_override("font_pressed_color", Color("fff0c8"))
	button.add_theme_stylebox_override("normal", _style(Color("493b2a"), Color("76603c")))
	button.add_theme_stylebox_override("hover", _style(Color("625033"), GOLD))
	button.add_theme_stylebox_override("pressed", _style(Color("746039"), GOLD))
	button.add_theme_stylebox_override("focus", _style(Color(0, 0, 0, 0), GOLD, 2))
	button.pressed.connect(callback)
	return button


func _build_hud() -> void:
	var layer := CanvasLayer.new()
	layer.name = "StudyHUD"
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
	row.add_theme_constant_override("separation", 24)
	ribbon.add_child(row)
	row.add_child(_label("N I L E", 21, Color("f2db9f")))
	row.add_child(_label("R I V E R B A N K", 12))
	var spacer := Control.new()
	spacer.size_flags_horizontal = Control.SIZE_EXPAND_FILL
	row.add_child(spacer)
	row.add_child(_label("STREET STUDY", 12, Color("c9b681")))
	var sidebar_texture := TextureRect.new()
	sidebar_texture.texture = load(ASSET_ROOT + "sidebar.png") as Texture2D
	sidebar_texture.expand_mode = TextureRect.EXPAND_IGNORE_SIZE
	sidebar_texture.stretch_mode = TextureRect.STRETCH_SCALE
	sidebar_texture.mouse_filter = Control.MOUSE_FILTER_IGNORE
	sidebar_texture.set_anchors_and_offsets_preset(Control.PRESET_RIGHT_WIDE)
	sidebar_texture.offset_left = -PANEL_WIDTH
	sidebar_texture.offset_top = RIBBON_HEIGHT
	hud.add_child(sidebar_texture)
	var sidebar := PanelContainer.new()
	sidebar.set_anchors_and_offsets_preset(Control.PRESET_RIGHT_WIDE)
	sidebar.offset_left = -PANEL_WIDTH
	sidebar.offset_top = RIBBON_HEIGHT
	var sidebar_style := _style(Color.TRANSPARENT, Color.TRANSPARENT, 0)
	sidebar_style.content_margin_left = 14
	sidebar_style.content_margin_right = 14
	sidebar_style.content_margin_top = 90
	sidebar_style.content_margin_bottom = 44
	sidebar.add_theme_stylebox_override("panel", sidebar_style)
	hud.add_child(sidebar)
	var column := VBoxContainer.new()
	column.add_theme_constant_override("separation", 7)
	sidebar.add_child(column)
	column.add_child(_label("BUILDINGS", 13, INK))
	palette.columns = 2
	column.add_child(palette)
	for asset_id: String in ["granary", "bazaar", "hut", "house"]:
		var spec: Dictionary = assets[asset_id]
		var tile := _button("", _inspect_kind.bind(asset_id))
		tile.icon = load(_asset_path(spec.path))
		tile.texture_filter = CanvasItem.TEXTURE_FILTER_LINEAR_WITH_MIPMAPS
		tile.expand_icon = true
		tile.icon_alignment = HORIZONTAL_ALIGNMENT_CENTER
		tile.custom_minimum_size = Vector2(85, 65)
		tile.tooltip_text = spec.label
		tile.accessibility_name = spec.label
		palette.add_child(tile)
	portrait = TextureRect.new()
	portrait.texture_filter = CanvasItem.TEXTURE_FILTER_LINEAR_WITH_MIPMAPS
	portrait.expand_mode = TextureRect.EXPAND_IGNORE_SIZE
	portrait.stretch_mode = TextureRect.STRETCH_KEEP_ASPECT_CENTERED
	portrait.custom_minimum_size.y = 106
	portrait.visible = false
	column.add_child(portrait)
	title_label = _label("The riverbank", 21, INK)
	title_label.autowrap_mode = TextServer.AUTOWRAP_WORD_SMART
	column.add_child(title_label)
	kind_label = _label("An Egyptian neighborhood", 13, Color("735936"))
	kind_label.autowrap_mode = TextServer.AUTOWRAP_WORD_SMART
	column.add_child(kind_label)
	var rule := HSeparator.new()
	column.add_child(rule)
	detail_label = _label("Choose a building on the map or in the palette.", 14, INK)
	detail_label.autowrap_mode = TextServer.AUTOWRAP_WORD_SMART
	column.add_child(detail_label)
	column.add_child(HSeparator.new())
	column.add_child(_label("VIEW", 12, Color("72552f")))
	neighborhood_button = _button("Street", func() -> void: _set_view(neighborhood_zoom))
	var views := HBoxContainer.new()
	column.add_child(views)
	views.add_child(neighborhood_button)
	close_button = _button("Detail", func() -> void: _set_view(minf(neighborhood_zoom * 1.55, 2.3)))
	views.add_child(close_button)
	footprints_button = _button("Footprints", _toggle_footprints)
	footprints_button.toggle_mode = true
	column.add_child(footprints_button)
	var controls := _label("Wheel to zoom\nRight-drag to pan\nF  Footprints     Esc  Clear", 12, Color("735936"))
	controls.add_theme_constant_override("line_spacing", 5)
	column.add_child(controls)



func _set_view(zoom_value: float) -> void:
	camera.zoom = Vector2.ONE * clampf(zoom_value, 0.35, 2.3)
	var center := focus
	if not selected.is_empty() and zoom_value > neighborhood_zoom:
		var rect: Rect2 = selected.rect
		center = _iso(rect.get_center().x, rect.get_center().y) - Vector2(0, 45)
	camera.position = center + Vector2(PANEL_WIDTH / (2.0 * camera.zoom.x), -RIBBON_HEIGHT / (2.0 * camera.zoom.y))
	camera.force_update_scroll()


func _zoom_at(screen_point: Vector2, factor: float) -> void:
	var before := get_canvas_transform().affine_inverse() * screen_point
	camera.zoom = Vector2.ONE * clampf(camera.zoom.x * factor, 0.35, 2.3)
	camera.force_update_scroll()
	var after := get_canvas_transform().affine_inverse() * screen_point
	camera.position += before - after
	_clamp_camera()
	camera.force_update_scroll()


func _clamp_camera() -> void:
	var tile := _tile_at(camera.position)
	tile.x = clampf(tile.x, -12, map_width + 12)
	tile.y = clampf(tile.y, -12, map_height + 12)
	camera.position = _iso(tile.x, tile.y)


func _input(event: InputEvent) -> void:
	# A drag released over the HUD must still end the camera gesture.
	if event is InputEventMouseButton and event.button_index == MOUSE_BUTTON_RIGHT and not event.pressed:
		dragging = false


func _unhandled_input(event: InputEvent) -> void:
	if event is InputEventMouseButton:
		if event.button_index == MOUSE_BUTTON_RIGHT:
			dragging = event.pressed
			get_viewport().set_input_as_handled()
		elif event.pressed and event.button_index == MOUSE_BUTTON_WHEEL_UP:
			_zoom_at(event.position, 1.12)
		elif event.pressed and event.button_index == MOUSE_BUTTON_WHEEL_DOWN:
			_zoom_at(event.position, 1.0 / 1.12)
		elif event.pressed and event.button_index == MOUSE_BUTTON_LEFT:
			_select_at(get_canvas_transform().affine_inverse() * event.position)
	elif event is InputEventMouseMotion and dragging:
		camera.position -= event.relative / camera.zoom
		_clamp_camera()
		camera.force_update_scroll()
	elif event is InputEventKey and event.pressed and not event.echo:
		if event.keycode == KEY_F:
			_toggle_footprints()
		elif event.keycode == KEY_ESCAPE:
			_select_at(Vector2(100000, 100000))


func _select_at(world_point: Vector2) -> void:
	selected = {}
	# Pick the visible roof or facade before falling back to its ground occupancy.
	for index in range(placements.size() - 1, -1, -1):
		if not bool(placements[index].get("selectable", true)):
			continue
		var sprite := placements[index].sprite as Sprite2D
		var local_point := sprite.to_local(world_point)
		if sprite.visible and sprite.get_rect().has_point(local_point) and sprite.is_pixel_opaque(local_point):
			selected = placements[index]
			break
	if selected.is_empty():
		var tile := _tile_at(world_point)
		for index in range(placements.size() - 1, -1, -1):
			if not bool(placements[index].get("selectable", true)):
				continue
			var rect: Rect2 = placements[index].rect
			if rect.has_point(tile):
				selected = placements[index]
				break
	_refresh_inspector()


func _inspect_kind(asset_id: String) -> void:
	for item: Dictionary in placements:
		if item.asset == asset_id:
			selected = item
			_refresh_inspector()
			_set_view(minf(neighborhood_zoom * 1.55, 2.3))
			return


func _refresh_inspector() -> void:
	portrait.visible = not selected.is_empty()
	if portrait.visible:
		portrait.texture = selected.sprite.texture
	if selected.is_empty():
		title_label.text = "The riverbank"
		kind_label.text = "An Egyptian neighborhood"
		detail_label.text = "Choose a building on the map or in the palette."
	else:
		var spec: Dictionary = selected.spec
		title_label.text = str(spec.get("label", spec.get("name", selected.asset.capitalize())))
		kind_label.text = str(spec.get("kind", "Building reference"))
		detail_label.text = str(spec.get("description", "Original illustrated asset. Review scale, roof forms and street placement."))
	overlay.queue_redraw()


func _toggle_footprints() -> void:
	show_footprints = not show_footprints
	footprints_button.set_pressed_no_signal(show_footprints)
	overlay.queue_redraw()


func _draw_overlay() -> void:
	for item: Dictionary in placements:
		if not bool(item.get("selectable", true)):
			continue
		var is_selected: bool = not selected.is_empty() and item.id == selected.id
		if not show_footprints and not is_selected:
			continue
		var points := _footprint(item.rect)
		var color := Color("fff0b9") if is_selected else Color("a89157")
		if is_selected:
			overlay.draw_colored_polygon(points, Color(0.95, 0.82, 0.47, 0.10))
		points.append(points[0])
		overlay.draw_polyline(points, color, 2.0 if is_selected else 1.0, true)


func _smoke_require(condition: bool, message: String) -> bool:
	if not condition:
		push_error("Street smoke: " + message)
		get_tree().quit(1)
	return condition


func _smoke_click_button(button: Button) -> void:
	var click := InputEventMouseButton.new()
	click.button_index = MOUSE_BUTTON_LEFT
	click.position = button.get_global_rect().get_center()
	click.pressed = true
	get_viewport().push_input(click, true)
	await get_tree().process_frame
	click.pressed = false
	get_viewport().push_input(click, true)
	await get_tree().process_frame


func _save_capture(directory: String, filename: String, size: Vector2i) -> bool:
	await get_tree().process_frame
	await get_tree().process_frame
	await RenderingServer.frame_post_draw
	var image := get_viewport().get_texture().get_image()
	if not _smoke_require(image.get_size() == size, "capture size mismatch: %s versus %s" % [image.get_size(), size]):
		return false
	return _smoke_require(image.save_png(directory.path_join(filename)) == OK, "capture write failed: " + filename)


func _run_smoke() -> void:
	await get_tree().process_frame
	if not _smoke_require(not placements.is_empty(), "no selectable placements"):
		return
	var target: Dictionary = {}
	for item: Dictionary in placements:
		if not bool(item.get("selectable", true)):
			continue
		if target.is_empty():
			target = item
		if "granary" in str(item.asset).to_lower():
			target = item
			break
	if not _smoke_require(not target.is_empty(), "no selectable placements"):
		return
	var rect: Rect2 = target.rect
	var click := InputEventMouseButton.new()
	click.button_index = MOUSE_BUTTON_LEFT
	click.pressed = true
	click.position = get_canvas_transform() * _iso(rect.get_center().x, rect.get_center().y)
	get_viewport().push_input(click, true)
	await get_tree().process_frame
	if not _smoke_require(not selected.is_empty() and selected.id == target.id, "occupancy selection failed"):
		return
	await _smoke_click_button(palette.get_child(1) as Button)
	if not _smoke_require(selected.get("asset") == "bazaar" and portrait.visible and portrait.texture == selected.sprite.texture, "bazaar palette selection failed"):
		return
	await _smoke_click_button(footprints_button)
	if not _smoke_require(show_footprints and footprints_button.button_pressed, "footprint toggle failed"):
		return
	await _smoke_click_button(footprints_button)
	var before_zoom := camera.zoom.x
	var wheel := InputEventMouseButton.new()
	wheel.button_index = MOUSE_BUTTON_WHEEL_UP
	wheel.pressed = true
	wheel.position = get_viewport_rect().size * 0.5
	get_viewport().push_input(wheel, true)
	await get_tree().process_frame
	if not _smoke_require(camera.zoom.x > before_zoom, "wheel zoom failed"):
		return
	var before_pan := camera.position
	var press := InputEventMouseButton.new()
	press.button_index = MOUSE_BUTTON_RIGHT
	press.pressed = true
	press.position = get_viewport_rect().size * 0.5
	get_viewport().push_input(press, true)
	await get_tree().process_frame
	var motion := InputEventMouseMotion.new()
	motion.relative = Vector2(30, 20)
	motion.position = press.position + motion.relative
	get_viewport().push_input(motion, true)
	await get_tree().process_frame
	press.pressed = false
	press.position = motion.position
	get_viewport().push_input(press, true)
	await get_tree().process_frame
	if not _smoke_require(camera.position != before_pan and not dragging, "right-drag pan failed"):
		return
	await _smoke_click_button(neighborhood_button)
	if not _smoke_require(is_equal_approx(camera.zoom.x, neighborhood_zoom), "neighborhood view failed"):
		return
	await _smoke_click_button(close_button)
	if not _smoke_require(camera.zoom.x > neighborhood_zoom, "detail view failed"):
		return
	await _smoke_click_button(neighborhood_button)
	var clear_selection := InputEventKey.new()
	clear_selection.keycode = KEY_ESCAPE
	clear_selection.pressed = true
	get_viewport().push_input(clear_selection, true)
	await get_tree().process_frame
	if not _smoke_require(selected.is_empty(), "clear selection failed"):
		return
	var capture_dir := OS.get_environment("NILE_STREET_CAPTURE_DIR")
	if not capture_dir.is_empty():
		if not _smoke_require(DisplayServer.get_name() != "headless", "capture requires a rendering display"):
			return
		if not _smoke_require(DirAccess.make_dir_recursive_absolute(capture_dir) == OK, "capture directory is unavailable"):
			return
		get_window().size = Vector2i(1920, 1080)
		await get_tree().process_frame
		await get_tree().process_frame
		await _smoke_click_button(close_button)
		if not await _save_capture(capture_dir, "street-detail.png", Vector2i(1920, 1080)):
			return
		await _smoke_click_button(neighborhood_button)
		for size: Vector2i in CAPTURE_SIZES:
			get_window().size = size
			var filename := "street-%dx%d.png" % [size.x, size.y]
			if not await _save_capture(capture_dir, filename, size):
				return
	print("NILE_STREET_PASS selection footprints wheel-zoom drag-pan view-presets" + (" captures=6" if not capture_dir.is_empty() else " headless"))
	get_tree().quit(0)
