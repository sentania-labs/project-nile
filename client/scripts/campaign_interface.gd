extends RefCounted

var game: Node
var panel: Control
var heading: Label
var body: Label
var decisions: VBoxContainer
var scroll: ScrollContainer
var button: Button
var previous_mission := -1
var won_shown := false
var briefing := false
var previous_event_tick := -1
var army_choices: Dictionary = {}
var army_naval: Dictionary = {}
var request_signature := ""

const Narrative = preload("res://scripts/mission_briefs.gd")


func setup(city: RefCounted, row: HBoxContainer) -> void:
	game = city.game
	button = game._button("Mission", show_objectives)
	button.flat = true
	row.add_child(button)
	button.hide()
	var modal: Array = city._modal("Campaign", Vector2(680, 640), dismiss)
	panel = modal[0]
	heading = game._label("", 22, Color("47301a"))
	modal[1].add_child(heading)
	scroll = ScrollContainer.new()
	scroll.horizontal_scroll_mode = ScrollContainer.SCROLL_MODE_DISABLED
	scroll.size_flags_vertical = Control.SIZE_EXPAND_FILL
	modal[1].add_child(scroll)
	var content := VBoxContainer.new()
	content.size_flags_horizontal = Control.SIZE_EXPAND_FILL
	content.add_theme_constant_override("separation", 12)
	scroll.add_child(content)
	body = game._label("", 16, Color("392617"))
	body.autowrap_mode = TextServer.AUTOWRAP_WORD_SMART
	content.add_child(body)
	decisions = VBoxContainer.new()
	content.add_child(decisions)


func refresh(operation: String) -> void:
	var mission: Dictionary = game.state.get("campaign", {})
	button.visible = not mission.is_empty()
	if mission.is_empty():
		previous_mission = -1
		previous_event_tick = -1
		won_shown = false
		briefing = false
		panel.hide()
		return
	if int(mission.mission_id) != previous_mission or operation == "new_campaign":
		previous_mission = int(mission.mission_id)
		won_shown = false
		briefing = operation in ["new_campaign", "advance_campaign"]
		if briefing: show_objectives()
	var event: Dictionary = game.state.get("city_events", {})
	var event_tick := int(event.get("last_tick", -1))
	if event_tick != previous_event_tick:
		previous_event_tick = event_tick
		if operation in ["state", "step"] and not str(event.get("last", "")).is_empty():
			game.status.text = str(event.last)
	var phases := PackedStringArray()
	for request: Dictionary in mission.get("requests", []):
		phases.append("%d:%s:%d" % [int(request.id), str(request.status), int(request.months)])
	var current_signature := ";".join(phases)
	if panel.visible and (operation in ["dispatch_request", "refuse_request", "dispatch_army"] or current_signature != request_signature):
		show_objectives()
	request_signature = current_signature
	if not mission.won and not mission.get("failed",false):
		won_shown = false
	if mission.won and not won_shown:
		won_shown = true
		briefing = false
		show_objectives()


