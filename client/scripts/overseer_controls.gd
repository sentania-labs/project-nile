extends VBoxContainer

var game: Node
var topic := ""
var route_pick := OptionButton.new()
var route_key := ""
var route_info := Label.new()
var route_button: Button
var pick := OptionButton.new()
var mode := OptionButton.new()
var limit := SpinBox.new()
var apply: Button
var selection_key := ""


func setup(owner: Node) -> void:
	game = owner
	add_child(route_pick)
	route_pick.item_selected.connect(func(_idx: int) -> void: sync(topic))
	route_button = game._button("", func() -> void: game._queue_command("open_trade", {"route":str(route_pick.get_selected_metadata())}))
	add_child(route_button)
	route_info.autowrap_mode = TextServer.AUTOWRAP_WORD_SMART
	add_child(route_info)
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
	var partner_ids := PackedStringArray()
	for partner: Dictionary in routes: partner_ids.append(str(partner.id))
	var next_key := ",".join(partner_ids)
	if next_key!=route_key:
		route_key=next_key
		route_pick.clear()
		for partner: Dictionary in routes:
			route_pick.add_item(str(partner.name))
			route_pick.set_item_metadata(route_pick.item_count-1,partner.id)
	route_pick.visible = topic=="Trade" and not routes.is_empty()
	var route: Dictionary = routes[route_pick.selected] if route_pick.selected>=0 and route_pick.selected<routes.size() else {}
	route_info.visible = topic=="Trade" and route.has("limits")
	if route_info.visible:
		var entries := PackedStringArray()
		for side in ["imports","exports"]:
			var allowed := PackedStringArray()
			for r: String in route.get(side,[]):
				allowed.append("%s %d" % [r,int(route.limits.get(r,0))])
			if not allowed.is_empty(): entries.append("%s per year · %s" % ["Buy" if side=="imports" else "Sell",", ".join(allowed)])
		route_info.text="\n".join(entries)
	route_button.visible = topic == "Trade" and route.get("available", false) and not route.get("open",false)
	route_button.text = "Open %s · %d Db" % [route.get("name","trade route"), int(route.get("open_cost",0))]
	limit.visible = topic == "Trade"
	apply.text = "Set order" if topic == "Trade" else "Hold festival"
	apply.disabled = pick.item_count == 0 or (not route.get("available",false) if topic=="Trade" else state.has("festival"))
