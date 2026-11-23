extends RefCounted
## Construction visuals are keyed to delivered and installed stone, never time.
var game: Node
var sprites: Dictionary = {}
var markers: Dictionary = {}
var parts: Dictionary = {}
var ghost_parts: Array[Sprite2D] = []

func setup(owner: Node) -> void:
	game = owner
	var manifest: Dictionary = JSON.parse_string(FileAccess.get_file_as_string("res://assets/monuments/manifest.json"))
	for key: String in manifest: game.assets[key] = manifest[key]

func sync() -> void:
	var seen := {}
	for data: Dictionary in game.state.get("buildings", []):
		if str(data.get("monument_family", "")).is_empty(): continue
		var id := str(data.id)
		seen[id] = true
		if not sprites.has(id):
			var sprite := Sprite2D.new()
			var material := ShaderMaterial.new()
			material.shader = preload("res://shaders/pyramid_courses.gdshader")
			sprite.material = material
			game.buildings.add_child(sprite)
			sprites[id] = sprite
			var marker := Sprite2D.new()
			game.buildings.add_child(marker)
			markers[id] = marker
		var sprite: Sprite2D = sprites[id]
		var completed := int(data.get("construction_done", 0))
		if data.kind != "smallstepped":
			_sync_family(data, sprite)
			_sync_parts(data)
			_sync_marker(data, markers[id])
			continue
		var at: Vector2 = game._front_corner(game._building_rect(data))
		game._skin_sprite(sprite, "smallstepped_base" if completed < 19200 else "smallstepped", at + Vector2(0, 0.1))
		sprite.visible = completed > 0 and completed < 24000
		sprite.material.set_shader_parameter("completed", completed)
		sprite.material.set_shader_parameter("base_only", completed < 19200)
		if completed < 24000:
			game._skin_sprite(game.building_nodes[id], "smallstepped_foundation" if completed < 19200 else "smallstepped_base", at)
		# A visible staging sign marks the one road connection required by workers.
		var staging := Vector2(float(data.x) + 2.5, float(data.y) + 8.5)
		var marker: Sprite2D = markers[id]
		game._skin_sprite(marker, "vacant_plot", game._iso(staging.x, staging.y), 48)
		marker.modulate = Color(1, 1, 1, 0.8) if data.get("staging_access", false) else Color(1, 0.6, 0.4, 1)
		marker.visible = completed < 24000
	for key: String in parts.keys():
		if not seen.has(key.get_slice(":",0)):
			parts[key].queue_free()
			parts.erase(key)
	for id: String in sprites.keys():
		if not seen.has(id):
			sprites[id].queue_free()
			markers[id].queue_free()
			sprites.erase(id)
			markers.erase(id)

func _sync_marker(data: Dictionary, marker: Sprite2D) -> void:
	var p: Dictionary=data.get("monument_staging",{})
	var at:=Vector2(float(p.get("x",data.x)),float(p.get("y",data.y)))+Vector2(0.5,0.5)
	game._skin_sprite(marker,"vacant_plot",game._iso(at.x,at.y),48)
	marker.modulate=Color.WHITE if data.get("staging_access",false) else Color(1,0.6,0.4)
	marker.visible=data.get("stage","")!="complete"

func _sync_family(data: Dictionary, sprite: Sprite2D) -> void:
	var base: Sprite2D=game.building_nodes[str(data.id)]
	var at: Vector2=game._front_corner(game._building_rect(data))
	var width: float=game.assets[str(data.kind)].draw_width
	var family:=str(data.monument_family)
	var installed: Dictionary=data.get("monument_materials",{})
	var required: Dictionary=data.get("monument_required_materials",{})
	var complete:=str(data.get("stage",""))=="complete"
	if sprite.material.shader!=preload("res://shaders/monument_layers.gdshader"):
		sprite.material=ShaderMaterial.new()
		sprite.material.shader=preload("res://shaders/monument_layers.gdshader")
	sprite.visible=not complete
	if complete:return
	var progress:=0.0
	var art:=str(data.kind)
	var under:="pyramid_foundation"
	if family=="sphinx":
		under="sphinx_work"
		progress=float(data.get("construction_done",0))/22.0
	elif family=="stepped":
		progress=float(installed.get("stone",0))/maxf(1,float(required.get("stone",1)))
	else:
		var core:="bricks" if family=="brick" else "stone"
		if int(installed.get(core,0))>=int(required.get(core,1)):
			under="brick-core" if family=="brick" else "stone-core"
			progress=float(installed.get("limestone",0))/maxf(1,float(required.get("limestone",1)))
		else:
			art="brick-core" if family=="brick" else "stone-core"
			progress=float(installed.get(core,0))/maxf(1,float(required.get(core,1)))
	game._skin_sprite(base,under,at,width)
	game._skin_sprite(sprite,art,at+Vector2(0,0.1),width)
	sprite.material.set_shader_parameter("progress",progress)
	sprite.material.set_shader_parameter("carving",family=="sphinx")
	sprite.material.set_shader_parameter("courses",float(data.width)*1.5)

