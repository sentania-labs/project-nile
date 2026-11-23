extends SceneTree
var game: Node
var directory: String
var seen := {}
var kind: String
func _initialize() -> void: _run.call_deferred()
func fail(message: String) -> void:
	push_error(message)
	if game != null and game.sim != null:game.sim.stop()
	quit(1)
func capture(name: String) -> void:
	await RenderingServer.frame_post_draw
	if root.get_texture().get_image().save_png(directory.path_join(name+".png"))!=OK:fail("Capture failed")
func _run() -> void:
	directory=OS.get_environment("NILE_ROYAL_TOMB_DIR")
	kind="smallroyaltomb"
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
	game.focus=game._iso(24,21)
	game._set_view(1.1)
	await game._smoke_choose_tool(kind)
	await game._click_cell(Vector2i(23,20))
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
		var b: Dictionary=game._building_at(Vector2i(23,20))
		var name:=""
		var percent:=int(b.get("monument_percent",0))
		if percent>=6 and percent<35:name="02-excavation"
		elif percent>=35 and percent<70:name="03-chambers"
		elif percent>=70 and percent<100:name="04-decoration"
		elif str(b.get("stage",""))=="complete":name="05-complete"
		if not name.is_empty() and not seen.has(name):
			seen[name]=true
			if game.city_ui.completion_modal.visible:
				await game._smoke_click_button(game.city_ui.completion_continue)
				await game._wait_idle()
			await game._click_cell(Vector2i(24,22))
			if game.selected.is_empty() or game.selected.get("building",{}).get("kind","")!=kind:fail("Monument inspection did not open");return
			await capture(name)
			await create_timer(0.5).timeout
		if str(b.get("stage",""))=="complete":break
	var final: Dictionary=game._building_at(Vector2i(23,20))
	var materials: Dictionary=final.get("monument_materials",{})
	if str(final.get("stage",""))!="complete" or int(materials.get("lamps",0))!=1600 or int(materials.get("paint",0))!=80 or int(materials.get("clay",0))!=80:fail("Royal Tomb did not complete its physical supply chain");return
	if int(game.state.get("resource_produced",{}).get("lamps",0))<1600 or int(game.state.get("resource_produced",{}).get("paint",0))<80:fail("Tomb bypassed the workshops");return
	var info_click:=InputEventMouseButton.new()
	info_click.button_index=MOUSE_BUTTON_RIGHT
	info_click.position=game.get_canvas_transform()*game._iso(float(Vector2i(24,22).x)+0.5,float(Vector2i(24,22).y)+0.5)
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
	print("NILE_ROYAL_TOMB_PASS cliff placement, local lamp and paint industries, physical haulers and supplied artisans, staged construction, SQLite replay")
	game.sim.stop()
	quit(0)
