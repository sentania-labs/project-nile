extends RefCounted

# One complex per city. Cache its paved court, sanctuary and processional
# avenue until a construction, camera turn, or load changes their appearance.
var parts: Array[Node2D] = []
var last_layout := ""

func sync(game: Node) -> void:
	var site: Dictionary={}
	for b: Dictionary in game.state.get("buildings", []):
		if str(b.kind).begins_with("complex_"):
			site=b
			break
	var key:="" if site.is_empty() else "%s,%s,%s,%s,%s,%s" % [site.kind,site.x,site.y,site.get("facing",0),site.get("temple_upgrades",0),game.view_turns]
	if not site.is_empty(): game.building_nodes[str(site.id)].visible=false
	if key==last_layout: return
	last_layout=key
	for part in parts: part.queue_free()
	parts.clear()
	if site.is_empty(): return
	var court:=Polygon2D.new()
	court.polygon=game._footprint(game._building_rect(site))
	court.color=Color("c8af85")
	court.z_index=-2
	game.buildings.add_child(court)
	parts.append(court)
	# Set the central processional path in alternating limestone courses.
	for y in range(1,12):
		for x in range(1,6):
			var cell:=Vector2i(x,y)
			match int(site.get("facing",0)):
				1: cell=Vector2i(12-y,x)
				2: cell=Vector2i(6-x,12-y)
				3: cell=Vector2i(y,6-x)
			var flagstone:=Polygon2D.new()
			var tile:=Rect2(Vector2(site.x,site.y)+Vector2(cell)+Vector2(.025,.025),Vector2(.95,.95))
			flagstone.polygon=game._footprint(tile)
			var edge:= x==1 or x==5
			flagstone.color=[Color("bb9a75"),Color("c3a27d")][y%2] if edge else [Color("d9bc92"),Color("d2b58c"),Color("d7ba95")][(x*2+y*7)%3]
			flagstone.z_index=-1
			game.buildings.add_child(flagstone)
			parts.append(flagstone)
	# Rows of statuary connect the entry to the main sanctuary.
	for y in [2,4,6,8,10]:
		for x in [0,6]:
			_place(game,site,"smallstatue",Vector2i(x,y),1)
	var upgrades:=int(site.get("temple_upgrades",0))
	_place(game,site,"complexsanctuary",Vector2i(2,0),3)
	_place(game,site,"complexaltar" if upgrades&1 else "temple_"+str(site.kind).trim_prefix("complex_"),Vector2i(2,4),3)
	_place(game,site,"complexoracle" if upgrades&2 else "garden",Vector2i(2,8),3)

func _place(game: Node,site: Dictionary,art: String,cell: Vector2i,size: int) -> void:
	var offset:=cell
	match int(site.get("facing",0)):
		1: offset=Vector2i(13-cell.y-size,cell.x)
		2: offset=Vector2i(7-cell.x-size,13-cell.y-size)
		3: offset=Vector2i(cell.y,7-cell.x-size)
	var rect:=Rect2(Vector2(int(site.x)+offset.x,int(site.y)+offset.y),Vector2(size,size))
	var sprite:=Sprite2D.new()
	game.buildings.add_child(sprite)
	game._skin_sprite(sprite,art,game._front_corner(rect),game.tile_width*size*0.90)
	parts.append(sprite)
