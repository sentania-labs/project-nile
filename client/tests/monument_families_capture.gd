extends SceneTree
var game: Node
var directory: String
var seen := {}
func _initialize() -> void: _run.call_deferred()
func fail(message: String) -> void:
	push_error(message)
	if game != null and game.sim != null:game.sim.stop()
	quit(1)
func capture(name: String) -> void:
	await RenderingServer.frame_post_draw
	if root.get_texture().get_image().save_png(directory.path_join(name+".png"))!=OK:fail("Capture failed")
func _run() -> void:
	directory=OS.get_environment("NILE_MONUMENT_FAMILIES_DIR")
	root.size=Vector2i(1920,1080)
	root.set_meta("nile_start_choice","load")
	game=load("res://scenes/sandbox.tscn").instantiate()
	root.add_child(game)
	var deadline:=Time.get_ticks_msec()+20000
	while not game.startup_done and Time.get_ticks_msec()<deadline:await process_frame
	if not game.startup_done or not await game._wait_idle():fail("Prepared city did not load: "+str(game.last_reply.get("error",game.status.text)));return
	if not game.paused:
		await game._smoke_click_button(game.pause_button)
		await game._wait_idle()
	game.focus=game._iso(19,25)
	game._set_view(0.92)
	await game._smoke_choose_tool("sphinx")
	await game._click_cell(Vector2i(11,24))
	if not await game._wait_idle() or not game._has_building_kind("sphinx"):fail("Sphinx menu placement failed");return
	await game._smoke_choose_tool("inspect")
	await capture("01-sphinx-scaffolding")
	var ready:=FileAccess.open(OS.get_environment("NILE_MONUMENT_READY"),FileAccess.WRITE)
	ready.store_string("ready");ready.close()
	for i in 2:
		await game._smoke_click_button(game.speed_button)
		await game._wait_idle()
	await game._smoke_click_button(game.pause_button)
	await game._wait_idle()
	await create_timer(6).timeout
	await game._smoke_click_button(game.pause_button)
	await game._wait_idle()
	for sample in 400:
		game._queue_command("step",{"ticks":48})
		if not await game._wait_idle():fail("Sphinx work failed: "+str(game.last_reply.get("error","no reply")));return
		var b: Dictionary=game._building_at(Vector2i(11,24))
		var done:=int(b.get("construction_done",0))
		var name:=""
		if done>=3 and done<10:name="02-sphinx-carving"
		elif done>=10 and done<17:name="03-sphinx-body"
		elif done>=17 and done<22:name="04-sphinx-detail"
		elif done==22:name="05-sphinx-complete"
		if not name.is_empty() and not seen.has(name):
			seen[name]=true
			if game.city_ui.completion_modal.visible:
				await game._smoke_click_button(game.city_ui.completion_continue)
				await game._wait_idle()
			await capture(name)
			await create_timer(0.5).timeout
		if done==22:break
	var final: Dictionary=game._building_at(Vector2i(11,24))
	if str(final.get("stage",""))!="complete" or int(final.get("monument_materials",{}).get("timber",0))!=2000:fail("Sphinx did not complete its original jobs and timber deliveries");return
	await game._smoke_click_button(game.save_button)
	if not await game._wait_idle():fail("Save failed");return
	var saved: Dictionary=game.state.duplicate(true)
	await game._smoke_click_button(game.load_button)
	if not await game._wait_idle() or game.state!=saved:fail("Sphinx save/load changed state");return
	game._queue_command("new_town",{"scenario":"valley"})
	if not await game._wait_idle():fail("Valley reset failed");return
	game.focus=game._iso(19,32)
	game._set_view(0.36)
	await game._smoke_choose_tool("grandpyramid")
	var mouse:=InputEventMouseMotion.new()
	mouse.position=game.get_canvas_transform()*game._iso(5.5,26.5)
	root.warp_mouse(mouse.position)
	root.push_input(mouse,true)
	await process_frame
	await process_frame
	if not game.hover_valid:fail("Grand complex preview is invalid: "+game._placement_error(game.hover_cell));return
	await capture("06-grand-complex-preview")
	await game._click_cell(Vector2i(5,26))
	if not await game._wait_idle() or not game._has_building_kind("grandpyramid"):fail("Grand complex west-bank placement failed");return
	await game._smoke_choose_tool("inspect")
	await capture("07-grand-complex-foundation")
	var b: Dictionary=game._building_at(Vector2i(5,26))
	if b.get("monument_parts",[]).size()!=3:fail("Grand complex lost its temple and causeway");return
	print("NILE_MONUMENT_FAMILIES_PASS Sphinx UI-place physical carpenters/masons22jobs 2000timber sqlite grandcomplex composite westbank UI-preview/placement")
	game.sim.stop()
	quit(0)
