extends Control
## A live navigation map drawn from the same authoritative city snapshot.
signal navigate(tile: Vector2)

var extent := 32
var snapshot: Dictionary = {}
var camera_tiles := PackedVector2Array()


func _ready() -> void:
	custom_minimum_size = Vector2(170, 105)
	clip_contents = true
	mouse_default_cursor_shape = Control.CURSOR_POINTING_HAND


func update_city(value: Dictionary, corners: PackedVector2Array) -> void:
	snapshot = value
	extent = int(snapshot.get("map_extent",32))
	camera_tiles = corners
	queue_redraw()


func _point(tile: Vector2) -> Vector2:
	return Vector2(4, 4) + tile / float(extent) * (size - Vector2(8, 8))


func _draw() -> void:
	draw_rect(Rect2(Vector2.ZERO, size), Color("80653c"))
	draw_rect(Rect2(Vector2(4, 4), size - Vector2(8, 8)), Color("c1a46d"))
	var cell := (size - Vector2(8, 8)) / float(extent)
	var banks: Array = snapshot.get("banks",[])
	for tile: Dictionary in snapshot.get("grassland", []):
		draw_rect(Rect2(_point(Vector2(tile.x, tile.y)), cell), Color("8c9966"))
	var flood := 4.0 * float(snapshot.get("flood_level",0.0))
	var west_banks: Array = snapshot.get("west_banks",[])
	for y in extent:
		var west := float(west_banks[y]) if not west_banks.is_empty() else 0.0
		var bank := float(banks[y]) if not banks.is_empty() else 6.0
		draw_rect(Rect2(_point(Vector2(maxf(0,west-4),y)),cell*Vector2(bank+4.0-maxf(0,west-4),1.0)),Color("7c8551"))
		draw_rect(Rect2(_point(Vector2(maxf(0,west-flood),y)),cell*Vector2(bank+flood-maxf(0,west-flood),1.0)),Color("467e8c"))
	for tile: Dictionary in snapshot.get("ore_rock", []):
		draw_rect(Rect2(_point(Vector2(tile.x, tile.y)), cell), Color("8d7958"))
	for road: Dictionary in snapshot.get("roads", []):
		draw_rect(Rect2(_point(Vector2(float(road.x), float(road.y))), cell), Color("ead8a5"))
	for building: Dictionary in snapshot.get("buildings", []):
		var color := Color("a96d42")
		if building.kind in ["farm", "granary", "bazaar"]: color = Color("556637")
		elif building.kind == "mastaba": color = Color("f2d27b")
		elif building.kind in ["well", "firehouse", "architect", "taxoffice"]: color = Color("647e9a")
		if str(building.get("damaged", "")) != "": color = Color("bb3e23")
		draw_rect(Rect2(_point(Vector2(float(building.x), float(building.y))), cell * Vector2(float(building.get("width", 1)), float(building.get("height", 1)))), color)
	for walker: Dictionary in snapshot.get("walkers", []):
		draw_circle(_point(Vector2(float(walker.x) + 0.5, float(walker.y) + 0.5)), 1.1, Color("fff2cb"))
	if camera_tiles.size() == 4:
		var outline := PackedVector2Array()
		for tile: Vector2 in camera_tiles: outline.append(_point(tile))
		outline.append(outline[0])
		draw_polyline(outline, Color("fff0b9"), 1.0, true)
	draw_rect(Rect2(Vector2.ZERO, size), Color("b59551"), false, 2)


func _gui_input(event: InputEvent) -> void:
	var moving: bool = event is InputEventMouseMotion and (event.button_mask & MOUSE_BUTTON_MASK_LEFT) != 0
	var clicked: bool = event is InputEventMouseButton and event.button_index == MOUSE_BUTTON_LEFT and event.pressed
	if moving or clicked:
		var tile: Vector2 = (event.position - Vector2(4, 4)) / (size - Vector2(8, 8)) * float(extent)
		navigate.emit(tile.clamp(Vector2.ZERO, Vector2(extent-1,extent-1)))
		accept_event()
