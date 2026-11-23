extends Node

const SAVE_PATH := "user://preferences.cfg"
const TRACKS := [
	"res://assets/music/first_light.ogg",
	"res://assets/music/river_market.ogg",
	"res://assets/music/evening_barges.ogg"
]
const DEFAULTS := {
	"Pan up": KEY_UP, "Pan down": KEY_DOWN, "Pan left": KEY_LEFT, "Pan right": KEY_RIGHT,
	"Zoom in": KEY_EQUAL, "Zoom out": KEY_MINUS, "Rotate left": KEY_Q, "Rotate right": KEY_E,
	"Rotate building":KEY_R, "Show cliffs":KEY_H, "Income tax":KEY_X, "Malaria risk":KEY_Y, "Disease risk":KEY_Z,
}
const PACES := {"classic50": "Pharaoh 50%", "classic80":"Pharaoh 80%", "playtest": "Playtest", "classic100": "Pharaoh 100%"}
signal pace_changed(pace: String)

const RESERVED := [KEY_ESCAPE, KEY_ENTER, KEY_SPACE, KEY_F, KEY_TAB, KEY_SHIFT, KEY_CTRL, KEY_ALT, KEY_META, KEY_F1, KEY_F2, KEY_F3]

var preferences := ConfigFile.new()
var bindings := DEFAULTS.duplicate()
var music := AudioStreamPlayer.new()
var river := AudioStreamPlayer.new()
var toolsound := AudioStreamPlayer.new()
var sound_level := 0.6
var river_proximity := 0.0
var pace := "classic80"
var track := 0
var settings_window: Window
var buttons: Dictionary = {}
var pending_action := ""
var notice: Label
var held_pan: Dictionary = {}
var camera_game: Node


func _ready() -> void:
	preferences.load(SAVE_PATH)
	pace=str(preferences.get_value("time","pace","classic80"))
	if not PACES.has(pace): pace="classic80"
	var used := {}
	# Preserve explicit bindings from older preferences when new actions arrive.
	for action: String in DEFAULTS:
		if not preferences.has_section_key("camera",action): continue
		var code:=int(preferences.get_value("camera",action))
		if code>0 and code not in RESERVED and not used.has(code):
			bindings[action]=code
			used[code]=action
	for action: String in DEFAULTS:
		if used.values().has(action): continue
		var code:=int(DEFAULTS[action])
		if used.has(code):
			for fallback in [KEY_T,KEY_U,KEY_I,KEY_O,KEY_P,KEY_J,KEY_K,KEY_L,KEY_B,KEY_N,KEY_M,KEY_V,KEY_G]:
				if not used.has(fallback) and not DEFAULTS.values().has(fallback):
					code=fallback
					break
		bindings[action]=code
		used[code]=action
	add_child(music)
	add_child(river)
	add_child(toolsound)
	sound_level = clampf(float(preferences.get_value("audio","effects",0.6)),0.0,1.0)
	toolsound.stream = load("res://assets/sounds/place.ogg")
	river.stream = load("res://assets/sounds/riverbank.ogg")
	river.finished.connect(func() -> void: if river_proximity>0.01: river.play())
	music.finished.connect(_next_track)
	_set_music(float(preferences.get_value("audio", "music", 0.55)))
	if AudioServer.get_driver_name() != "Dummy":
		music.stream = load(TRACKS[0])
		music.play()


func _next_track() -> void:
	track = (track + 1) % TRACKS.size()
	music.stream = load(TRACKS[track])
	music.play()


func _set_music(linear: float) -> void:
	music.volume_db = linear_to_db(clampf(linear, 0.0, 1.0))


func set_river_distance(tiles: float) -> void:
	river_proximity = clampf(1.0-tiles/20.0,0.0,1.0)
	river.volume_db = linear_to_db(maxf(0.0001,0.22*sound_level*river_proximity))
	if river_proximity*sound_level<0.01: river.stop()
	elif not river.playing and AudioServer.get_driver_name()!="Dummy": river.play()


func placement_sound() -> void:
	if sound_level==0.0 or AudioServer.get_driver_name()=="Dummy": return
	toolsound.volume_db=linear_to_db(0.29*sound_level)
	toolsound.play()


