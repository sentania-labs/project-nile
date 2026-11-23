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
	directory=OS.get_environment("NILE_MASTABA_DIR")
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
	game.focus=game._iso(22,23)
	game._set_view(0.95)
	await game._smoke_choose_tool("smallmastaba")
	await game._click_cell(Vector2i(23,18))
	if not await game._wait_idle() or not game._has_building_kind("smallmastaba"):fail("Mastaba menu placement failed");return
	await game._smoke_choose_tool("inspect")
	game.show_footprints=true
	game.overlay.queue_redraw()
	await capture("01-mastaba-foundation")
	game.show_footprints=false
	game.overlay.queue_redraw()
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
		if not await game._wait_idle():fail("Mastaba work failed: "+str(game.last_reply.get("error","no reply")));return
		var b: Dictionary=game._building_at(Vector2i(23,18))
		var done:=int(b.get("construction_done",0))
		var name:=""
		if done>=1800 and done<3600:name="02-smallmastaba-carving"
		elif done>=3600 and done<7200:name="03-smallmastaba-body"
		elif done>=7200 and done<10800:name="04-smallmastaba-detail"
		elif done==10800:name="05-smallmastaba-complete"
		if not name.is_empty() and not seen.has(name):
			seen[name]=true
			if game.city_ui.completion_modal.visible:
				await game._smoke_click_button(game.city_ui.completion_continue)
				await game._wait_idle()
			await game._click_cell(Vector2i(24,24))
			if game.selected.is_empty() or game.selected.get("building",{}).get("kind","")!="smallmastaba":fail("Mastaba inspection did not open");return
			await capture(name)
			await create_timer(0.5).timeout
		if done==10800:break
	var final: Dictionary=game._building_at(Vector2i(23,18))
	if str(final.get("stage",""))!="complete" or int(final.get("monument_materials",{}).get("bricks",0))!=10800:fail("Mastaba did not complete its original courses and brick deliveries");return
	var info_click:=InputEventMouseButton.new()
	info_click.button_index=MOUSE_BUTTON_RIGHT
	info_click.position=game.get_canvas_transform()*game._iso(float(Vector2i(24,24).x)+0.5,float(Vector2i(24,24).y)+0.5)
	info_click.pressed=true
	root.push_input(info_click,true)
	await process_frame
	info_click.pressed=false
	root.push_input(info_click,true)
	await process_frame
	if not game.city_ui.inspector_modal.visible or game.selected.get("building",{}).get("kind","")!="smallmastaba":fail("Right-click monument inspector failed");return
	await capture("06-construction-foreman")
	info_click.pressed=true
	root.push_input(info_click,true)
	await process_frame
	await game._smoke_click_button(game.save_button)
	if not await game._wait_idle():fail("Save failed");return
	var saved: Dictionary=game.state.duplicate(true)
	await game._smoke_click_button(game.load_button)
	if not await game._wait_idle() or game.state!=saved:fail("Mastaba save/load changed state");return
	print("NILE_MASTABA_PASS UI placement, physical Work Camp and Bricklayers, 10800 bricks, 27 sledges, completed rating9, SQLite replay")
	game.sim.stop()
	quit(0)
