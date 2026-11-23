extends Control

const SANDBOX := "res://scenes/sandbox.tscn"
const GOLD := Color("#ebca85")
const INK := Color("#301e13")
const Saves = preload("res://scripts/save_browser.gd")
var opening := false
var main_menu: VBoxContainer
var chooser: PanelContainer
var choices: ItemList
var mission_start: Button
var chooser_return: Button


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
	main_menu=menu
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
	var start := _add_choice(menu, "Found a Nile Valley city", "valley")
	var campaign := _add_choice(menu, "Begin the Pharaoh campaign", "campaign")
	_add_choice(menu, "Enter the riverbank village", "village")
	var restore := _add_choice(menu, "Load saved town", "load")
	restore.disabled = not FileAccess.file_exists(Saves.last_path())
	chooser_return=_add_choice(menu, "Choose a mission", "missions")
	_add_choice(menu, "Sound and controls", "settings")
	_add_choice(menu, "Quit", "quit")
	start.grab_focus()
	if "--campaign-smoke" in OS.get_cmdline_user_args():
		await get_tree().process_frame
		var click := InputEventMouseButton.new()
		click.button_index = MOUSE_BUTTON_LEFT
		click.position = campaign.get_global_rect().get_center()
		click.pressed = true
		get_viewport().push_input(click, true)
		click.pressed = false
		get_viewport().push_input(click, true)


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
	if choice=="missions":
		_choose_mission()
		return
	if choice=="back":
		chooser.hide()
		main_menu.show()
		chooser_return.grab_focus()
		return
	if choice=="selected":
		var selected:=choices.get_selected_items()
		if selected.is_empty(): return
		get_tree().root.set_meta("nile_mission_id",int(choices.get_item_metadata(selected[0])))
		choice="campaign"
	elif choice=="campaign":
		get_tree().root.set_meta("nile_mission_id",0)
	opening = true
	if choice == "settings":
		opening = false
		CitySettings.open_settings()
		return
	if choice == "quit":
		get_tree().quit()
		return
	get_tree().root.set_meta("nile_save_path", Saves.last_path() if choice=="load" else Saves.new_path("Pharaoh campaign" if choice=="campaign" else "Nile city"))
	get_tree().root.set_meta("nile_start_choice", choice)
	get_tree().change_scene_to_file.call_deferred(SANDBOX)

func _choose_mission() -> void:
	if not is_instance_valid(chooser):
		chooser=PanelContainer.new()
		add_child(chooser)
		chooser.set_anchors_and_offsets_preset(Control.PRESET_CENTER)
		chooser.offset_left=-250
		chooser.offset_right=250
		chooser.offset_top=-222
		chooser.offset_bottom=222
		var stone:=StyleBoxFlat.new()
		stone.bg_color=Color("cda46b")
		stone.border_color=INK
		stone.set_border_width_all(4)
		stone.content_margin_left=20
		stone.content_margin_right=20
		stone.content_margin_top=14
		stone.content_margin_bottom=14
		chooser.add_theme_stylebox_override("panel",stone)
		var column:=VBoxContainer.new()
		chooser.add_child(column)
		var heading:=Label.new()
		heading.text="Choose a mission"
		heading.add_theme_color_override("font_color",INK)
		heading.add_theme_font_size_override("font_size",26)
		column.add_child(heading)
		choices=ItemList.new()
		choices.add_theme_font_size_override("font_size",19)
		choices.size_flags_vertical=Control.SIZE_EXPAND_FILL
		column.add_child(choices)
		choices.item_activated.connect(func(_index: int): _choose("selected"))
		var output: Array=[]
		var executable:=preload("res://scripts/sidecar.gd").executable_path()
		if FileAccess.file_exists(executable) and OS.execute(executable,["--missions"],output)==0:
			var catalog=JSON.parse_string("".join(output))
			if catalog is Array:
				for mission: Dictionary in catalog:
					var index:=choices.add_item("%s     %s BC" % [mission.name,absi(int(mission.year))])
					choices.set_item_metadata(index,int(mission.id))
					choices.set_item_tooltip(index,str(mission.period))
		mission_start=_add_choice(column,"Enter settlement","selected")
		mission_start.disabled=choices.item_count==0
		if choices.item_count>0: choices.select(0)
		else: heading.text="The mission list could not be loaded"
		_add_choice(column,"Back","back")
	main_menu.hide()
	chooser.show()
	choices.grab_focus()

func _unhandled_key_input(event: InputEvent) -> void:
	if event.is_action_pressed("ui_cancel") and is_instance_valid(chooser) and chooser.visible:
		_choose("back")
		get_viewport().set_input_as_handled()
