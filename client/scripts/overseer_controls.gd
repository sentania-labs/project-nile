extends VBoxContainer

var game: Node
var topic := ""
var route_button: Button
var pick := OptionButton.new()
var mode := OptionButton.new()
var limit := SpinBox.new()
var apply: Button
var selection_key := ""


func setup(owner: Node) -> void:
	game = owner
	route_button = game._button("", func() -> void: game._queue_command("open_trade", {"route":"kingdom_road"}))
	add_child(route_button)
	var row := HBoxContainer.new()
	add_child(row)
	row.add_child(pick)
	row.add_child(mode)
	limit.max_value = 3200
	limit.step = 1
	limit.suffix = "units"
	row.add_child(limit)
	apply = game._button("Apply", _submit)
	row.add_child(apply)
	pick.item_selected.connect(func(_idx: int) -> void: _set_selection())
	hide()


func _set_selection() -> void:
	if topic == "Trade" and pick.selected >= 0:
		var policy: Dictionary = game.state.get("trade_policies", {}).get(pick.get_selected_metadata(), {})
		mode.select(["off", "import", "export"].find(policy.get("mode", "off")))
		limit.value = int(policy.get("limit", 0))


func _submit() -> void:
	if pick.selected < 0: return
	if topic == "Trade":
		game._queue_command("trade_policy", {"resource":str(pick.get_selected_metadata()), "mode":["off", "import", "export"][mode.selected],"limit":int(limit.value)})
	else:
		game._queue_command("festival", {"god":str(pick.get_selected_metadata()), "festival_size":mode.selected+1})


func sync(next_topic: String) -> void:
	visible = next_topic in ["Trade", "Religion"]
	if not visible: return
	var state: Dictionary = game.state
	var items: Dictionary = {}
	if next_topic == "Trade":
		for price: Dictionary in state.get("trade_prices", []):
			items[price.resource] = "%s   %d / %d Db" % [str(price.resource).capitalize(),int(price.buy),int(price.sell)]
	else:
		var catalog: Array = state.get("building_catalog", [])
		for god: String in ["Osiris", "Ra", "Ptah", "Seth", "Bast"]:
			if catalog.any(func(entry: Dictionary) -> bool: return entry.kind == "temple_"+god.to_lower()): items[god] = god
	var key := next_topic + JSON.stringify(items)
	if key != selection_key:
		topic = next_topic
		selection_key = key
		pick.clear()
		mode.clear()
		for id: String in items:
			pick.add_item(items[id])
			pick.set_item_metadata(pick.item_count-1, id)
		for item: String in (["Off", "Import to", "Export above"] if topic == "Trade" else ["Common", "Large", "Lavish"]): mode.add_item(item)
		_set_selection()
	var routes: Array = state.get("trade_routes", [])
	var route: Dictionary = routes[0] if not routes.is_empty() else {}
	route_button.visible = topic == "Trade" and route.get("available", false) and not route.get("open",false)
	route_button.text = "Open %s · %d Db" % [route.get("name","trade route"), int(route.get("open_cost",0))]
	limit.visible = topic == "Trade"
	apply.text = "Set order" if topic == "Trade" else "Hold festival"
	apply.disabled = pick.item_count == 0 or (not route.get("available",false) if topic=="Trade" else state.has("festival"))
