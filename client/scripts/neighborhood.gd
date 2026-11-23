extends Node3D
## Original procedural visual study. No housing or service rules are implemented.
const Transport = preload("res://scripts/sidecar.gd")
var sim: Node
var camera: Camera3D
var walker: Node3D
var readout: Label
var notice: Label
var controls: Array[Button] = []
var state: Dictionary = {}
var last_reply: Dictionary = {}
var focus := Vector3(-1, 0, 0)
var smoke_mode := false
var restarting := false
var response_count := 0
var failure_count := 0

func material(color: String) -> StandardMaterial3D:
	var result := StandardMaterial3D.new()
	result.albedo_color = Color(color)
	result.roughness = 0.95
	return result

func box(parent: Node3D, at: Vector3, size: Vector3, color: String) -> MeshInstance3D:
	var mesh := MeshInstance3D.new()
	var shape := BoxMesh.new()
	shape.size = size
	mesh.mesh = shape
	mesh.material_override = material(color)
	mesh.position = at
	parent.add_child(mesh)
	return mesh

func cylinder(parent: Node3D, at: Vector3, radius: float, height: float, color: String) -> MeshInstance3D:
	var mesh := MeshInstance3D.new()
	var shape := CylinderMesh.new()
	shape.top_radius = radius * 0.85
	shape.bottom_radius = radius
	shape.height = height
	shape.radial_segments = 10
	mesh.mesh = shape
	mesh.material_override = material(color)
	mesh.position = at
	parent.add_child(mesh)
	return mesh

func house(at: Vector3, variant: int) -> void:
	var home := Node3D.new()
	add_child(home)
	home.position = at
	var height := 1.7 + variant * 0.3
	var clay: String = ["ae7e52", "ba8c5b", "c09a69"][variant % 3]
	box(home, Vector3(0, height / 2, 0), Vector3(2.7, height, 2.45), clay)
	box(home, Vector3(0, height + .04, 0), Vector3(2.72, .12, 2.5), "d0b58a")
	for x in [-1.3, 1.3]:
		box(home, Vector3(x, height + .22, 0), Vector3(.16, .38, 2.6), clay)
	for z in [-1.2, 1.2]:
		box(home, Vector3(0, height + .22, z), Vector3(2.6, .38, .16), clay)
	box(home, Vector3(-.45, .55, 1.24), Vector3(.5, 1.1, .05), "493d32")
	box(home, Vector3(-.45, 1.16, 1.31), Vector3(.7, .16, .18), "ddc496")
	box(home, Vector3(.65, 1.25, 1.24), Vector3(.36, .36, .04), "61513d")
	box(home, Vector3(-.45, .08, 1.48), Vector3(.8, .16, .6), "c6aa7e")
	cylinder(home, Vector3(1.03, .34, 1.55), .22, .65, "986749")
	if variant == 1:
		box(home, Vector3(.1, .7, -1.8), Vector3(2.6, 1.4, 1.1), clay)

func palm(at: Vector3, angle: float) -> void:
	var tree := Node3D.new()
	add_child(tree)
	tree.position = at
	cylinder(tree, Vector3(0, 1.85, 0), .16, 3.7, "7b6341")
	for i in range(7):
		var leaf := box(tree, Vector3.ZERO, Vector3(.4, .08, 2.2), "657950")
		var yaw := angle + i * TAU / 7
		leaf.position = Vector3(sin(yaw) * .8, 3.65, cos(yaw) * .8)
		leaf.rotation = Vector3(.2, yaw, .08)

