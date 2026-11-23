extends VBoxContainer
## Commands and inspection only. Recruitment, movement and combat belong to Go.

var game: Node
var companies := OptionButton.new()
var x := SpinBox.new()
var y := SpinBox.new()
var muster: Button
var recall: Button
var raid: Button
var formation:=OptionButton.new()
var rotate: Button
var place: Button
var picking_fort:=0
var picking_order:=""
var summary := Label.new()
var company_ids: Array[int] = []


func setup(owner: Node) -> void:
	game = owner
	summary.autowrap_mode = TextServer.AUTOWRAP_WORD_SMART
	add_child(summary)
	add_child(companies)
	var row := HBoxContainer.new()
	add_child(row)
	for picker: SpinBox in [x, y]:
		picker.min_value = 0
		picker.max_value = 31
		picker.step = 1
		row.add_child(picker)
	x.prefix = "X "
	y.prefix = "Y "
	x.value = 26
	y.value = 18
	for order in ["muster","tight","loose","engage","mopup","charge"]:
		formation.add_item(str(order).capitalize())
		formation.set_item_metadata(formation.item_count-1,order)
	row.add_child(formation)
	muster = game._button("Issue order", func() -> void: _order(str(formation.get_selected_metadata())))
	recall = game._button("Return to fort", func() -> void: _order("return"))
	row.add_child(muster)
	place=game._button("Place formation on map",func() -> void:
		if companies.selected<0:return
		picking_fort=int(companies.get_selected_metadata())
		picking_order=str(formation.get_selected_metadata())
		game.city_ui.overseer_modal.hide()
		game.status.text="Choose dry ground for the company. Right-click cancels.")
	add_child(place)
	row.add_child(recall)
	rotate=game._button("Rotate ranks",func() -> void: _order("rotate"))
	row.add_child(rotate)
	raid = game._button("Test invasion: four raiders", func() -> void: game._queue_command("test_invasion"))
	add_child(raid)
	add_child(game._label("Archers equip themselves. Infantry need weapons; charioteers need timber-made chariots. Recruits visit an active Academy. Towers require wall foundations, staff and sentries. Combat timing is provisional.", 13, Color("392617")))
	hide()


func _order(action: String) -> void:
	if companies.selected < 0: return
	game._queue_command("military_order", {"fort_id": companies.get_selected_metadata(), "order": action, "x": int(x.value), "y": int(y.value)})

func _unhandled_input(event: InputEvent) -> void:
	if picking_fort==0:return
	if event is InputEventKey and event.pressed and event.keycode==KEY_ESCAPE:
		picking_fort=0
		get_viewport().set_input_as_handled()
	elif event is InputEventMouseButton and event.pressed:
		if event.button_index==MOUSE_BUTTON_RIGHT:picking_fort=0
		elif event.button_index==MOUSE_BUTTON_LEFT:
			var tile: Vector2=game._tile_at(game.get_canvas_transform().affine_inverse()*event.position)
			game._queue_command("military_order",{"fort_id":picking_fort,"order":picking_order,"x":floori(tile.x),"y":floori(tile.y)})
			picking_fort=0
		else:return
		get_viewport().set_input_as_handled()


func sync(topic: String) -> void:
	visible = topic == "Military"
	# Original directional military animation sheets supply each figure.
	for id: String in game.walker_nodes:
		var role := str(game.walker_data[id].get("role", ""))
		if role not in ["infantry", "archer", "charioteer", "sentry", "invader"]: continue
		var actor: Sprite2D = game.walker_nodes[id]
		actor.modulate = Color.WHITE if role != "invader" else Color("e39171")
	if not visible: return
	var entries: Array = game.state.get("companies", [])
	x.max_value = int(game.state.get("map_extent", 32)) - 1
	y.max_value = x.max_value
	var ids: Array[int] = []
	for company: Dictionary in entries: ids.append(int(company.fort_id))
	if ids != company_ids:
		company_ids = ids
		companies.clear()
		for id: int in ids:
			companies.add_item("")
			companies.set_item_metadata(companies.item_count - 1, id)
	for index in range(entries.size()):
		var company: Dictionary = entries[index]
		companies.set_item_text(index, "Fort %d: %d / 16 %s · %d arriving · %d at home · experience %d · morale %d" % [int(company.fort_id), int(company.soldiers), str(company.get("kind","infantry")), int(company.arriving), int(company.stationed),int(company.get("experience",0)),int(company.get("morale",60))])
	muster.disabled = entries.is_empty()
	place.disabled=entries.is_empty()
	recall.disabled = entries.is_empty()
	var stats: Dictionary = game.state.get("military", {})
	raid.disabled = game.state.has("campaign") or bool(stats.get("invasion_active", false)) or int(game.state.get("population", 0)) == 0
	summary.text = "Recruited %d · Lost %d · Invaders defeated %d · Breaches %d" % [int(stats.get("recruited", 0)), int(stats.get("lost", 0)), int(stats.get("defeated", 0)), int(stats.get("breaches", 0))]