func _sync_parts(data: Dictionary) -> void:
	var index:=0
	for part: Dictionary in data.get("monument_parts",[]):
		var key:="%s:%d"%[str(data.id),index]
		index+=1
		if not parts.has(key):
			var node:=Sprite2D.new()
			game.buildings.add_child(node)
			parts[key]=node
		var node: Sprite2D=parts[key]
		var rect:=Rect2(float(part.x),float(part.y),float(part.width),float(part.height))
		var art:=str(part.kind) if int(data.get("monument_percent",0))>=90 else "pyramid_foundation"
		game._skin_sprite(node,art,game._front_corner(rect),213.5)
		for y in range(int(part.y),int(part.y)+int(part.height)):
			for x in range(int(part.x),int(part.x)+int(part.width)):game.blocked_tiles[Vector2i(x,y)]=true
		game.placements.append({"id":str(data.id),"asset":art,"spec":game.assets[art],"rect":rect,"sprite":node,"building":data})

func is_complex(kind: String) -> bool:
	return kind.begins_with("complex") or kind.begins_with("grand")

func preview_parts(cell: Vector2i,kind: String) -> Array[Dictionary]:
	var out: Array[Dictionary]=[]
	if not is_complex(kind):return out
	var size: Vector2i=game._building_size(kind)
	var x:=cell.x+size.x
	var y:=cell.y+size.y/2
	out.append({"kind":"mortuary_temple","x":x,"y":y,"width":2,"height":2})
	x+=2
	for count in 22:
		var west: bool=not game.geography.water_at(x,y) and not game.geography.water_at(x,y+1)
		var east: bool=game.geography.water_at(x+1,y) and game.geography.water_at(x+1,y+1)
		if west and east:
			if count==0:return []
			out.append({"kind":"valley_temple","x":x,"y":y,"width":2,"height":2})
			return out
		if not west or game.geography.water_at(x+1,y) or game.geography.water_at(x+1,y+1) or x+1>=game.map_width:return []
		out.append({"kind":"causeway","x":x,"y":y,"width":2,"height":2})
		x+=2
	return []

func placement_error(cell: Vector2i,kind: String) -> String:
	if kind=="sphinx" and game._has_building_kind("sphinx"):return "This city already has a Sphinx"
	if not is_complex(kind):return ""
	var planned:=preview_parts(cell,kind)
	if planned.is_empty():return "Place the complex west of the Nile, with space for a causeway to an east-facing bank"
	for p: Dictionary in planned:
		for y in range(int(p.y),int(p.y)+2):
			for x in range(int(p.x),int(p.x)+2):
				if game.blocked_tiles.has(Vector2i(x,y)) or game.resources.canals.has(Vector2i(x,y)):return "The temple or causeway crosses an occupied site"
	return ""

func preview() -> void:
	var planned: Array[Dictionary] = []
	if game.ghost.visible: planned = preview_parts(game.hover_cell,game.active_tool)
	while ghost_parts.size()<planned.size():
		var node:=Sprite2D.new()
		node.z_index=20
		game.buildings.add_child(node)
		ghost_parts.append(node)
	for i in ghost_parts.size():
		var node:=ghost_parts[i]
		node.visible=i<planned.size()
		if not node.visible:continue
		var p: Dictionary=planned[i]
		game._skin_sprite(node,str(p.kind),game._front_corner(Rect2(float(p.x),float(p.y),2,2)))
		node.modulate=game.ghost.modulate

func inspect(data: Dictionary) -> void:
	var stage:=str(data.get("stage","leveling"))
	game.kind_label.text=stage.capitalize()
	var lines: Array[String]=[]
	var installed: Dictionary=data.get("monument_materials",{})
	var required: Dictionary=data.get("monument_required_materials",{})
	for resource in ["stone","bricks","limestone","timber"]:
		if int(required.get(resource,0))>0:
			lines.append("%s: %d / %d loads installed"%[resource.capitalize(),int(installed.get(resource,0))/100,int(required[resource])/100])
	lines.append("Crew: %d on site · %d approaching"%[int(data.get("crew_on_site",0)),int(data.get("crew_en_route",0))])
	if stage=="complete":lines.append("The monument is complete.")
	elif not data.get("staging_access",false):lines.append("Connect a road to the marked staging area.")
	elif data.kind=="sphinx":lines.append("Masons carve the bedrock. Carpenters bring timber for scaffolding; no stone deliveries are needed.")
	elif stage=="polishing":lines.append("Masons polish the limestone. No further stone is needed.")
	elif str(data.get("monument_guild",""))=="workcamp":lines.append("Work Camp laborers prepare the foundation.")
	elif str(data.get("monument_guild",""))=="carpenters":lines.append("Carpenters build access ramps. Supply their guild with timber.")
	else:lines.append("Paired guild workers install blocks. Work Camp sledges bring four loads from storage.")
	game.detail_label.text="\n\n".join(lines)
	game.production_bar.visible=true
	game.production_bar.max_value=100
	game.production_bar.value=int(data.get("monument_percent",0))
