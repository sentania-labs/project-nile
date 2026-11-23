extends Control

const SANDBOX := "res://scenes/sandbox.tscn"
const GOLD := Color("#ebca85")
const INK := Color("#301e13")
var opening := false


func _ready() -> void:
	for argument in OS.get_cmdline_user_args():
		if argument in ["--diagnostic", "--smoke", "--study", "--street-smoke", "--gold-smoke", "--city-smoke", "--sandbox-smoke"]:
			get_tree().change_scene_to_file.call_deferred(SANDBOX)
			return
	get_window().title = "Project Nile"
	var painting := TextureRect.new()
	painting.texture = load("res://assets/street/title.png")
	painting.expand_mode = TextureRect.EXPAND_IGNORE_SIZE
	painting.stretch_mode = TextureRect.STRETCH_KEEP_ASPECT_COVERED
	painting.mouse_filter = Control.MOUSE_FILTER_IGNORE
	add_child(painting)
	painting.set_anchors_and_offsets_preset(Control.PRESET_FULL_RECT)

	var menu := VBoxContainer.new()
	menu.add_theme_constant_override("separation", 12)
	add_child(menu)
	menu.anchor_left = 0.085
	menu.anchor_top = 0.22
	menu.anchor_right = 0.085
	menu.anchor_bottom = 0.22
	menu.custom_minimum_size.x = 330

	var title := Label.new()
	title.text = "PROJECT NILE"
	var typeface := SystemFont.new()
	typeface.font_names = PackedStringArray(["Cinzel", "Georgia", "DejaVu Serif"])
	title.add_theme_font_override("font", typeface)
	title.add_theme_font_size_override("font_size", 54)
	title.add_theme_color_override("font_color", GOLD)
	title.add_theme_color_override("font_shadow_color", INK)
	title.add_theme_constant_override("shadow_offset_y", 3)
	menu.add_child(title)
	var spacer := Control.new()
	spacer.custom_minimum_size.y = 18
	menu.add_child(spacer)
	var start := _add_choice(menu, "Found a new town", "empty")
	_add_choice(menu, "Enter the riverbank village", "village")
	var restore := _add_choice(menu, "Load saved town", "load")
	var save_path := OS.get_environment("NILE_SANDBOX_SAVE")
	restore.disabled = not FileAccess.file_exists("user://riverbank.nile" if save_path.is_empty() else save_path)
	_add_choice(menu, "Sound and controls", "settings")
	_add_choice(menu, "Quit", "quit")
	start.grab_focus()


func _add_choice(menu: VBoxContainer, caption: String, choice: String) -> Button:
	var button := Button.new()
	button.text = caption
	button.name = choice
	button.custom_minimum_size.y = 48
	button.add_theme_font_size_override("font_size", 20)
	for style_name in ["normal", "hover", "pressed", "disabled", "focus"]:
		var box := StyleBoxFlat.new()
		box.bg_color = Color("#d6aa71")
		if style_name == "hover": box.bg_color = Color("#f3d393")
		if style_name == "pressed": box.bg_color = Color("#bd894f")
		if style_name == "disabled": box.bg_color = Color("#a18a69")
		box.border_color = INK if style_name != "focus" else GOLD
		box.set_border_width_all(2)
		if style_name == "focus": box.draw_center = false
		button.add_theme_stylebox_override(style_name, box)
	for color_name in ["font_color", "font_hover_color", "font_pressed_color", "font_focus_color"]:
		button.add_theme_color_override(color_name, INK)
	button.add_theme_color_override("font_disabled_color", Color("#554735"))
	button.pressed.connect(_choose.bind(choice))
	menu.add_child(button)
	return button


func _choose(choice: String) -> void:
	if opening:
		return
	opening = true
	if choice == "settings":
		opening = false
		CitySettings.open_settings()
		return
	if choice == "quit":
		get_tree().quit()
		return
	get_tree().root.set_meta("nile_start_choice", choice)
	get_tree().change_scene_to_file.call_deferred(SANDBOX)
