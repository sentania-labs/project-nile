extends RefCounted
## Cliff terrain is authoritative. Only the monument can occupy its rock tiles.
var cliffs: Dictionary = {}
var nodes: Dictionary = {}
var cliffs_visible := true

func set_cliffs_visible(value: bool) -> void:
	cliffs_visible=value
	for node: Sprite2D in nodes.values():node.visible=value

func sync(game: Node) -> void:
	cliffs.clear()
	for tile: Dictionary in game.state.get("cliffs",[]):cliffs[Vector2i(tile.x,tile.y)]=true
	var covered: Dictionary = {}
	for b: Dictionary in game.state.get("buildings",[]):
		if b.get("monument_family","")!="royaltomb":continue
		for y in range(int(b.y),int(b.y)+int(b.height)):
			for x in range(int(b.x),int(b.x)+int(b.width)):covered[Vector2i(x,y)]=true
	var seen: Dictionary = {}
	var used: Dictionary = {}
	for cell: Vector2i in cliffs:
		if covered.has(cell) or used.has(cell):continue
		var side:=1
		for candidate in [4,2]:
			var fits:=true
			for y in candidate:
				for x in candidate:
					if not cliffs.has(cell+Vector2i(x,y)) or covered.has(cell+Vector2i(x,y)) or used.has(cell+Vector2i(x,y)):fits=false
			if fits:
				side=candidate
				break
		for y in side:
			for x in side:used[cell+Vector2i(x,y)]=true
		seen[cell]=true
		if not nodes.has(cell):
			var node:=Sprite2D.new()
			game.buildings.add_child(node)
			nodes[cell]=node
		nodes[cell].visible=cliffs_visible
		game._skin_sprite(nodes[cell],"cliffrock",game._front_corner(Rect2(cell,Vector2i(side,side))),float(side)*96.0)
	for cell: Vector2i in nodes.keys():
		if not seen.has(cell):
			nodes[cell].queue_free()
			nodes.erase(cell)

func placement_error(game: Node,cell: Vector2i,kind: String) -> String:
	if kind!="smallroyaltomb":return ""
	for y in 4:
		for x in 4:
			if not cliffs.has(cell+Vector2i(x,y)):return "Place the Royal Tomb in a cliff face"
	var stage:=cell+Vector2i(2,4)
	if cliffs.has(stage) or game.geography.water_at(stage.x,stage.y):return "The entrance needs open ground for a staging road"
	return ""
