extends HBoxContainer

const ORDERS := ["accept", "get", "refuse", "empty"]
var game: Node
var resource := OptionButton.new()
var order := OptionButton.new()
var bound_building: Dictionary


func setup(owner: Node) -> void:
	game = owner
	add_child(game._label("Storage orders", 14, Color("392617")))
	add_child(resource)
	add_child(order)
	for action: String in ORDERS: order.add_item(action.capitalize())
	resource.item_selected.connect(func(_index: int) -> void: _show_order())
	order.item_selected.connect(func(index: int) -> void:
		game._queue_command("storage_order", {"x": int(bound_building.x), "y": int(bound_building.y), "resource": resource.get_selected_metadata(), "order": ORDERS[index]}))
	hide()


func bind(building: Dictionary) -> void:
	visible = str(building.get("kind", "")) == "storageyard"
	if not visible: return
	var rebuild: bool = bound_building.get("id") != building.get("id") or bound_building.get("resource_limits") != building.get("resource_limits")
	bound_building = building
	if rebuild:
		resource.clear()
		for r: Dictionary in game.state.get("resource_catalog", []):
			if not building.get("resource_limits", {}).has(r.id): continue
			resource.add_item(r.label)
			resource.set_item_metadata(resource.item_count - 1, r.id)
	_show_order()


func _show_order() -> void:
	if resource.item_count > 0:
		order.select(ORDERS.find(bound_building.get("storage_orders", {}).get(resource.get_selected_metadata(), "accept")))
