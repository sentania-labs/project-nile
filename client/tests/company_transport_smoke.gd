extends SceneTree
## Board a normally recruited company, sail and disembark through map controls.
var game: Node
var directory:="/mnt/holding/project-nile/company-transport-review-20261123"
func _initialize() -> void:_run.call_deferred()
func _check(ok: bool,why: String) -> bool:
	if not ok:push_error(why);quit(1)
	return ok
func _step(ticks: int) -> bool:
	game._queue_command("step",{"ticks":ticks})
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
	while (not game.startup_done or game.paused) and Time.get_ticks_msec()<deadline:await process_frame
	if not _check(game.startup_done and not game.paused,"Transport settlement failed to load"):return
	await game._smoke_click_button(game.pause_button)
	if not _check(await game._wait_idle() and game.paused,"Transport clock failed to pause"):return
	await game._smoke_click_button(game.city_ui.overseers_button)
	await game._smoke_click_button(game.city_ui.overseer_tabs["River"])
	var controls=game.city_ui.naval_controls
	var index:=-1
	for i in range(controls.vessels.item_count):
		if controls.vessels.get_item_text(i).begins_with("Transportship"):index=i
	if not _check(index>=0 and controls.company_picker.item_count==1,"Transport or company selector missing"):return
	controls.vessels.select(index)
	controls.company_picker.select(0)
	await game._smoke_click_button(controls.transport_pick)
	game.focus=game._iso(8,28)
	game._set_view(1.2)
	await game._click_cell(Vector2i(6,30))
	if not _check(await game._wait_idle() and not game.state.deployments.is_empty(),"Map landing order failed: "+str(game.last_reply.get("error",""))+" picking "+str(controls.picking)):return
	var aboard:=false
	for i in range(100):
		if not _check(await _step(1),"Company boarding journey failed"):return
		for b: Dictionary in game.state.boats:
			if b.kind=="transportship" and int(b.soldiers)>=8:
				aboard=true
				game.focus=game._iso(float(b.x)+4,float(b.y))
				await _capture("company-aboard.png")
		if aboard:break
	if not _check(aboard,"Soldiers never boarded the ship"):return
	game._queue_command("save")
	if not _check(await game._wait_idle(),"Passenger save failed"):return
	var hash:=str(game.last_reply.hash)
	if not _check(await _step(4),"Passenger post-save step failed"):return
	game._queue_command("load")
	if not _check(await game._wait_idle() and str(game.last_reply.hash)==hash,"Passenger reload changed company"):return
	if not _check(await _step(40),"Transport crossing failed"):return
	if not _check(game.state.deployments[0].status=="landed","Company failed to disembark"):return
	game.focus=game._iso(8,29)
	await _capture("company-landed.png")
	await game._smoke_click_button(game.city_ui.overseers_button)
	await game._smoke_click_button(game.city_ui.overseer_tabs["Military"])
	await game._smoke_click_button(game.city_ui.military_controls.recall)
	if not _check(await game._wait_idle(),"Disembarked company recall failed"):return
	game.city_ui.overseer_modal.hide()
	if not _check(await _step(90),"Company return march failed"):return
	if not _check(game.state.companies[0].stationed>=8,"Transported soldiers did not return home"):return
	print("NILE_COMPANY_TRANSPORT_PASS ",JSON.stringify(game.state.deployments))
	quit(0)
