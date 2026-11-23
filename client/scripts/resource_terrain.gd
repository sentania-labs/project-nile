extends RefCounted

var meadow: Dictionary = {}
var rock: Dictionary = {}
var meadow_tiles: Array = []
var rock_tiles: Array = []
var patches: Array[Polygon2D] = []
var deposits: Array[Sprite2D] = []


func sync(game: Node, state: Dictionary) -> void:
	_sync_scenery(game, state)
	_sync_canals(game,state)
	var grass: Array = state.get("grassland", [])
	var ore: Array = state.get("ore_rock", [])
	if grass != meadow_tiles:
		meadow_tiles = grass
		meadow.clear()
		for patch in patches: patch.free()
		patches.clear()
		for cell: Dictionary in grass: meadow[Vector2i(cell.x, cell.y)] = true
		var remaining := meadow.duplicate()
		for cell: Vector2i in meadow:
			if not game.geography.banks.is_empty(): break
			if not remaining.has(cell): continue
			var origin := cell
			while remaining.has(origin + Vector2i.LEFT): origin.x -= 1
			while remaining.has(origin + Vector2i.UP): origin.y -= 1
			var width := 1
			while remaining.has(origin + Vector2i(width, 0)): width += 1
			var height := 1
			while _has_row(remaining, origin + Vector2i(0, height), width): height += 1
			for y in range(height):
				for x in range(width): remaining.erase(origin + Vector2i(x, y))
			game._add_terrain_polygon("grass", game._footprint(Rect2(origin, Vector2i(width, height))), 5)
			var patch: Polygon2D = game.ground.get_child(-1)
			game.ground.move_child(patch, 1 if game.geography.banks.is_empty() else game.geography.original.size() + game.geography.patches.size())
			patches.append(patch)
	if ore != rock_tiles:
		rock_tiles = ore
		rock.clear()
		for deposit in deposits: deposit.free()
		deposits.clear()
		for cell: Dictionary in ore:
			var tile := Vector2i(cell.x, cell.y)
			rock[tile] = true
			var deposit := Sprite2D.new()
			game._skin_sprite(deposit, "gold_outcrop", Vector2.ZERO, 86 + posmod(tile.x * 7 + tile.y * 11, 12))
			deposit.flip_h = (tile.x + tile.y) % 2 == 1
			game.buildings.add_child(deposit)
			deposits.append(deposit)
	for i in deposits.size():
		var cell: Dictionary = ore[i]
		deposits[i].position = game._front_corner(Rect2(cell.x, cell.y, 1, 1))


func _has_row(cells: Dictionary, at: Vector2i, width: int) -> bool:
	for x in range(width):
		if not cells.has(at + Vector2i(x, 0)): return false
	return true


func near_deposit(origin: Vector2i, size: Vector2i, quarry := false) -> bool:
	var deposits_map := quarry_tiles if quarry else rock
	for dx in size.x:
		if deposits_map.has(origin + Vector2i(dx, -1)) or deposits_map.has(origin + Vector2i(dx, size.y)): return true
	for dy in size.y:
		if deposits_map.has(origin + Vector2i(-1, dy)) or deposits_map.has(origin + Vector2i(size.x, dy)): return true
	return false


func has_grass(origin: Vector2i, size: Vector2i) -> bool:
	for dx in size.x:
		for dy in size.y:
			if meadow.has(origin + Vector2i(dx, dy)): return true
	return false


var scenery_nodes: Dictionary = {}
var features: Dictionary = {}
var canals: Dictionary = {}
var canal_nodes: Array[Line2D] = []
var canal_key := ""
var spans: Dictionary = {}
var obstacles: Dictionary = {}
var quarry_tiles: Dictionary = {}
var wildlife_tiles: Dictionary = {}


func _sync_scenery(game: Node, state: Dictionary) -> void:
	var seen: Dictionary = {}
	spans.clear()
	for span: Dictionary in state.get("bridges",[]):
		for x in range(int(span.west),int(span.east)+1):
			var tile := Vector2i(x,int(span.y))
			spans[tile]=span
			var key := "bridge/%s" % tile
			seen[key]=true
			if not scenery_nodes.has(key):
				var deck := Sprite2D.new()
				game._skin_sprite(deck,"bridge",Vector2.ZERO,101)
				game.buildings.add_child(deck)
				scenery_nodes[key]=deck
			scenery_nodes[key].position = game._front_corner(Rect2(tile,Vector2i.ONE))
	features.clear()
	obstacles.clear()
	quarry_tiles.clear()
	wildlife_tiles.clear()
	for layer: String in ["reed_beds", "hunting_grounds", "road_features", "woodland","quarry_rock"]:
		for item: Dictionary in state.get(layer, []):
			var tile := Vector2i(item.x, item.y)
			var kind := str(item.get("kind", "reedbed" if layer == "reed_beds" else "gazelle"))
			if layer in ["woodland","quarry_rock"]:
				kind = "woodland" if layer=="woodland" else "quarryrock"
				obstacles[tile]=true
				if layer=="quarry_rock": quarry_tiles[tile]=true
			if layer == "road_features": features[tile] = item
			if layer == "hunting_grounds": wildlife_tiles[tile] = true
			var id := "%s/%d/%d" % [kind, tile.x, tile.y]
			seen[id] = true
			var sprite: Sprite2D = scenery_nodes.get(id)
			if sprite == null:
				sprite = Sprite2D.new()
				game._skin_sprite(sprite, kind, Vector2.ZERO)
				game.buildings.add_child(sprite)
				scenery_nodes[id] = sprite
			sprite.position = game._front_corner(Rect2(tile, Vector2i.ONE))
			sprite.visible = layer != "reed_beds" or game._building_at(tile).is_empty()
	for id: String in scenery_nodes.keys():
		if not seen.has(id):
			scenery_nodes[id].free()
			scenery_nodes.erase(id)

func _sync_canals(game: Node, state: Dictionary) -> void:
	var tiles: Array = state.get("canals",[])
	var watered: Array = state.get("watered_canals",[])
	var signature := "%s/%s/%s" % [game.view_turns,tiles,watered]
	if signature == canal_key: return
	canal_key = signature
	for node in canal_nodes: node.free()
	canal_nodes.clear()
	canals.clear()
	for item: Dictionary in tiles: canals[Vector2i(item.x,item.y)] = false
	for item: Dictionary in watered: canals[Vector2i(item.x,item.y)] = true
	for tile: Vector2i in canals:
		var branches: Array[Vector2i] = []
		for offset: Vector2i in [Vector2i.LEFT,Vector2i.RIGHT,Vector2i.UP,Vector2i.DOWN]:
			if canals.has(tile+offset): branches.append(offset)
		if branches.is_empty(): branches.assign([Vector2i.LEFT,Vector2i.RIGHT])
		for offset in branches:
			for layer in 2:
				var line := Line2D.new()
				line.width = 7.0 if layer==0 else 3.5
				line.default_color = Color("6f5534") if layer==0 else (Color("528c91") if canals[tile] else Color("b9946a"))
				line.points = PackedVector2Array([game._iso(tile.x+0.5,tile.y+0.5),game._iso(tile.x+0.5+0.50*offset.x,tile.y+0.5+0.50*offset.y)])
				game.ground.add_child(line)
				canal_nodes.append(line)