func _save_preferences() -> void:
	if preferences.save(SAVE_PATH) != OK:
		push_warning("Could not save local sound and control preferences")


func is_open() -> bool:
	return is_instance_valid(settings_window) and settings_window.visible


func open_settings() -> void:
	if is_open(): return
	held_pan.clear()
	pending_action = ""
	if not is_instance_valid(settings_window):
		_build_window()
	settings_window.popup_centered()
	_refresh_keys()


func _build_window() -> void:
	settings_window = Window.new()
	settings_window.title = "Sound and controls"
	settings_window.size = Vector2i(410, 480)
	settings_window.transient = true
	settings_window.exclusive = true
	settings_window.visible = false
	add_child(settings_window)
	settings_window.close_requested.connect(settings_window.hide)
	settings_window.window_input.connect(_settings_input)
	var panel := PanelContainer.new()
	settings_window.add_child(panel)
	panel.set_anchors_and_offsets_preset(Control.PRESET_FULL_RECT)
	var paper := StyleBoxFlat.new()
	paper.bg_color = Color("ecd2a4")
	paper.border_color = Color("83552d")
	paper.set_border_width_all(2)
	for edge in [SIDE_LEFT, SIDE_RIGHT, SIDE_TOP, SIDE_BOTTOM]:
		paper.set_content_margin(edge, 16)
	panel.add_theme_stylebox_override("panel", paper)
	var column := VBoxContainer.new()
	panel.add_child(column)
	var scroll:=ScrollContainer.new()
	scroll.size_flags_vertical=Control.SIZE_EXPAND_FILL
	column.add_child(scroll)
	var grid := GridContainer.new()
	grid.size_flags_horizontal=Control.SIZE_EXPAND_FILL
	grid.columns = 2
	scroll.add_child(grid)
	grid.add_child(_label("Calendar pace"))
	var calendar := OptionButton.new()
	for id: String in PACES:
		calendar.add_item(PACES[id])
		calendar.set_item_metadata(calendar.item_count-1,id)
		if id==pace: calendar.select(calendar.item_count-1)
	calendar.item_selected.connect(func(index: int) -> void:
		pace=str(calendar.get_item_metadata(index))
		preferences.set_value("time","pace",pace)
		_save_preferences()
		pace_changed.emit(pace))
	grid.add_child(calendar)
	grid.add_child(_label("Music volume"))
	var volume := HSlider.new()
	volume.size_flags_horizontal = Control.SIZE_EXPAND_FILL
	volume.min_value = 0
	volume.max_value = 100
	volume.step = 1
	volume.value = round(db_to_linear(music.volume_db) * 100)
	volume.value_changed.connect(func(value: float) -> void:
		_set_music(value / 100)
		preferences.set_value("audio", "music", value / 100)
		_save_preferences())
	grid.add_child(volume)
	grid.add_child(_label("City sounds"))
	var effects := HSlider.new()
	effects.min_value = 0
	effects.max_value = 100
	effects.value = round(sound_level*100)
	effects.value_changed.connect(func(value: float) -> void:
		sound_level = value/100
		preferences.set_value("audio","effects",sound_level)
		_save_preferences())
	grid.add_child(effects)
	for action: String in bindings:
		grid.add_child(_label(action))
		var key := Button.new()
		key.custom_minimum_size.x = 155
		key.pressed.connect(_listen_for_key.bind(action))
		grid.add_child(key)
		buttons[action] = key
	notice = _label("")
	column.add_child(notice)
	var footer := HBoxContainer.new()
	column.add_child(footer)
	var restore := Button.new()
	restore.text = "Restore keys"
	restore.pressed.connect(func() -> void:
		bindings = DEFAULTS.duplicate()
		_store_bindings())
	footer.add_child(restore)
	var done := Button.new()
	done.text = "Done"
	done.pressed.connect(settings_window.hide)
	footer.add_child(done)


func _label(caption: String) -> Label:
	var label := Label.new()
	label.text = caption
	label.add_theme_color_override("font_color", Color("392617"))
	return label


func _refresh_keys() -> void:
	for action: String in buttons:
		buttons[action].text = OS.get_keycode_string(bindings[action])
	notice.text = ""