func terrain() -> void:
	box(self, Vector3(0, -.35, 0), Vector3(64, .6, 48), "cbb083")
	# Floodplain and Nile run through the same world as the neighborhood.
	box(self, Vector3(11, -.02, 0), Vector3(8, .08, 48), "899264")
	box(self, Vector3(20, -.08, 0), Vector3(11, .12, 48), "437f85")
	box(self, Vector3(14.65, -.01, 0), Vector3(.8, .1, 48), "b8ac7e")
	for z in range(-23, 24, 3):
		box(self, Vector3(20 + sin(z) * 2, -.008, z), Vector3(4, .012, .035), "69989a")
	for x in range(-12, 5, 2):
		for z in [-8, 8]: box(self, Vector3(x, .01, z), Vector3(1.97, .08, 2), "ae9470")
	for z in range(-6, 8, 2):
		for x in [-12, 4]: box(self, Vector3(x, .01, z), Vector3(2, .08, 1.97), "ae9470")
	for x in range(6, 15, 2): box(self, Vector3(x, .01, 8), Vector3(2, .08, 2), "ae9470")
	# Crop rows are visual terrain reference only, not a simulated farm.
	for x in range(7, 14):
		box(self, Vector3(x, .06, -10), Vector3(.3, .12, 10), "777b45")
		for z in range(-14, -5): cylinder(self, Vector3(x, .3, z), .13, .45, "a4a36b")
	for at in [Vector3(12,0,3), Vector3(13,0,12), Vector3(10,0,-18), Vector3(-15,0,-10), Vector3(-15,0,7)]:
		palm(at, at.x)
	for i in range(4): house(Vector3(-8 + (i % 2) * 4, 0, -3 + (i / 2) * 5), i % 3)
	var granary: Node3D = load("res://assets/granary-reference.glb").instantiate()
	add_child(granary)
	granary.position = Vector3(-5, 0, -13)
	# Bazaar silhouette: a linen canopy, timber poles, jars and baskets.
	for x in [-1.8, 1.8]:
		for z in [-1.1, 1.1]: cylinder(self, Vector3(x, 1, 12 + z), .07, 2, "69523c")
	box(self, Vector3(0, 2.05, 12), Vector3(4.2, .12, 2.8), "b77a58")
	for x in [-1, 0, 1]:
		box(self, Vector3(x, .45, 12), Vector3(.8, .9, .8), "ac8659")
		cylinder(self, Vector3(x, 1, 12), .28, .25, "d2b375")
	# A diagnostic walker whose road-loop index comes only from Go.
	walker = Node3D.new()
	add_child(walker)
	cylinder(walker, Vector3(0, .63, 0), .19, .8, "eee2c5")
	cylinder(walker, Vector3(0, 1.17, 0), .16, .3, "896348")
	box(walker, Vector3(-.1, .16, 0), Vector3(.1, .32, .14), "896348")
	box(walker, Vector3(.1, .16, 0), Vector3(.1, .32, .14), "896348")

func _ready() -> void:
	get_tree().auto_accept_quit = false
	smoke_mode = "--smoke" in OS.get_cmdline_user_args()
	terrain()
	var environment := WorldEnvironment.new()
	var settings := Environment.new()
	settings.background_mode = Environment.BG_COLOR
	settings.background_color = Color("2c393a")
	settings.ambient_light_source = Environment.AMBIENT_SOURCE_COLOR
	settings.ambient_light_color = Color("ead9b6")
	settings.ambient_light_energy = .38
	settings.tonemap_mode = Environment.TONE_MAPPER_LINEAR
	environment.environment = settings
	add_child(environment)
	var sun := DirectionalLight3D.new()
	sun.rotation_degrees = Vector3(-55, -35, 0)
	sun.light_color = Color("fff1d4")
	sun.light_energy = .65
	sun.shadow_enabled = true
	add_child(sun)
	camera = Camera3D.new()
	camera.projection = Camera3D.PROJECTION_ORTHOGONAL
	camera.size = 39
	camera.far = 200
	camera.current = true
	add_child(camera)
	position_camera()
	interface()
	sim = Transport.new()
	add_child(sim)
	sim.response_received.connect(on_reply)
	sim.failed.connect(on_failure)
	if sim.start(): sim.request("state")
	if smoke_mode: run_smoke.call_deferred()

func position_camera() -> void:
	camera.position = focus + Vector3(42, 42, 52)
	camera.look_at(focus)
	camera.h_offset = 7

func label(text: String, size: int, color: String = "e7dbc3") -> Label:
	var item := Label.new()
	item.text = text
	item.add_theme_font_size_override("font_size", size)
	item.add_theme_color_override("font_color", Color(color))
	return item

