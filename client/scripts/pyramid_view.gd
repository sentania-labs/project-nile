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
	elif family=="obelisk":
		under=str(data.kind)+"_work"
		var prep:=4 if data.kind=="smallobelisk" else 8
		progress=maxf(0.0,float(data.get("construction_done",0)-prep)/float(6 if prep==4 else 10))
	elif family=="royaltomb":
		under="smallroyaltomb_work"
		progress=maxf(0.0,float(data.get("monument_phase",0)-4)/5.0)
	elif family=="mastaba":
		under=str(data.kind)+"_foundation"
		progress=float(installed.get("bricks",0))/maxf(1,float(required.get("bricks",1)))
	elif family=="suntemple":
		under="suntemple_work"
		progress=float(data.get("monument_percent",0))/100.0
	elif family=="mausoleum":
		under=str(data.kind)+"_foundation"
		progress=maxf(0.0,float(installed.get("sandstone",0)-24000)/25600.0)
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
	sprite.material.set_shader_parameter("carving",family in ["sphinx","obelisk","suntemple","mausoleum","royaltomb"])
	sprite.material.set_shader_parameter("courses",12.0 if data.kind=="largemastaba" else (6.0 if family=="mastaba" else float(data.width)*1.5))
	sprite.material.set_shader_parameter("front_x",0.2695 if family=="mastaba" else 0.5)
	sprite.material.set_shader_parameter("front_y",0.9453 if family=="mastaba" else 0.95)
	sprite.material.set_shader_parameter("left_slope",0.345 if family=="mastaba" else 0.46)
	sprite.material.set_shader_parameter("right_slope",0.292 if family=="mastaba" else 0.46)
	sprite.material.set_shader_parameter("wall_height",0.135 if family=="mastaba" else 0.72)

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
		game._skin_sprite(node,art,game._front_corner(rect),(float(part.width)+float(part.height))*48.0/0.9)
		for y in range(int(part.y),int(part.y)+int(part.height)):
			for x in range(int(part.x),int(part.x)+int(part.width)):game.blocked_tiles[Vector2i(x,y)]=true
		game.placements.append({"id":str(data.id),"asset":art,"spec":game.assets[art],"rect":rect,"sprite":node,"building":data})

func is_complex(kind: String) -> bool:
	return kind=="suntemple" or kind.begins_with("complex") or kind.begins_with("grand")

func preview_parts(cell: Vector2i,kind: String) -> Array[Dictionary]:
	var out: Array[Dictionary]=[]
	if not is_complex(kind):return out
	if kind=="suntemple":
		for y in range(10,18,2):out.append({"kind":"causeway","x":cell.x+6,"y":cell.y+y,"width":2,"height":2})
		out.append({"kind":"sun_foretemple","x":cell.x+5,"y":cell.y+18,"width":3,"height":3})
		return out
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
	if kind in ["smallobelisk","largeobelisk"]:
		var granite:=0
		for b: Dictionary in game.state.get("buildings",[]):
			if str(b.get("monument_family",""))=="obelisk" and b.get("stage","")!="complete":return "Finish the current obelisk first"
			if b.kind=="storageyard" and b.get("damaged","")=="":granite+=int(b.get("goods",{}).get("granite",0))
		var needed:=10000 if kind=="smallobelisk" else 20000
		if granite<needed:return "Store %d granite before placing this obelisk"%needed
	if kind in ["suntemple","smallmausoleum","mediummausoleum","largemausoleum"]:
		var stone:=0
		for b: Dictionary in game.state.get("buildings",[]):
			if kind=="suntemple" and b.kind=="suntemple" and b.get("stage","")=="complete":return "This city already has a completed Sun Temple"
			if b.kind=="storageyard" and b.get("damaged","")=="":stone+=int(b.get("goods",{}).get("sandstone",0))
		var needed:=22000 if kind=="suntemple" else 24000
		if stone<needed:return "Store %d sandstone blocks before placing this monument"%(needed/100)
	if not is_complex(kind):return ""
	var planned:=preview_parts(cell,kind)
	if planned.is_empty():return "Place the complex west of the Nile, with space for a causeway to an east-facing bank"
	for p: Dictionary in planned:
		for y in range(int(p.y),int(p.y)+int(p.height)):
			for x in range(int(p.x),int(p.x)+int(p.width)):
				if kind=="suntemple" and (x<0 or y<0 or x>=game.map_width or y>=game.map_height or game.geography.water_at(x,y)):return "The temple and processional way need dry land"
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
		game._skin_sprite(node,str(p.kind),game._front_corner(Rect2(float(p.x),float(p.y),float(p.width),float(p.height))))
		node.modulate=game.ghost.modulate

