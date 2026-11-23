extends RefCounted


static func run(game: Node) -> void:
	var ui = game.city_ui.campaign_ui
	var deadline := Time.get_ticks_msec() + 15000
	while not ui.briefing and Time.get_ticks_msec() < deadline:
		await game.get_tree().process_frame
	if not game._smoke_require(ui.briefing and game.paused and "Nubt" in ui.heading.text, "Nubt briefing did not open"): return
	var archive := OS.get_environment("NILE_CAMPAIGN_CAPTURE")
	if not archive.is_empty():
		game.get_window().size = Vector2i(1920, 1080)
		await game._save_capture(archive, "nubt-briefing.png", Vector2i(1920, 1080))
	await game._smoke_click_button(ui.decisions.get_child(-1))
	if not game._smoke_require(await game._wait_idle() and not game.paused, "Briefing did not start city clock"): return
	await game._smoke_click_button(game.pause_button)
	if not game._smoke_require(await game._wait_idle() and game.paused, "Could not pause city"): return
	game.focus = game._iso(10, 8)
	game._set_view(0.72)
	await game._drag_cells("hut", Vector2i(9, 1), Vector2i(11, 15))
	if not game._smoke_require(await game._wait_idle(), "Could not zone Nubt"): return
	if not game._smoke_require(game.state.buildings.size() == 30, "Housing drag did not reserve thirty plots"): return
	game.focus = game._iso(11, 22)
	game._set_view(0.75)
	var choices := [["firehouse",9,17],["architect",9,19],["granary",11,21],["huntinglodge",11,26],["bazaar",11,18],["watersupply",11,29]]
	while int(game.state.tick) < 8000 and not game.state.campaign.won:
		for choice: Array in choices:
			if not game.city_ui.category_for(choice[0]).is_empty() and not game._has_building_kind(choice[0]):
				await game._smoke_choose_tool(choice[0])
				await game._click_cell(Vector2i(choice[1],choice[2]))
				if not game._smoke_require(await game._wait_idle() and game._has_building_kind(choice[0]), "Failed placing " + choice[0] + " " + game.placement_notice.text): return
		for b: Dictionary in game.state.buildings:
			if not str(b.get("damaged","")).is_empty():
				game._queue_command("repair",{"x": int(b.x),"y": int(b.y)})
				if not game._smoke_require(await game._wait_idle(),"Could not repair Nubt building"): return
		game._queue_command("step",{"ticks":32})
		if not game._smoke_require(await game._wait_idle(),"Mission simulation failed"): return
	if not game._smoke_require(game.state.campaign.won and ui.panel.visible and game.state.resource_produced.meat > 0, "Nubt did not reach victory"): return
	if not archive.is_empty(): await game._save_capture(archive,"nubt-victory.png",Vector2i(1920,1080))
	var victory_tick := int(game.state.campaign.victory_tick)
	if ui.decisions.get_child_count() > 1:
		await game._smoke_click_button(ui.decisions.get_child(0))
		if not game._smoke_require(await game._wait_idle() and game.state.campaign.name == "Thinis" and ui.briefing and game.paused,"Could not advance to Thinis"): return
		if not archive.is_empty(): await game._save_capture(archive,"thinis-briefing.png",Vector2i(1920,1080))
	print("NILE_CAMPAIGN_PASS Nubt zoning hunting granaries water housing victory ", victory_tick)
	game.sim.stop()
	game.get_tree().quit()
