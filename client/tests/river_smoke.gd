extends SceneTree
## Load a settlement built with ordinary commands; exercise real naval controls.
var game: Node
var directory := "/mnt/holding/project-nile/river-review-20261123"
func _initialize() -> void: _run.call_deferred()
func _check(ok: bool, why: String) -> bool:
	if not ok:
		push_error(why)
		quit(1)
	return ok
func _step(ticks: int) -> bool:
	game._queue_command("step", {"ticks":ticks})
	return await game._wait_idle()
func _capture(name: String) -> void:
	await process_frame
	await RenderingServer.frame_post_draw
	root.get_texture().get_image().save_png(directory.path_join(name))
func _run() -> void:
	DirAccess.make_dir_recursive_absolute(directory)
	root.size=Vector2i(1920,1080)
	root.set_meta("nile_start_choice","load")
	game=load("res://scenes/sandbox.tscn").instantiate()
	root.add_child(game)
	var deadline:=Time.get_ticks_msec()+20000
	while (not game.startup_done or game.paused) and Time.get_ticks_msec()<deadline: await process_frame
	if not _check(game.startup_done and not game.paused,"River settlement did not load"):return
	await game._smoke_click_button(game.pause_button)
	if not _check(await game._wait_idle() and game.paused,"River clock did not pause"):return
	game.focus=game._iso(9,16)
	game._set_view(0.7)
	await _capture("river-city.png")
	await game._smoke_click_button(game.city_ui.overseers_button)
	await game._smoke_click_button(game.city_ui.overseer_tabs["River"])
	var controls=game.city_ui.naval_controls
	if not _check(controls.visible and controls.vessels.item_count==2,"River vessel controls missing"):return
	controls.x.value=1
	controls.y.value=12
	await game._smoke_click_button(controls.sail)
	if not _check(await game._wait_idle(),"River sail order failed"):return
	game.city_ui.overseer_modal.hide()
	if not _check(await _step(24),"River voyage failed"):return
	game.focus=game._iso(5,13)
	game._set_view(1.0)
	await _capture("river-voyage.png")
	var fish:=false
	var merchant:=false
	for i in range(80):
		if not _check(await _step(2),"River economy failed"):return
		for b: Dictionary in game.state.get("boats",[]):
			if int(b.cargo)>0 and b.kind=="fishingboat" and not fish:
				fish=true
				game.focus=game._iso(float(b.x)+4,float(b.y))
				await _capture("fishing-catch.png")
			if int(b.cargo)>0 and b.kind=="merchantship" and not merchant:
				merchant=true
				game.focus=game._iso(float(b.x)+4,float(b.y))
				await _capture("merchant-cargo.png")
		if fish and merchant:break
	if not _check(fish and merchant,"Fishing and merchant cargo were not visible"):return
	game._queue_command("save")
	if not _check(await game._wait_idle(),"River save failed"):return
	var hash:=str(game.last_reply.hash)
	if not _check(await _step(8),"River post-save tick failed"):return
	game._queue_command("load")
	if not _check(await game._wait_idle() and str(game.last_reply.hash)==hash,"River reload changed physical cargo"):return
	await game._smoke_click_button(game.city_ui.overseers_button)
	await game._smoke_click_button(game.city_ui.overseer_tabs["River"])
	await game._smoke_click_button(controls.recall)
	if not _check(await game._wait_idle(),"Return to wharf failed"):return
	game.focus=game._iso(7,20)
	game._set_view(0.9)
	await _capture("river-overseer.png")
	game.city_ui.overseer_modal.hide()
	if not _check(await _step(80),"Return voyage failed"):return
	var returned:=false
	for b: Dictionary in game.state.boats:
		if b.kind=="transportship" and b.stage=="moored":returned=true
	if not _check(returned,"Transport did not return to wharf"):return
	print("NILE_RIVER_PASS fishing merchant cargo timber construction sailing save reload return ",JSON.stringify(game.state.boats))
	quit(0)