func show_objectives() -> void:
	var mission: Dictionary = game.state.get("campaign", {})
	if mission.is_empty(): return
	game._cancel_gesture()
	heading.text = "%s  ·  %d BC" % [str(mission.name), absi(int(mission.start_year))]
	var selected: Dictionary = {}
	var naval_selected: Dictionary = {}
	for id in army_choices:
		selected[id] = []
		for choice: CheckBox in army_choices[id]:
			if choice.button_pressed: selected[id].append(int(choice.get_meta("fort_id")))
		if army_naval.has(id): naval_selected[id] = army_naval[id].button_pressed
	for child in decisions.get_children(): child.free()
	army_choices.clear()
	army_naval.clear()
	var lines := PackedStringArray()
	if briefing:
		var id:=int(mission.mission_id)
		lines.append(Narrative.BRIEFS[id] if id>=0 and id<Narrative.BRIEFS.size() else mission.guidance)
	else:
		lines.append("Your city has met its objectives." if mission.won else str(mission.guidance))
	for objective: Dictionary in mission.objectives:
		lines.append("%s   %d / %d" % [objective.label, int(objective.current), int(objective.target)])
	var last_event := str(game.state.get("city_events", {}).get("last", ""))
	if not last_event.is_empty(): lines.append(last_event)
	var requests = mission.get("requests")
	for request: Dictionary in (requests if requests is Array else []):
		if str(request.resource) == "troops":
			_army_request(request, lines)
			continue
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
	for army: Dictionary in game.state.get("army_dispatches", []):
		var sent := 0
		var alive := 0
		var returned := 0
		var defeated := 0
		for leg: Dictionary in game.state.get("deployments", []):
			if leg.request_id not in army.legs: continue
			sent += int(leg.strength)
			var troops = leg.get("troops")
			alive += troops.size() if troops is Array else 0
			if str(leg.status) == "returned": returned += 1
			if str(leg.status) in ["sunk", "defeated", "cancelled"]: defeated += 1
		var outcome := "%d soldiers sent · %d surviving" % [sent, alive]
		if defeated == army.legs.size(): outcome += " · Army defeated"
		elif returned + defeated == army.legs.size(): outcome += " · Survivors returned"
		lines.append(outcome)
	body.text = "\n".join(lines)
	for id in army_choices:
		for choice: CheckBox in army_choices[id]:
			choice.button_pressed = not choice.disabled and int(choice.get_meta("fort_id")) in selected.get(id, [])
		if army_naval.has(id): army_naval[id].button_pressed = naval_selected.get(id, false)
	if mission.won:
		for next: Dictionary in mission.next_missions:
			if next.playable:
				decisions.add_child(game._button("Continue to " + str(next.name), _advance.bind(int(next.id))))
	decisions.add_child(game._button("Begin settlement" if briefing else "Return to city", dismiss))
	panel.show()


func _army_request(request: Dictionary, lines: PackedStringArray) -> void:
	var phase := str(request.status)
	if phase.begins_with("army_"):
		var label: String = {
			"army_outbound": "Army travelling to the battle",
			"army_battle": "Army preparing to meet the enemy",
			"army_late": "Army missed the battle",
			"army_return_wait": "Survivors travelling home",
			"army_return": "Survivors approaching the city",
			"army_homebound": "Survivors marching back to their forts"
		}.get(phase, "Army away from the city")
		lines.append("%s · %d months remain." % [label, maxi(0, int(request.months))])
		return
	if phase != "requested": return
	lines.append("Pharaoh requests an army of %d strength. %d months remain; the journey takes %d months. Charioteers count double. Companies leave from their forts." % [int(request.quantity), int(request.months), int(request.get("travel_months", 0))])
	var choices: Array[CheckBox] = []
	for company: Dictionary in game.state.get("companies", []):
		var choice := CheckBox.new()
		var count := int(company.stationed)
		var kind := str(company.get("kind", "infantry"))
		choice.text = "Fort %d · %d %s at home" % [int(company.fort_id), count, {"archer": "archers", "charioteer": "charioteers"}.get(kind, kind)]
		choice.set_meta("fort_id", int(company.fort_id))
		choice.disabled = count == 0 or str(company.order) == "deployment"
		choices.append(choice)
		decisions.add_child(choice)
	army_choices[int(request.id)] = choices
	var naval := CheckBox.new()
	naval.text = "Send by river · one ready transport ship per company"
	decisions.add_child(naval)
	army_naval[int(request.id)] = naval
	var dispatch: Button = game._button("Dispatch selected companies", _dispatch_army.bind(int(request.id)))
	dispatch.disabled = not bool(request.can_dispatch)
	decisions.add_child(dispatch)


func _dispatch_army(id: int) -> void:
	var forts: Array[int] = []
	for choice: CheckBox in army_choices.get(id, []):
		if choice.button_pressed and not choice.disabled: forts.append(int(choice.get_meta("fort_id")))
	if forts.is_empty():
		game.status.text = "Select at least one company with soldiers at its fort."
		return
	game._queue_command("dispatch_army", {"request_id": id, "fort_ids": forts, "naval": army_naval[id].button_pressed})


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