func interface() -> void:
	var layer := CanvasLayer.new()
	add_child(layer)
	var header := PanelContainer.new()
	header.set_anchors_and_offsets_preset(Control.PRESET_TOP_WIDE)
	header.offset_bottom = 70
	var top_style := StyleBoxFlat.new()
	top_style.bg_color = Color("243b3b")
	top_style.content_margin_left = 28
	top_style.content_margin_top = 12
	top_style.content_margin_bottom = 12
	header.add_theme_stylebox_override("panel", top_style)
	layer.add_child(header)
	var title := VBoxContainer.new()
	header.add_child(title)
	title.add_child(label("N I L E", 24))
	title.add_child(label("A neighborhood by the river", 13, "b9c5ad"))
	var panel := PanelContainer.new()
	panel.set_anchors_and_offsets_preset(Control.PRESET_RIGHT_WIDE)
	panel.offset_left = -316
	panel.offset_top = 70
	panel.offset_bottom = -45
	var style := StyleBoxFlat.new()
	style.bg_color = Color("293b39")
	style.content_margin_left = 24
	style.content_margin_right = 24
	style.content_margin_top = 26
	panel.add_theme_stylebox_override("panel", style)
	layer.add_child(panel)
	var column := VBoxContainer.new()
	column.add_theme_constant_override("separation", 16)
	panel.add_child(column)
	column.add_child(label("FIRST VISUAL STUDY", 13, "cdb17b"))
	column.add_child(label("Riverbank neighborhood", 20))
	var description := label("Mudbrick homes, storage bins,\na shaded market and a road loop.\n\nOriginal study assets.\nArt direction awaits your review.", 15)
	column.add_child(description)
	column.add_child(HSeparator.new())
	column.add_child(label("LOCAL FOUNDATION", 13, "cdb17b"))
	readout = label("Starting…", 16)
	column.add_child(readout)
	for entry in [["Advance walker", "step"], ["Save civilization", "save"], ["Restore saved state", "load"]]:
		var button := Button.new()
		button.text = entry[0]
		button.custom_minimum_size.y = 42
		button.pressed.connect(send.bind(entry[1]))
		column.add_child(button)
		controls.append(button)
	var restart := Button.new()
	restart.text = "Restart and recover save"
	restart.custom_minimum_size.y = 38
	restart.pressed.connect(restart_sim)
	column.add_child(restart)
	notice = label("", 14, "d0bd94")
	notice.autowrap_mode = TextServer.AUTOWRAP_WORD_SMART
	notice.custom_minimum_size = Vector2(260, 68)
	column.add_child(notice)
	var spacer := Control.new()
	spacer.size_flags_vertical = Control.SIZE_EXPAND_FILL
	column.add_child(spacer)
	column.add_child(label("Wheel to zoom · Arrow keys to pan\nHome to reset view", 13, "aebcb0"))
	var footer := PanelContainer.new()
	footer.set_anchors_and_offsets_preset(Control.PRESET_BOTTOM_WIDE)
	footer.offset_top = -45
	footer.add_theme_stylebox_override("panel", top_style)
	layer.add_child(footer)
	footer.add_child(label("Kickoff study  ·  Walker movement and save/restore are diagnostic. Housing, services and farming are not simulated yet.", 14))

func send(op: String) -> void:
	if sim.request(op, 1):
		for button in controls: button.disabled = true

func restart_sim() -> void:
	sim.stop()
	restarting = true
	if sim.start(): sim.request("load")

func on_reply(reply: Dictionary) -> void:
	last_reply = reply
	response_count += 1
	for button in controls: button.disabled = false
	# Every response includes the current authoritative state, even failed loads.
	# A newly restarted process may have reset while the old view still shows work.
	state = reply.state
	readout.text = "Tick %d   ·   Grain %d\nWalker position %d / 32" % [state.tick, state.grain, state.walker]
	if not reply.get("ok", false):
		notice.text = str(reply.get("error", "Command failed"))
	else:
		notice.text = "Local state ready." if not restarting else "Recovered the last committed save."
	restarting = false
	var index := int(state.walker)
	var position_2d: Vector2
	if index < 8: position_2d = Vector2(-12 + index * 2, -8)
	elif index < 16: position_2d = Vector2(4, -8 + (index - 8) * 2)
	elif index < 24: position_2d = Vector2(4 - (index - 16) * 2, 8)
	else: position_2d = Vector2(-12, 8 - (index - 24) * 2)
	walker.position = Vector3(position_2d.x, .08, position_2d.y)

func on_failure(reason: String) -> void:
	failure_count += 1
	notice.text = reason
	for button in controls: button.disabled = true