func inspect(data: Dictionary) -> void:
	var stage:=str(data.get("stage","leveling"))
	game.kind_label.text=stage.capitalize()
	var lines: Array[String]=[]
	var installed: Dictionary=data.get("monument_materials",{})
	var required: Dictionary=data.get("monument_required_materials",{})
	for resource in ["stone","bricks","limestone","granite","sandstone","timber","lamps","clay","paint"]:
		if int(required.get(resource,0))>0:
			var unit:="units" if resource in ["bricks","lamps","clay","paint"] else ("loads" if resource=="timber" else "blocks")
			var divisor:=1 if resource in ["bricks","lamps","clay","paint"] else 100
			lines.append("%s: %d / %d %s installed"%[resource.capitalize(),int(installed.get(resource,0))/divisor,int(required[resource])/divisor,unit])
	lines.append("Crew: %d on site · %d approaching"%[int(data.get("crew_on_site",0)),int(data.get("crew_en_route",0))])
	if stage=="complete":lines.append("The monument is complete.")
	elif not data.get("staging_access",false):lines.append("Connect a road to the marked staging area.")
	elif str(data.get("monument_family",""))=="royaltomb":lines.append("Work Camp laborers deliver lamps from storage. Masons excavate the cliff; artisans bring paint and plaster supplies from their guild.")
	elif str(data.get("monument_family",""))=="obelisk":lines.append("Granite was installed when the site was placed. Carpenters bring scaffolding timber, then masons carve the shaft.")
	elif data.kind=="sphinx":lines.append("Masons carve the bedrock. Carpenters bring timber for scaffolding; no stone deliveries are needed.")
	elif data.get("monument_family","")=="mastaba" and str(data.get("monument_guild",""))=="bricklayers":lines.append("Bricklayers lay the courses. Work Camp sledges bring 400 bricks at a time from storage.")
	elif stage=="polishing":lines.append("Masons polish the limestone. No further stone is needed.")
	elif str(data.get("monument_guild",""))=="workcamp":lines.append("Work Camp laborers prepare the foundation.")
	elif str(data.get("monument_guild",""))=="carpenters":lines.append("Carpenters build access ramps. Supply their guild with timber.")
	else:lines.append("Paired guild workers install blocks. Work Camp sledges bring four loads from storage.")
	game.detail_label.text="\n\n".join(lines)
	game.production_bar.visible=true
	game.production_bar.max_value=100
	game.production_bar.value=int(data.get("monument_percent",0))

func inspect_walker(walker: Dictionary) -> void:
	if walker.role=="stonehauler":
		game.title_label.text="Lamp delivery" if walker.get("resource","")=="lamps" else "Monument material sledge"
		game.detail_label.text="Work Camp laborers collect reserved cargo from storage, bring it to the staging area, then return to their camp."
	elif walker.role=="builder":
		var guild: Dictionary={}
		for building: Dictionary in game.state.get("buildings",[]):
			if building.id==walker.origin:guild=building;break
		if guild.get("kind","")=="artisans":
			game.title_label.text="Royal Tomb artisan"
			game.detail_label.text="Bringing paint and plaster supplies from the guild, then decorating the excavated chambers."
		elif guild.get("kind","")!="hut":
			game.detail_label.text="Travelling from the guild or Work Camp to the monument's staging area. Work starts after arrival."
		var supplies: Dictionary=walker.get("supplies",{})
		if not supplies.is_empty():
			var names: Array=supplies.keys()
			names.sort()
			var lines: Array[String]=[]
			for name: String in names:lines.append("%d %s"%[int(supplies[name]),name])
			game.detail_label.text+="\n\nCarrying: "+", ".join(lines)
