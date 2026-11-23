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
var engage: Button
var seek: Button
var repair: Button
var raid: Button
var evade: Button
var company_picker:=OptionButton.new()
var transport: Button
var sail_pick: Button
var transport_pick: Button
var picking: Dictionary={}
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
	sail_pick=game._button("Choose sailing destination",func() -> void: _pick(false))
	row.add_child(sail_pick)
	var orders:=HBoxContainer.new()
	add_child(orders)
	engage=game._button("Engage nearby",func() -> void: _order("engage"))
	seek=game._button("Seek enemies",func() -> void: _order("seek"))
	repair=game._button("Shipwright repair",func() -> void: _order("repair"))
	for button: Button in [engage,seek,repair]: orders.add_child(button)
	evade=game._button("Evade enemies",func() -> void: _order("evade"))
	orders.add_child(evade)
	var transport_row:=HBoxContainer.new()
	add_child(transport_row)
	transport_row.add_child(company_picker)
	transport=game._button("Carry company to shore X/Y",func() -> void:
		if vessels.selected>=0 and company_picker.selected>=0:
			game._queue_command("transport_company",{"boat_id":vessels.get_selected_metadata(),"fort_id":int(company_picker.get_selected_metadata()),"x":int(x.value),"y":int(y.value)}))
	transport_row.add_child(transport)
	transport_pick=game._button("Choose landing shore",func() -> void:_pick(true))
	add_child(transport_pick)
	raid=game._button("Test sea invasion",func() -> void: game._queue_command("test_sea_invasion"))
	add_child(raid)
	var hint: Label = game._label("Build shore facilities across a straight riverbank. A Shipwright supplies fishing boats without timber; military ships need 400 timber. Dock carts carry imports to storage and fetch exports. Set resource orders with the Trade overseer.",13,Color("392617"))
	hint.autowrap_mode=TextServer.AUTOWRAP_WORD_SMART
	add_child(hint)
	hide()

func _order(action: String) -> void:
	if vessels.selected < 0: return
	game._queue_command("naval_order",{"boat_id":vessels.get_selected_metadata(),"order":action,"x":int(x.value),"y":int(y.value)})

func _pick(carry: bool) -> void:
	if vessels.selected<0 or carry and company_picker.selected<0:return
	picking={"boat_id":vessels.get_selected_metadata(),"order":"hold"}
	if carry:picking["fort_id"]=int(company_picker.get_selected_metadata())
	game.city_ui.overseer_modal.hide()
	game.status.text="Choose an open dry shore for the company. Right-click cancels." if carry else "Choose navigable water. Right-click cancels."