func _listen_for_key(action: String) -> void:
	_refresh_keys()
	pending_action = action
	buttons[action].text = "Press a key"
	notice.text = "Esc cancels."


func _store_bindings() -> void:
	for action: String in bindings:
		preferences.set_value("camera", action, bindings[action])
	_save_preferences()
	pending_action = ""
	_refresh_keys()


func _settings_input(event: InputEvent) -> void:
	if not event is InputEventKey or not event.pressed or event.echo: return
	if event.keycode == KEY_ESCAPE:
		if pending_action.is_empty(): settings_window.hide()
		pending_action = ""
		_refresh_keys()
		settings_window.set_input_as_handled()
		return
	if pending_action.is_empty(): return
	settings_window.set_input_as_handled()
	if event.keycode in RESERVED or event.is_command_or_control_pressed() or event.alt_pressed or event.shift_pressed:
		notice.text = "Choose an unmodified, unreserved key."
		return
	var existing: Variant = bindings.find_key(event.keycode)
	if existing != null and existing != pending_action:
		notice.text = "Already assigned to " + str(existing).to_lower() + "."
		return
	bindings[pending_action] = event.keycode
	_store_bindings()


func handle_camera_key(game: Node, event: InputEvent) -> bool:
	if not event is InputEventKey or not event.pressed or is_open(): return false
	if event.keycode>=KEY_F1 and event.keycode<=KEY_F3 and not event.echo:
		var slot:=str(event.keycode-KEY_F1+1)
		var section:="map_"+str(game.saves.active_path).sha256_text()+"_"+str(game.state.get("campaign",{}).get("mission_id",-1))
		if event.ctrl_pressed:
			var tile: Vector2=game._tile_at(game.camera.get_screen_center_position())
			preferences.set_value(section,slot,tile)
			_save_preferences()
			game.status.text="Map position %s saved" % slot
		elif preferences.has_section_key(section,slot):
			var tile: Vector2=preferences.get_value(section,slot)
			game.selected={}
			game.camera.position=game._iso(tile.x,tile.y)
			game.camera.force_update_scroll()
		game.get_viewport().set_input_as_handled()
		return true
	if event.is_command_or_control_pressed() or event.alt_pressed: return false
	var action: Variant = bindings.find_key(event.keycode)
	if action == null: return false
	if event.echo: return true
	match action:
		"Show cliffs":
			game.royal_tombs.set_cliffs_visible(not game.royal_tombs.cliffs_visible)
		"Income tax","Malaria risk","Disease risk":
			var overlay: String={"Income tax":"tax","Malaria risk":"malaria","Disease risk":"disease"}[action]
			game._set_overlay("" if game.active_overlay==overlay else overlay)
		"Rotate building":
			if game.active_tool.begins_with("complex_"):
				game.temple_facing=(game.temple_facing+1)%4
				game._update_ghost()
		"Zoom in", "Zoom out":
			var center: Vector2 = game.get_viewport_rect().size / 2
			game._zoom_at(center, 1.15 if action == "Zoom in" else 1.0 / 1.15)
		"Rotate left", "Rotate right":
			game._turn_city(1 if action == "Rotate right" else -1)
		_:
			camera_game = game
			held_pan[action] = event.keycode
	game.get_viewport().set_input_as_handled()
	return true


func _process(delta: float) -> void:
	if held_pan.is_empty(): return
	if not is_instance_valid(camera_game) or not get_window().has_focus() or is_open():
		held_pan.clear()
		return
	for action: String in held_pan.keys():
		if not Input.is_key_pressed(held_pan[action]):
			held_pan.erase(action)
	var axis := Vector2(int(held_pan.has("Pan right")) - int(held_pan.has("Pan left")),
			int(held_pan.has("Pan down")) - int(held_pan.has("Pan up")))
	camera_game.camera.position += axis.normalized() * (640 * delta) / camera_game.camera.zoom
	camera_game._clamp_camera()
	camera_game.camera.force_update_scroll()


func _exit_tree() -> void:
	music.stop()
	river.stop()
	toolsound.stop()
	river.stream=null
	toolsound.stream=null
	music.stream = null