func _unhandled_input(event: InputEvent) -> void:
	if event is InputEventMouseButton and event.pressed:
		if event.button_index == MOUSE_BUTTON_WHEEL_UP: camera.size = maxf(18, camera.size - 2)
		if event.button_index == MOUSE_BUTTON_WHEEL_DOWN: camera.size = minf(65, camera.size + 2)
	if event is InputEventKey and event.pressed:
		match event.keycode:
			KEY_LEFT: focus += Vector3(-2, 0, 2)
			KEY_RIGHT: focus += Vector3(2, 0, -2)
			KEY_UP: focus += Vector3(-2, 0, -2)
			KEY_DOWN: focus += Vector3(2, 0, 2)
			KEY_HOME:
				focus = Vector3(-1, 0, 0)
				camera.size = 39
		position_camera()

func _notification(what: int) -> void:
	if what == NOTIFICATION_WM_CLOSE_REQUEST:
		sim.stop()
		get_tree().quit()

func wait_response(previous: int) -> bool:
	var until := Time.get_ticks_msec() + 7000
	while response_count == previous and Time.get_ticks_msec() < until:
		await get_tree().process_frame
	return response_count > previous and bool(last_reply.get("ok", false))

func checked_request(op: String, ticks: int = 0) -> bool:
	var previous := response_count
	if not sim.request(op, ticks): return false
	return await wait_response(previous)

func run_smoke() -> void:
	if not await wait_response(0): return smoke_fail("startup")
	# Restart before any save: failed load must display the new process's tick zero.
	if not await checked_request("step", 5): return smoke_fail("unsaved step")
	var before_unsaved_restart := response_count
	restart_sim()
	if await wait_response(before_unsaved_restart): return smoke_fail("unexpected initial save")
	if response_count == before_unsaved_restart or state.tick != 0 or state.walker != 0:
		return smoke_fail("failed recovery did not resynchronize state")
	if restarting or walker.position != Vector3(-12, .08, -8):
		return smoke_fail("failed recovery left stale presentation")
	if not await checked_request("step", 11): return smoke_fail("step")
	if not await checked_request("save"): return smoke_fail("save")
	var saved_hash: String = last_reply.hash
	var saved_position := walker.position
	if not await checked_request("step", 7): return smoke_fail("second step")
	if walker.position == saved_position: return smoke_fail("walker did not move")
	if not await checked_request("load"): return smoke_fail("load")
	if last_reply.hash != saved_hash or walker.position != saved_position: return smoke_fail("restore mismatch")
	# Kill the actual child, then recover through the same UI action.
	# Use an external kill so Godot can observe/reap the failed child normally.
	var failures_before := failure_count
	if OS.execute("kill", ["-KILL", str(sim.child.pid)]) != 0: return smoke_fail("kill command failed")
	var deadline := Time.get_ticks_msec() + 3000
	while failure_count == failures_before and Time.get_ticks_msec() < deadline:
		await get_tree().process_frame
	if failure_count == failures_before or not sim.child.is_empty(): return smoke_fail("child crash was not observed")
	var previous := response_count
	restart_sim()
	if not await wait_response(previous): return smoke_fail("restart")
	if last_reply.hash != saved_hash: return smoke_fail("restart hash mismatch")
	notice.text = "Verified: advance → save → restore\nand crash → restart → restore."
	var output := OS.get_environment("NILE_CAPTURE_DIR")
	if not output.is_empty():
		for size in [Vector2i(1920,1080), Vector2i(1280,720), Vector2i(2560,1440), Vector2i(3456,2160), Vector2i(3440,1440)]:
			get_window().size = size
			for i in range(5): await get_tree().process_frame
			await RenderingServer.frame_post_draw
			var path := output.path_join("neighborhood-%dx%d.png" % [size.x,size.y])
			var captured := get_viewport().get_texture().get_image()
			if captured.get_size() != size: return smoke_fail("capture dimensions %s != %s" % [captured.get_size(), size])
			if captured.save_png(path) != OK: return smoke_fail("capture")
	print("NILE_GODOT_SMOKE_PASS ", saved_hash)
	sim.stop()
	get_tree().quit(0)

func smoke_fail(message: String) -> void:
	push_error("NILE_GODOT_SMOKE_FAIL " + message)
	sim.stop()
	get_tree().quit(1)
