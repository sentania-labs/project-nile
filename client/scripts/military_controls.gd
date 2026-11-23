extends VBoxContainer
## Commands and inspection only. Recruitment, movement and combat belong to Go.

var game: Node
var companies := OptionButton.new()
var x := SpinBox.new()
var y := SpinBox.new()
var muster: Button
var recall: Button
var raid: Button
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
	muster = game._button("Muster here", func() -> void: _order("muster"))
	recall = game._button("Return to fort", func() -> void: _order("return"))
	row.add_child(muster)
	row.add_child(recall)
	raid = game._button("Test invasion: four raiders", func() -> void: game._queue_command("test_invasion"))
	add_child(raid)
	add_child(game._label("Copper → weapons → Recruiter → Infantry Fort. Muster a company to intercept raiders. The test invasion and combat timing are provisional.", 13, Color("392617")))
	hide()


func _order(action: String) -> void:
	if companies.selected < 0: return
	game._queue_command("military_order", {"fort_id": companies.get_selected_metadata(), "order": action, "x": int(x.value), "y": int(y.value)})


func sync(topic: String) -> void:
	visible = topic == "Military"
	# The existing directional walking animation supplies the feet. Uniform
	# colors and original vector equipment distinguish military figures.
	for id: String in game.walker_nodes:
		var role := str(game.walker_data[id].get("role", ""))
		if role not in ["infantry", "invader"]: continue
		var actor: Sprite2D = game.walker_nodes[id]
		actor.modulate = Color("c9e5ff") if role == "infantry" else Color("e39171")
		var equipment := actor.get_node_or_null("MilitaryGear")
		if equipment == null:
			equipment = MilitaryGear.new()
			equipment.name = "MilitaryGear"
			equipment.hostile = role == "invader"
			actor.add_child(equipment)
		equipment.scale = Vector2(1.0 / actor.scale.x, 1.0 / actor.scale.y)
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
		companies.set_item_text(index, "Fort %d: %d / 16 infantry, %d arriving, %d at home" % [int(company.fort_id), int(company.soldiers), int(company.arriving), int(company.stationed)])
	muster.disabled = entries.is_empty()
	recall.disabled = entries.is_empty()
	var stats: Dictionary = game.state.get("military", {})
	raid.disabled = game.state.has("campaign") or bool(stats.get("invasion_active", false)) or int(game.state.get("population", 0)) == 0
	summary.text = "Recruited %d · Lost %d · Invaders defeated %d · Breaches %d" % [int(stats.get("recruited", 0)), int(stats.get("lost", 0)), int(stats.get("defeated", 0)), int(stats.get("breaches", 0))]


class MilitaryGear extends Node2D:
	var hostile := false
	func _draw() -> void:
		draw_line(Vector2(7, -29), Vector2(7, -1), Color("73522e"), 1.5)
		draw_colored_polygon(PackedVector2Array([Vector2(7,-34),Vector2(4,-26),Vector2(10,-26)]), Color("cfb790"))
		var shield := PackedVector2Array([Vector2(-9,-18),Vector2(-4,-20),Vector2(0,-17),Vector2(0,-8),Vector2(-5,-5),Vector2(-10,-8)])
		draw_colored_polygon(shield, Color("973d2c") if hostile else Color("466a91"))
		draw_polyline(shield + PackedVector2Array([shield[0]]), Color("d7ae65"), 1.2)
