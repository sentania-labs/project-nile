extends SceneTree
## A live town, equipped through its economy, exercises tactical and defense UI.
var game: Node
var directory := "/mnt/holding/project-nile/combined-arms-review-20261123"

func _initialize() -> void: _run.call_deferred()

func _check(ok: bool, reason: String) -> bool:
	if not ok:
		push_error(reason)
		quit(1)
	return ok

func _step(ticks: int) -> bool:
	game._queue_command("step", {"ticks":ticks})
	return await game._wait_idle()

func _capture(name: String) -> void:
	await process_frame
	await RenderingServer.frame_post_draw
	root.get_texture().get_image().save_png(directory.path_join(name))

func _place(kind: String, cell: Vector2i) -> bool:
	game.focus=game._iso(cell.x,cell.y)
	game._set_view(1.1)
	await game._smoke_choose_tool(kind)
	await game._click_cell(cell)
	var ok: bool=await game._wait_idle() and game._building_at(cell).get("kind","")==kind
	if not ok: print("BUILD_FAILURE ",kind," ",cell," tool ",game.active_tool," ",game.last_reply.get("error",""))
	return ok

func _company(kind: String) -> Dictionary:
	for company: Dictionary in game.state.companies:
		if company.get("kind","")==kind:return company
	return {}

func _run() -> void:
	DirAccess.make_dir_recursive_absolute(directory)
	root.size=Vector2i(1920,1080)
	root.set_meta("nile_start_choice","load")
	game=load("res://scenes/sandbox.tscn").instantiate()
	root.add_child(game)
	var deadline:=Time.get_ticks_msec()+20000
	while (not game.startup_done or game.paused) and Time.get_ticks_msec()<deadline:await process_frame
	if not _check(game.startup_done and not game.paused,"Combined-arms town did not load"):return
	await game._smoke_click_button(game.pause_button)
	if not _check(await game._wait_idle() and game.paused,"Combined-arms clock did not pause"):return
	if not _check(_company("archer").get("stationed",0)>=8 and _company("charioteer").get("stationed",0)>=8,"Real economy did not provide both trained companies"):return
	game.focus=game._iso(19,15)
	game._set_view(0.8)
	await _capture("combined-arms-city.png")
	for cell: Vector2i in [Vector2i(28,15),Vector2i(29,15),Vector2i(28,16),Vector2i(29,16)]:
		if not _check(await _place("wall",cell),"Wall placement through palette failed"):return
	if not _check(await _place("tower",Vector2i(28,15)),"Tower wall-foundation placement failed"):return
	var roads: Array=[]
	for x in range(22,28):roads.append({"x":x,"y":12})
	for y in range(13,16):roads.append({"x":27,"y":y})
	game._queue_command("road",{"tiles":roads})
	if not _check(await game._wait_idle(),"Tower access road failed"):return
	for y in range(17,22):
		if not _check(await _place("wall",Vector2i(28,y)),"Fortification wall placement failed"):return
	roads=[]
	for y in range(20,23):roads.append({"x":22,"y":y})
	for x in range(23,30):roads.append({"x":x,"y":22})
	game._queue_command("road",{"tiles":roads})
	if not _check(await game._wait_idle(),"Gate access road failed"):return
	if not _check(await _place("gatehouse",Vector2i(28,22)),"Gatehouse placement failed"):return
	await game._smoke_choose_tool("inspect")
	if not _check(await _step(150),"Sentry recruitment failed"):return
	if not _check(game.state.military.get("sentries_recruited",0)>=2,"Tower lacks recruited sentries"):return
	game.focus=game._iso(27,18)
	game._set_view(1.2)
	await _capture("walls-gate-and-tower.png")
	await game._smoke_click_button(game.city_ui.overseers_button)
	await game._smoke_click_button(game.city_ui.overseer_tabs["Military"])
	var controls=game.city_ui.military_controls
	if not _check(controls.visible and controls.companies.item_count==2,"Combined-arms controls missing"):return
	controls.companies.select(0)
	controls.formation.select(2)
	controls.x.value=25
	controls.y.value=25
	await game._smoke_click_button(controls.place)
	game.focus=game._iso(25,25)
	await game._click_cell(Vector2i(25,25))
	if not _check(await game._wait_idle() and _company("archer").order=="loose","Loose archer formation order failed"):return
	await game._smoke_click_button(game.city_ui.overseers_button)
	await game._smoke_click_button(game.city_ui.overseer_tabs["Military"])
	await game._smoke_click_button(controls.rotate)
	if not _check(await game._wait_idle() and bool(_company("archer").get("rotated",false)),"Formation rotation failed"):return
	controls.companies.select(1)
	controls.formation.select(3)
	controls.x.value=27
	controls.y.value=18
	await game._smoke_click_button(controls.muster)
	if not _check(await game._wait_idle() and _company("charioteer").order=="engage","Chariot engagement order failed"):return
	game.city_ui.overseer_modal.hide()
	if not _check(await _step(55),"Both companies failed to march"):return
	game.focus=game._iso(27,23)
	game._set_view(1.2)
	await _capture("archers-and-chariots.png")
	await game._smoke_click_button(game.city_ui.overseers_button)
	await game._smoke_click_button(game.city_ui.overseer_tabs["Military"])
	await game._smoke_click_button(controls.raid)
	if not _check(await game._wait_idle() and game.state.military.invasion_active,"Recorded invasion failed"):return
	game.city_ui.overseer_modal.hide()
	if not _check(await _step(4),"Invasion progression failed"):return
	game.focus=game._iso(29,16)
	await _capture("tower-and-company-intercept.png")
	game._queue_command("save")
	if not _check(await game._wait_idle(),"Combat save failed"):return
	var saved_hash:=str(game.last_reply.hash)
	if not _check(await _step(4),"Combat progression failed"):return
	game._queue_command("load")
	if not _check(await game._wait_idle() and str(game.last_reply.hash)==saved_hash,"Combat reload changed state"):return
	for i in range(60):
		if not game.state.military.invasion_active:break
		if not _check(await _step(4),"Combined defense progression failed"):return
	if not _check(game.state.military.defeated==4 and game.state.military.breaches==0,"Combined defense did not repel the invasion"):return
	await game._smoke_click_button(game.city_ui.overseers_button)
	await game._smoke_click_button(game.city_ui.overseer_tabs["Military"])
	await _capture("combined-arms-victory.png")
	for i in range(2):
		controls.companies.select(i)
		await game._smoke_click_button(controls.recall)
		if not _check(await game._wait_idle(),"Company recall failed"):return
	game.city_ui.overseer_modal.hide()
	if not _check(await _step(100),"Return journey failed"):return
	for c: Dictionary in game.state.companies:
		if not _check(c.stationed==c.soldiers,"Company did not return to fort"):return
	print("NILE_COMBINED_ARMS_PASS ",JSON.stringify(game.state.military))
	quit(0)
