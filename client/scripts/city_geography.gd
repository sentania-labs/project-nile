extends RefCounted

var patches: Array[Polygon2D] = []
var original: Array[Polygon2D] = []
var extent := 32
var banks: Array = []
var west: Array = []


func setup(game: Node) -> void:
	for i in 4: original.append(game.ground.get_child(i))


func bank_at(y: int) -> int:
	return int(banks[clampi(y,0,banks.size()-1)]) if not banks.is_empty() else 6


func west_at(y: int) -> int:
	return int(west[clampi(y,0,west.size()-1)]) if not west.is_empty() else 0


func water_at(x: int, y: int) -> bool:
	return x >= west_at(y) and x < bank_at(y)


func sync(game: Node, state: Dictionary) -> void:
	var next_extent := int(state.get("map_extent",32))
	var next_banks: Array = state.get("banks",[])
	var next_west: Array = state.get("west_banks",[])
	if extent==next_extent and banks==next_banks and west==next_west: return
	west = next_west
	extent = next_extent
	banks = next_banks
	game.map_width = extent
	game.map_height = extent
	for patch in patches: patch.free()
	patches.clear()
	for polygon in original: polygon.visible = banks.is_empty()
	game.river_material.set_shader_parameter("bank_count",banks.size())
	game.river_material.set_shader_parameter("two_banks",not west.is_empty())
	if banks.is_empty(): return
	var heights := PackedFloat32Array()
	heights.resize(228)
	for y in banks.size(): heights[y] = float(banks[y])
	game.river_material.set_shader_parameter("riverbanks",heights.duplicate())
	for y in west.size(): heights[y] = float(west[y])
	game.river_material.set_shader_parameter("westbanks",heights)
	_add(game,"sand",game._footprint(Rect2(-64,-64,extent+128,extent+128)))
	for kind: String in ["grass","silt"]:
		var margin := 4.0 if kind=="silt" else 8.8
		for y in range(0,extent,16):
			var end := mini(extent,y+16)
			var points := PackedVector2Array()
			for row in range(y,end+1):
				var edge := 0.55*sin(row*0.37)+0.26*sin(row*1.53) if kind=="grass" else 0.11*sin(row*0.61)
				points.append(game._iso(bank_at(row)+margin+edge,row))
			for row in range(end,y-1,-1): points.append(game._iso(-64 if west.is_empty() else west_at(row)-margin,row))
			_add(game,kind,points)
	_add(game,"water",game._footprint(Rect2(-64,-64,extent+128,extent+128)))
	patches[-1].material = game.river_material


func _add(game: Node, kind: String, points: PackedVector2Array) -> void:
	game._add_terrain_polygon(kind,points)
	var patch: Polygon2D = game.ground.get_child(-1)
	game.ground.move_child(patch,original.size()+patches.size())
	patches.append(patch)