func _unhandled_input(event: InputEvent) -> void:
	if picking.is_empty():return
	if event is InputEventKey and event.pressed and event.keycode==KEY_ESCAPE:
		picking={}
		get_viewport().set_input_as_handled()
	elif event is InputEventMouseButton and event.pressed:
		if event.button_index==MOUSE_BUTTON_RIGHT:picking={}
		elif event.button_index==MOUSE_BUTTON_LEFT:
			var tile: Vector2=game._tile_at(game.get_canvas_transform().affine_inverse()*event.position)
			picking["x"]=floori(tile.x)
			picking["y"]=floori(tile.y)
			game._queue_command("transport_company" if picking.has("fort_id") else "naval_order",picking.duplicate())
			picking={}
		else:return
		get_viewport().set_input_as_handled()

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
		if asset in ["transportship","enemytransport"]: asset="merchantship"
		if asset=="ferryboat":
			asset="fishingboat"
			if data.stage=="crossing":
				track.target += Vector2(0,float(data.get("ferry_lane",0))*game.tile_height*0.24)
		if asset=="enemywarship": asset="warship"
		if not game.assets.has(asset): continue
		var actor: Sprite2D=track.sprite
		var position: Vector2=actor.position
		if game.paused or tick<last_tick or last_turn!=game.view_turns or position==Vector2.ZERO: position=track.target
		else: position=position.move_toward(track.target, 260.0*get_process_delta_time()*game.simulation_speed)
		game._skin_sprite(actor,asset,position)
		var heading: Vector2=game._iso(track.heading.x,track.heading.y)-game._iso(0,0)
		actor.flip_h=heading.x<0
		actor.visible=str(data.stage)!="away"
		actor.modulate=Color("dd8f78") if str(data.kind).begins_with("enemy") else (Color("e6d9ba") if data.kind=="transportship" else Color.WHITE)
		var status=actor.get_node_or_null("ShipStatus")
		if status==null:
			status=ShipStatus.new()
			status.name="ShipStatus"
			actor.add_child(status)
		status.scale=Vector2.ONE/actor.scale
		status.position=Vector2(0,-actor.texture.get_height()*0.85)
		status.health=int(data.get("hull",0))
		status.maximum=int(data.get("hull_max",0))
		status.passengers=int(data.get("soldiers",0))
		status.hostile=str(data.kind).begins_with("enemy")
		status.visible=status.maximum>0 and (status.hostile or status.health<status.maximum or status.passengers>0)
		status.queue_redraw()
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
		vessels.set_item_text(i,"%s %d · %s · hull %d · fatigue %d" % [str(b.kind).capitalize(),int(b.id),str(b.stage),int(b.get("hull",0)),int(b.get("crew_fatigue",0))])
	var selected_company: int=int(company_picker.get_selected_metadata()) if company_picker.selected>=0 else 0
	company_picker.clear()
	for c: Dictionary in game.state.get("companies",[]):
		company_picker.add_item("Fort %d · %d %s" % [int(c.fort_id),int(c.soldiers),str(c.get("kind","infantry"))])
		company_picker.set_item_metadata(company_picker.item_count-1,int(c.fort_id))
		if int(c.fort_id)==selected_company:company_picker.select(company_picker.item_count-1)
	sail.disabled=military.is_empty()
	recall.disabled=military.is_empty()
	for button: Button in [engage,seek,repair,evade]: button.disabled=military.is_empty()
	transport.disabled=military.is_empty() or company_picker.item_count==0
	transport_pick.disabled=transport.disabled
	sail_pick.disabled=military.is_empty()
	raid.disabled=game.state.has("campaign") or bool(game.state.get("military",{}).get("invasion_active",false))
	x.max_value=int(game.state.get("map_extent",32))-1
	y.max_value=x.max_value
	var trade: Dictionary={}
	for r: Dictionary in game.state.get("trade_routes",[]):
		if r.id=="nile_route": trade=r
	route.visible=bool(trade.get("available",false)) and not bool(trade.get("open",false))
	var lines: Array[String]=["Ships built: %d · Fish caught: %d" % [int(game.state.get("ships_built",0)),int(game.state.get("resource_produced",{}).get("fish",0))]]
	for b: Dictionary in entries:
		lines.append("%s %d: %s%s" % [str(b.kind).capitalize(),int(b.id),str(b.stage), " · %d %s" % [int(b.cargo),str(b.resource)] if int(b.cargo)>0 else ""])
	var losses: Dictionary=game.state.get("naval",{})
	lines.append("Enemy ships sunk: %d · Own ships lost: %d" % [int(losses.get("destroyed",0)),int(losses.get("lost",0))])
	summary.text="\n".join(lines)


class ShipStatus extends Node2D:
	var health:=0
	var maximum:=0
	var passengers:=0
	var hostile:=false
	func _draw() -> void:
		if maximum<=0:return
		draw_rect(Rect2(-30,-8,60,7),Color("382a21"))
		draw_rect(Rect2(-29,-7,58*clampf(float(health)/maximum,0,1),5),Color("c54a36") if hostile else Color("77ad61"))
		if passengers>0:
			draw_string(ThemeDB.fallback_font,Vector2(-28,-12),"%d aboard" % passengers,HORIZONTAL_ALIGNMENT_LEFT,-1,12,Color("fff4d2"))
