extends VBoxContainer
## River presentation and commands; every voyage and cargo quantity belongs to Go.
var game: Node
var summary := Label.new()
var route: Button
var vessels := OptionButton.new()
var x := SpinBox.new()
var y := SpinBox.new()
var sail: Button
var recall: Button
var boats: Dictionary = {}
var boat_ids: Array[int] = []
var last_tick := -1
var last_turn := -1

func setup(owner: Node) -> void:
	game = owner
	summary.autowrap_mode = TextServer.AUTOWRAP_WORD_SMART
	add_child(summary)
	route = game._button("Open Nile merchant route · 300 Db", func() -> void: game._queue_command("open_trade", {"route":"nile_route"}))
	add_child(route)
	add_child(vessels)
	var row := HBoxContainer.new()
	add_child(row)
	for picker: SpinBox in [x,y]:
		picker.min_value=0
		picker.max_value=31
		picker.step=1
		row.add_child(picker)
	x.prefix="X "
	y.prefix="Y "
	x.value=2
	y.value=24
	sail=game._button("Sail and hold",func() -> void: _order("hold"))
	recall=game._button("Return to wharf",func() -> void: _order("return"))
	row.add_child(sail)
	row.add_child(recall)
	var hint: Label = game._label("Build shore facilities across a straight riverbank. A Shipwright supplies fishing boats without timber; military ships need 400 timber. Dock carts carry imports to storage and fetch exports. Set resource orders with the Trade overseer.",13,Color("392617"))
	hint.autowrap_mode=TextServer.AUTOWRAP_WORD_SMART
	add_child(hint)
	hide()

func _order(action: String) -> void:
	if vessels.selected < 0: return
	game._queue_command("naval_order",{"boat_id":vessels.get_selected_metadata(),"order":action,"x":int(x.value),"y":int(y.value)})

func sync(topic: String) -> void:
	var entries: Array=game.state.get("boats",[])
	var seen := {}
	var tick:=int(game.state.get("tick",0))
	for data: Dictionary in entries:
		var id:=str(data.id)
		seen[id]=true
		if not boats.has(id):
			var sprite:=Sprite2D.new()
			game.buildings.add_child(sprite)
			boats[id]={"sprite":sprite,"tile":Vector2(float(data.x),float(data.y)),"target":Vector2.ZERO,"heading":Vector2(0,1)}
		var track: Dictionary=boats[id]
		var tile:=Vector2(float(data.x),float(data.y))
		if tile!=track.tile: track.heading=tile-track.tile
		track.tile=tile
		track.target=game._iso(tile.x+0.5,tile.y+0.5)
		var asset:=str(data.kind)
		if asset=="transportship": asset="merchantship"
		if not game.assets.has(asset): continue
		var actor: Sprite2D=track.sprite
		var position: Vector2=actor.position
		if game.paused or tick<last_tick or last_turn!=game.view_turns or position==Vector2.ZERO: position=track.target
		else: position=position.move_toward(track.target, 260.0*get_process_delta_time()*game.simulation_speed)
		game._skin_sprite(actor,asset,position)
		var heading: Vector2=game._iso(track.heading.x,track.heading.y)-game._iso(0,0)
		actor.flip_h=heading.x<0
		actor.visible=str(data.stage)!="away"
		actor.modulate=Color("e6d9ba") if data.kind=="transportship" else Color.WHITE
	for id: String in boats.keys():
		if not seen.has(id):
			boats[id].sprite.queue_free()
			boats.erase(id)
	last_tick=tick
	last_turn=game.view_turns
	visible=topic=="River"
	if not visible: return
	var military: Array=[]
	var ids: Array[int]=[]
	for b: Dictionary in entries:
		if b.kind in ["warship","transportship"]:
			military.append(b)
			ids.append(int(b.id))
	if ids!=boat_ids:
		boat_ids=ids
		vessels.clear()
		for id: int in ids:
			vessels.add_item("")
			vessels.set_item_metadata(vessels.item_count-1,id)
	for i in range(military.size()):
		var b: Dictionary=military[i]
		vessels.set_item_text(i,"%s %d · %s" % [str(b.kind).capitalize(),int(b.id),str(b.stage)])
	sail.disabled=military.is_empty()
	recall.disabled=military.is_empty()
	x.max_value=int(game.state.get("map_extent",32))-1
	y.max_value=x.max_value
	var trade: Dictionary={}
	for r: Dictionary in game.state.get("trade_routes",[]):
		if r.id=="nile_route": trade=r
	route.visible=bool(trade.get("available",false)) and not bool(trade.get("open",false))
	var lines: Array[String]=["Ships built: %d · Fish caught: %d" % [int(game.state.get("ships_built",0)),int(game.state.get("resource_produced",{}).get("fish",0))]]
	for b: Dictionary in entries:
		lines.append("%s %d: %s%s" % [str(b.kind).capitalize(),int(b.id),str(b.stage), " · %d %s" % [int(b.cargo),str(b.resource)] if int(b.cargo)>0 else ""])
	summary.text="\n".join(lines)
