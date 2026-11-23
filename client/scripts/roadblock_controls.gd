extends RefCounted

const PATROLS := {"seller":"Bazaar sellers","watercarrier":"Water carriers","firefighter":"Firefighters","architect":"Architects","taxcollector":"Tax collectors","priest":"Priests","entertainer":"Entertainers","doctor":"Healthcare","teacher":"Education","policeman":"Police"}
var panel: Control
var game: Node
var at := Vector2i.ZERO
var choices: Dictionary = {}


func setup(ui: RefCounted, owner: Node) -> void:
	game = owner
	var modal: Array = ui._modal("Roadblock", Vector2(420, 495))
	panel = modal[0]
	modal[1].add_child(game._label("Allow these walkers through", 16, Color("392617")))
	for role: String in PATROLS:
		var button := CheckButton.new()
		button.text = PATROLS[role]
		button.add_theme_color_override("font_color", Color("392617"))
		modal[1].add_child(button)
		choices[role] = button
		button.toggled.connect(func(allow: bool) -> void:
			game._queue_command("road_permission", {"x": at.x, "y": at.y, "role": role, "allow": allow}))


func open_at(cell: Vector2i) -> bool:
	var feature: Dictionary = game.resources.features.get(cell, {})
	if feature.get("kind") != "roadblock": return false
	at = cell
	panel.show()
	sync()
	return true


func sync() -> void:
	if not panel.visible: return
	var feature: Dictionary = game.resources.features.get(at, {})
	if feature.get("kind") != "roadblock":
		panel.hide()
		return
	for role: String in choices:
		choices[role].set_pressed_no_signal(role in feature.get("pass", []))
