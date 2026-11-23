extends SceneTree
## Isolated Selima source-event fixture, after physical company recruitment.
## Export with NILE_ARMY_CAPTURE_SAVE from the Go campaign army regression.
var game: Node
var directory := "/mnt/holding/project-nile/campaign-army-review-20261123"

func _initialize() -> void: _run.call_deferred()
func _check(ok: bool, why: String) -> bool:
	if not ok: push_error(why); quit(1)
	return ok
func _step(ticks: int) -> bool:
	game._queue_command("step", {"ticks": ticks})
	return await game._wait_idle()
func _capture(name: String) -> void:
	await process_frame
	await RenderingServer.frame_post_draw
	root.get_texture().get_image().save_png(directory.path_join(name))
func _run() -> void:
	DirAccess.make_dir_recursive_absolute(directory)
	root.size = Vector2i(1920, 1080)
	root.set_meta("nile_start_choice", "load")
	game = load("res://scenes/sandbox.tscn").instantiate()
	root.add_child(game)
	var deadline := Time.get_ticks_msec() + 20000
	while (not game.startup_done or game.paused) and Time.get_ticks_msec() < deadline: await process_frame
	if not _check(game.startup_done and not game.paused, "Army request fixture failed to load"): return
	await game._smoke_click_button(game.pause_button)
	if not _check(await game._wait_idle() and game.paused, "Army request clock failed to pause"): return
	var mission = game.city_ui.campaign_ui
	await game._smoke_click_button(mission.button)
	if not _check(mission.army_choices.has(1) and mission.army_choices[1].size() == 2, "Mission omitted available companies"): return
	for choice: CheckBox in mission.army_choices[1]:
		mission.scroll.ensure_control_visible(choice)
		await process_frame
		await game._smoke_click_button(choice)
	await _capture("army-selection.png")
	var dispatch: Button
	for child in mission.decisions.get_children():
		if child is Button and child.text == "Dispatch selected companies": dispatch = child
	if not _check(dispatch != null and not dispatch.disabled, "Army request cannot be dispatched"): return
	mission.scroll.ensure_control_visible(dispatch)
	await process_frame
	await game._smoke_click_button(dispatch)
	if not _check(await game._wait_idle() and game.last_reply.ok and game.state.army_dispatches.size() == 1, "Army dispatch failed"): return
	mission.dismiss()
	game.focus = game._iso(17, 17)
	game._set_view(0.9)
	if not _check(await _step(8), "Army failed to march"): return
	await _capture("army-departure.png")
	game._queue_command("save")
	if not _check(await game._wait_idle(), "Army save failed"): return
	var hash := str(game.last_reply.hash)
	if not _check(await _step(4), "Army post-save step failed"): return
	game._queue_command("load")
	if not _check(await game._wait_idle() and str(game.last_reply.hash) == hash, "Army reload changed journey"): return
	var saw_held := false
	var saw_homebound := false
	for i in range(200):
		if not _check(await _step(2), "Army campaign journey failed"): return
		var returned := 0
		for leg: Dictionary in game.state.deployments:
			if str(leg.status) == "awaits_return": saw_held = true
			if str(leg.status) == "returning": saw_homebound = true
			if str(leg.status) == "returned": returned += 1
		if returned == 2: break
	if not _check(saw_held and saw_homebound and int(game.state.military.lost) == 2, "Army skipped casualty or delayed return stages"): return
	for leg: Dictionary in game.state.deployments:
		if not _check(leg.status == "returned" and int(leg.survivors) == 15, "Company survivors failed to return"): return
	await game._smoke_click_button(mission.button)
	if not _check("Survivors returned" in mission.body.text, "Mission omitted army outcome"): return
	await _capture("army-returned.png")
	print("NILE_CAMPAIGN_ARMY_PASS ", JSON.stringify(game.state.army_dispatches), " lost=", game.state.military.lost)
	quit(0)
