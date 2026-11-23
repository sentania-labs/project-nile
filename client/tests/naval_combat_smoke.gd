extends SceneTree
## A normally equipped river town intercepts a physical enemy landing force.
var game: Node
var directory:="/mnt/holding/project-nile/naval-combat-review-20261123"
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
	if not _check(game.startup_done and not game.paused,"Naval town failed to load"):return
	await game._smoke_click_button(game.pause_button)
	if not _check(await game._wait_idle() and game.paused,"Naval clock failed to pause"):return
	await game._smoke_click_button(game.city_ui.overseers_button)
	await game._smoke_click_button(game.city_ui.overseer_tabs["River"])
	var controls=game.city_ui.naval_controls
	var index:=-1
	for i in range(controls.vessels.item_count):
		if controls.vessels.get_item_text(i).begins_with("Warship"):index=i
	if not _check(index>=0,"Real economy did not construct warship"):return
	controls.vessels.select(index)
	controls.x.value=3
	controls.y.value=8
	await game._smoke_click_button(controls.sail)
	if not _check(await game._wait_idle(),"Warship station order failed"):return
	game.city_ui.overseer_modal.hide()
	if not _check(await _step(32),"Warship failed approach"):return
	await game._smoke_click_button(game.city_ui.overseers_button)
	await game._smoke_click_button(game.city_ui.overseer_tabs["River"])
	await game._smoke_click_button(controls.seek)
	if not _check(await game._wait_idle(),"Warship seek order failed"):return
	await game._smoke_click_button(controls.raid)
	if not _check(await game._wait_idle() and game.state.military.invasion_active,"Enemy ships did not enter world"):return
	game.city_ui.overseer_modal.hide()
	game.focus=game._iso(6,9)
	game._set_view(1.1)
	if not _check(await _step(4),"Naval approach failed"):return
	await _capture("naval-interception.png")
	var saw_passengers:=false
	var saw_damage:=false
	for i in range(320):
		for b: Dictionary in game.state.boats:
			if b.kind=="enemytransport" and int(b.soldiers)>0:saw_passengers=true
			if b.kind=="warship" and int(b.hull)<80 and not saw_damage:
				saw_damage=true
				game.focus=game._iso(float(b.x)+4,float(b.y)+2)
				await _capture("ramming-and-hull-damage.png")
		for b: Dictionary in game.state.boats:
			if b.kind=="warship" and b.get("order","")!="seek" and b.stage!="commission" and b.stage!="repair":
				await game._smoke_click_button(game.city_ui.overseers_button)
				await game._smoke_click_button(game.city_ui.overseer_tabs["River"])
				for j in range(controls.vessels.item_count):
					if int(controls.vessels.get_item_metadata(j))==int(b.id):controls.vessels.select(j)
				await game._smoke_click_button(controls.seek)
				if not _check(await game._wait_idle(),"Replacement warship order failed"):return
				game.city_ui.overseer_modal.hide()
		if not game.state.military.invasion_active:break
		if not _check(await _step(2),"Naval combat failed"):return
	if not _check(saw_passengers and saw_damage and game.state.military.defeated==8 and game.state.military.breaches==0 and not game.state.military.invasion_active,"Navy failed to intercept transported attackers"):return
	game._queue_command("save")
	if not _check(await game._wait_idle(),"Naval save failed"):return
	var saved_hash:=str(game.last_reply.hash)
	if not _check(await _step(8),"Naval post-save failed"):return
	game._queue_command("load")
	if not _check(await game._wait_idle() and str(game.last_reply.hash)==saved_hash,"Naval reload changed ships or casualties"):return
	await game._smoke_click_button(game.city_ui.overseers_button)
	await game._smoke_click_button(game.city_ui.overseer_tabs["River"])
	await _capture("naval-interception-overseer.png")
	print("NILE_NAVAL_COMBAT_PASS ",JSON.stringify(game.state.naval)," ",JSON.stringify(game.state.military))
	quit(0)
