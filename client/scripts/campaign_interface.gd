extends RefCounted

var game: Node
var panel: Control
var heading: Label
var body: Label
var decisions: VBoxContainer
var button: Button
var previous_mission := -1
var won_shown := false
var briefing := false

const BRIEFS := {
	0: "On this stretch of the Nile, families need a place to settle. Lay a street from the Kingdom Road and mark house plots. As the village grows, hunters can bring meat from the desert, and water carriers can serve the homes. Establish six occupied Meager Shanties to secure Nubt’s future.",
	1: "Lead the families of Thinis as it grows into a center of craft and exchange. Consult the mission objectives as new responsibilities open.",
	2: "Guide Perwadjyt through the Nile’s agricultural year. Establish food supplies and the services required by the growing town."
}


func setup(city: RefCounted, row: HBoxContainer) -> void:
	game = city.game
	button = game._button("Mission", show_objectives)
	button.flat = true
	row.add_child(button)
	button.hide()
	var modal: Array = city._modal("Campaign", Vector2(610, 360), dismiss)
	panel = modal[0]
	heading = game._label("", 22, Color("47301a"))
	modal[1].add_child(heading)
	body = game._label("", 16, Color("392617"))
	body.autowrap_mode = TextServer.AUTOWRAP_WORD_SMART
	body.size_flags_vertical = Control.SIZE_EXPAND_FILL
	modal[1].add_child(body)
	decisions = VBoxContainer.new()
	modal[1].add_child(decisions)


func refresh(operation: String) -> void:
	var mission: Dictionary = game.state.get("campaign", {})
	button.visible = not mission.is_empty()
	if mission.is_empty():
		previous_mission = -1
		won_shown = false
		briefing = false
		panel.hide()
		return
	if int(mission.mission_id) != previous_mission or operation == "new_campaign":
		previous_mission = int(mission.mission_id)
		won_shown = false
		briefing = operation in ["new_campaign", "advance_campaign"]
		if briefing: show_objectives()
	if panel.visible and operation in ["dispatch_request", "refuse_request"]:
		show_objectives()
	if mission.won and not won_shown:
		won_shown = true
		briefing = false
		show_objectives()


func show_objectives() -> void:
	var mission: Dictionary = game.state.get("campaign", {})
	if mission.is_empty(): return
	game._cancel_gesture()
	heading.text = "%s  ·  %d BC" % [str(mission.name), absi(int(mission.start_year))]
	for child in decisions.get_children(): child.free()
	var lines := PackedStringArray()
	if briefing:
		lines.append(BRIEFS.get(int(mission.mission_id), mission.guidance))
	else:
		lines.append("Your city has met its objectives." if mission.won else str(mission.guidance))
	for objective: Dictionary in mission.objectives:
		lines.append("%s   %d / %d" % [objective.label, int(objective.current), int(objective.target)])
	var requests = mission.get("requests")
	for request: Dictionary in (requests if requests is Array else []):
		if not str(request.status) in ["requested", "overdue", "dispatched", "dispatched_late"]: continue
		var cargo := "%d %s" % [int(request.quantity), str(request.resource).capitalize()]
		if str(request.status).begins_with("dispatched"):
			lines.append("On its way to Pharaoh: " + cargo)
			continue
		lines.append("Pharaoh requests %s. %d in stock. %d months remain." % [cargo, int(request.available), maxi(0, int(request.months))])
		var row := HBoxContainer.new()
		decisions.add_child(row)
		var dispatch: Button = game._button("Dispatch " + cargo, _respond.bind("dispatch_request", int(request.id)))
		dispatch.disabled = not request.can_dispatch
		row.add_child(dispatch)
	body.text = "\n\n".join(lines)
	if mission.won:
		for next: Dictionary in mission.next_missions:
			if next.playable:
				decisions.add_child(game._button("Continue to " + str(next.name), _advance.bind(int(next.id))))
	decisions.add_child(game._button("Begin settlement" if briefing else "Return to city", dismiss))
	panel.show()


func _respond(operation: String, id: int) -> void:
	game._queue_command(operation, {"request_id": id})


func _advance(id: int) -> void:
	panel.hide()
	game._queue_command("pause")
	game._queue_command("advance_campaign", {"mission_id": id})


func dismiss() -> void:
	panel.hide()
	if briefing:
		briefing = false
		game._queue_command("resume")
