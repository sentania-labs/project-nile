extends RefCounted
## Presentation only: animated feet, worksite activity, and visible construction.
## Every route, staff count, cargo and completed brick remains owned by Go.

const ROOT := "res://assets/motion/"
const WORKSITES := {
	"farm": {"clip": "worker-dig", "points": [Vector2(0.35, 0.60), Vector2(0.64, 0.70)]},
	"claypit": {"clip": "worker-dig", "points": [Vector2(0.43, 0.83), Vector2(0.67, 0.71)]},
	"goldmine": {"clip": "worker-dig", "points": [Vector2(0.39, 0.62), Vector2(0.67, 0.77)]},
	"brickworks": {"clip": "worker-brick", "points": [Vector2(0.51, 0.85), Vector2(0.72, 0.80)]},
	"bazaar": {"clip": "vendor-work", "points": [Vector2(0.54, 0.72)]},
	"granary": {"clip": "vendor-work", "points": [Vector2(0.34, 0.80)]},
	"storageyard": {"clip": "worker-brick", "points": [Vector2(0.54, 0.64)]},
}
var game: Node
var clips: Dictionary = {}
var frames: Dictionary = {}
var walkers: Dictionary = {}
var sites: Dictionary = {}
var monuments: Dictionary = {}
var clock := 0.0
var last_tick := -1
var current_turn := 0


func setup(owner: Node) -> void:
	game = owner
	var manifest: Dictionary = JSON.parse_string(FileAccess.get_file_as_string(ROOT + "manifest.json"))
	clips = manifest.clips
	for name: String in clips:
		var spec: Dictionary = clips[name]
		var texture := load(ROOT + str(spec.path)) as Texture2D
		var atlas_frames: Array[Texture2D] = []
		for row in range(int(spec.rows)):
			for column in range(int(spec.columns)):
				var frame := AtlasTexture.new()
				frame.atlas = texture
				frame.region = Rect2(column * int(spec.cell_size), row * int(spec.cell_size), int(spec.cell_size), int(spec.cell_size))
				frame.filter_clip = true
				atlas_frames.append(frame)
		frames[name] = atlas_frames


func _pose(sprite: Sprite2D, clip: String, frame: int, mirrored: bool = false) -> void:
	var spec: Dictionary = clips[clip]
	var actual := posmod(frame, frames[clip].size())
	sprite.texture = frames[clip][actual]
	sprite.centered = false
	sprite.offset = -Vector2(float(spec.anchor[0]), float(spec.anchor[1]))
	sprite.scale = Vector2.ONE * float(spec.pixel_scale)
	sprite.skew = 0.0
	sprite.rotation = 0.0
	sprite.flip_h = mirrored
	sprite.texture_filter = CanvasItem.TEXTURE_FILTER_LINEAR_WITH_MIPMAPS
	sprite.set_meta("motion_clip", clip)
	sprite.set_meta("motion_frame", actual)


func _direction(heading: Vector2) -> int:
	var screen: Vector2 = game._iso(heading.x, heading.y) - game._iso(0, 0)
	if screen.y >= 0: return 0 if screen.x >= 0 else 1
	return 3 if screen.x >= 0 else 2


func _walking_clip(walker: Dictionary) -> String:
	match str(walker.role):
		"cart", "goldcart", "porter": return "haul-walk" if int(walker.get("cargo", 0)) > 0 else "haul-empty"
		"seller", "buyer": return "vendor-walk"
		"architect", "taxcollector", "firefighter": return "official-walk"
	return "worker-walk"


func sync() -> void:
	if game == null: return
	var tick := int(game.state.get("tick", 0))
	# A new/load operation invalidates visual interpolation, never simulation.
	if tick < last_tick:
		walkers.clear()
		clock = 0.0
	last_tick = tick
	for id: String in game.walker_nodes:
		var data: Dictionary = game.walker_data[id]
		var tile := Vector2(float(data.x), float(data.y))
		if not walkers.has(id): walkers[id] = {"tile": tile, "heading": Vector2(1, 0), "phase": float(int(id) % 4)}
		var track: Dictionary = walkers[id]
		if tile != track.tile:
			track.heading = tile - track.tile
			track.tile = tile
		if game.paused or current_turn != game.view_turns:
			game.walker_nodes[id].position = game.walker_targets[id]
		_apply_walker(id, false)
	for id: String in walkers.keys():
		if not game.walker_nodes.has(id): walkers.erase(id)
	current_turn = game.view_turns
	_sync_sites()
	_sync_monuments()


func _apply_walker(id: String, moving: bool) -> void:
	var sprite: Sprite2D = game.walker_nodes[id]
	var data: Dictionary = game.walker_data[id]
	var track: Dictionary = walkers[id]
	if data.role == "builder" and str(data.get("stage", "")) == "work":
		_pose(sprite, "worker-brick", int(clock * 5 + int(id)) % 4, _direction(track.heading) in [1, 2])
	else:
		var column := int(track.phase) % 4 if moving else 1
		_pose(sprite, _walking_clip(data), _direction(track.heading) * 4 + column)


