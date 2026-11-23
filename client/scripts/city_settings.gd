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
}
const RESERVED := [KEY_ESCAPE, KEY_ENTER, KEY_SPACE, KEY_F, KEY_TAB, KEY_SHIFT, KEY_CTRL, KEY_ALT, KEY_META]

var preferences := ConfigFile.new()
var bindings := DEFAULTS.duplicate()
var music := AudioStreamPlayer.new()
var track := 0
var settings_window: Window
var buttons: Dictionary = {}
var pending_action := ""
var notice: Label
var held_pan: Dictionary = {}
var camera_game: Node


func _ready() -> void:
	preferences.load(SAVE_PATH)
	var used := {}
	for action: String in DEFAULTS:
		var code := int(preferences.get_value("camera", action, DEFAULTS[action]))
		if code <= 0 or code in RESERVED or used.has(code):
			bindings = DEFAULTS.duplicate()
			break
		bindings[action] = code
		used[code] = true
	add_child(music)
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
	settings_window.size = Vector2i(410, 440)
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
	var grid := GridContainer.new()
	grid.columns = 2
	column.add_child(grid)
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
	if event.is_command_or_control_pressed() or event.alt_pressed: return false
	var action: Variant = bindings.find_key(event.keycode)
	if action == null: return false
	if event.echo: return true
	match action:
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
	music.stream = null
