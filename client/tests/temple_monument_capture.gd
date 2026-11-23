extends SceneTree
var game: Node
var directory: String
var seen := {}
var kind: String
var expected_timber: int
var expected_sandstone: int
func _initialize() -> void: _run.call_deferred()
func fail(message: String) -> void:
	push_error(message)
	if game != null and game.sim != null:game.sim.stop()
	quit(1)
func capture(name: String) -> void:
	await RenderingServer.frame_post_draw
	if root.get_texture().get_image().save_png(directory.path_join(name+".png"))!=OK:fail("Capture failed")
func _run() -> void:
	directory=OS.get_environment("NILE_TEMPLE_MONUMENT_DIR")
	kind=OS.get_environment("NILE_TEMPLE_KIND")
	expected_timber=3200 if kind=="suntemple" else 1600
	expected_sandstone=38000 if kind=="suntemple" else 49600
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
	game.focus=game._iso(30,26) if kind=="suntemple" else game._iso(35,24)
	game._set_view(0.62)
	await game._smoke_choose_tool(kind)
	await game._click_cell(Vector2i(24,20))
	if not await game._wait_idle() or not game._has_building_kind(kind):fail("Monument menu placement failed");return
	await game._smoke_choose_tool("inspect")
	game.show_footprints=true
	game.overlay.queue_redraw()
	await capture("01-foundation")
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
	for sample in 550:
		game._queue_command("step",{"ticks":48})
		if not await game._wait_idle():fail("Monument work failed: "+str(game.last_reply.get("error","no reply")));return
		var b: Dictionary=game._building_at(Vector2i(24,20))
		var name:=""
		var percent:=int(b.get("monument_percent",0))
		if percent>=6 and percent<35:name="02-first-courses"
		elif percent>=35 and percent<70:name="03-building"
		elif percent>=70 and percent<100:name="04-finishing"
		elif str(b.get("stage",""))=="complete":name="05-complete"
		if not name.is_empty() and not seen.has(name):
			seen[name]=true
			if game.city_ui.completion_modal.visible:
				await game._smoke_click_button(game.city_ui.completion_continue)
				await game._wait_idle()
			await game._click_cell(Vector2i(25,26))
			if game.selected.is_empty() or game.selected.get("building",{}).get("kind","")!=kind:fail("Monument inspection did not open");return
			await capture(name)
			await create_timer(0.5).timeout
		if str(b.get("stage",""))=="complete":break
	var final: Dictionary=game._building_at(Vector2i(24,20))
	if str(final.get("stage",""))!="complete" or int(final.get("monument_materials",{}).get("timber",0))!=expected_timber or int(final.get("monument_materials",{}).get("sandstone",0))!=expected_sandstone:fail("Monument did not complete its original carving jobs and timber deliveries");return
	var info_click:=InputEventMouseButton.new()
	info_click.button_index=MOUSE_BUTTON_RIGHT
	info_click.position=game.get_canvas_transform()*game._iso(float(Vector2i(25,26).x)+0.5,float(Vector2i(25,26).y)+0.5)
	info_click.pressed=true
	root.push_input(info_click,true)
	await process_frame
	info_click.pressed=false
	root.push_input(info_click,true)
	await process_frame
	if not game.city_ui.inspector_modal.visible or game.selected.get("building",{}).get("kind","")!=kind:fail("Right-click monument inspector failed");return
	await capture("06-construction-foreman")
	info_click.pressed=true
	root.push_input(info_click,true)
	await process_frame
	await game._smoke_click_button(game.save_button)
	if not await game._wait_idle():fail("Save failed");return
	var saved: Dictionary=game.state.duplicate(true)
	await game._smoke_click_button(game.load_button)
	if not await game._wait_idle() or game.state!=saved:fail("Monument save/load changed state");return
	print("NILE_TEMPLE_MONUMENT_PASS ",kind," UI placement, physical Work Camp, Carpenters and Masons, sandstone conservation, SQLite replay")
	game.sim.stop()
	quit(0)