func update(delta: float) -> void:
	if game == null: return
	if not game.paused: clock += delta * game.simulation_speed
	for id: String in game.walker_nodes:
		if not walkers.has(id): continue
		var sprite: Sprite2D = game.walker_nodes[id]
		var target: Vector2 = game.walker_targets[id]
		var before := sprite.position
		if not game.paused:
			var tile_distance: float = Vector2(game.tile_width * 0.5, game.tile_height * 0.5).length()
			sprite.position = sprite.position.move_toward(target, tile_distance * delta * game.simulation_speed / 0.2)
			walkers[id].phase += before.distance_to(sprite.position) / tile_distance * 4.0
		_apply_walker(id, before.distance_to(sprite.position) > 0.01)
	for id: String in sites:
		var site: Dictionary = sites[id]
		var active: bool = site.active
		for index in range(site.actors.size()):
			var sprite: Sprite2D = site.actors[index]
			var phase := int(clock * 4.5 + int(id) + index * 1.7) % 4 if active else 0
			_pose(sprite, site.clip, phase, index % 2 == 1)


func _site_active(data: Dictionary) -> bool:
	if int(data.get("workers", 0)) == 0 or str(data.get("damaged", "")) != "": return false
	match str(data.kind):
		"farm": return float(game.state.get("flood_level", 0)) < 0.35 and str(game.state.get("season", "")).to_lower() not in ["flood", "inundation"]
		"brickworks": return int(data.get("clay", 0)) > 0 and int(data.get("straw", 0)) > 0
		"bazaar", "granary": return int(data.get("stock", 0)) > 0
		"storageyard": return int(data.get("clay", 0)) + int(data.get("straw", 0)) + int(data.get("bricks", 0)) > 0
		# Grain capacity is zero at the pit; clay has its own 32-unit stock.
		"claypit": return int(data.get("clay", 0)) < 32
		"goldmine": return int(data.get("gold", 0)) < int(data.get("gold_capacity", 200))
	return true


func _sync_sites() -> void:
	var seen := {}
	for data: Dictionary in game.state.get("buildings", []):
		var kind := str(data.kind)
		if not WORKSITES.has(kind): continue
		var id := str(data.id)
		seen[id] = true
		var spec: Dictionary = WORKSITES[kind]
		if not sites.has(id):
			var group := Node2D.new()
			group.name = "Worksite_" + id
			game.buildings.add_child(group)
			var actors: Array[Sprite2D] = []
			for point: Vector2 in spec.points:
				var actor := Sprite2D.new()
				group.add_child(actor)
				actors.append(actor)
			sites[id] = {"group": group, "actors": actors, "clip": str(spec.clip), "active": false}
		var site: Dictionary = sites[id]
		var building: Sprite2D = game.building_nodes[id]
		site.group.position = building.position + Vector2(0, 0.2)
		site.group.visible = int(data.get("workers", 0)) > 0 and str(data.get("damaged", "")) == ""
		site.active = _site_active(data)
		for index in range(site.actors.size()):
			var point: Vector2 = spec.points[index]
			var at: Vector2 = building.to_global(building.offset + building.texture.get_size() * point)
			var sprite: Sprite2D = site.actors[index]
			sprite.position = at - site.group.global_position
			sprite.visible = index == 0 or int(data.workers) >= maxi(2, int(data.get("workers_needed", 2)) / 2)
			_pose(sprite, site.clip, int(clock * 4.5 + int(id) + index * 1.7) % 4 if site.active else 0, index % 2 == 1)
	for id: String in sites.keys():
		if not seen.has(id):
			sites[id].group.queue_free()
			sites.erase(id)


func _sync_monuments() -> void:
	var seen := {}
	for data: Dictionary in game.state.get("buildings", []):
		if data.kind != "mastaba": continue
		var id := str(data.id)
		seen[id] = true
		if not monuments.has(id):
			var sprite := Sprite2D.new()
			var material := ShaderMaterial.new()
			material.shader = preload("res://shaders/mastaba_courses.gdshader")
			sprite.material = material
			game.buildings.add_child(sprite)
			monuments[id] = sprite
		var sprite: Sprite2D = monuments[id]
		game._skin_sprite(sprite, "mastaba", game._front_corner(game._building_rect(data)) + Vector2(0, 0.1))
		var completed := int(data.get("construction_done", 0))
		sprite.visible = completed > 0 and completed < 64
		sprite.material.set_shader_parameter("completed", completed)
		# Foundation remains below the accumulated courses until the final cap.
		if completed < 64:
			var base: Sprite2D = game.building_nodes[id]
			game._skin_sprite(base, "mastaba_foundation", game._front_corner(game._building_rect(data)))
	for id: String in monuments.keys():
		if not seen.has(id):
			monuments[id].queue_free()
			monuments.erase(id)
