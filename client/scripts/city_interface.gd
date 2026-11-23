extends RefCounted
## City menus and overseers. This class reads state; it never advances simulation.

const MiniMap = preload("res://scripts/city_minimap.gd")
const CATEGORIES := {
	"Homes": ["hut", "well"],
	"Food": ["farm", "granary", "bazaar"],
	"Industry": ["goldmine", "claypit", "brickworks", "storageyard"],
	"Safety": ["firehouse", "architect"],
	"Civic": ["taxoffice", "palace"],
	"Monument": ["mastaba"],
}
const LABELS := {"goldmine": "Gold mine", "palace": "Village palace", "hut": "Housing", "well": "Well", "farm": "Grain farm", "granary": "Granary", "bazaar": "Bazaar", "claypit": "Clay pit", "brickworks": "Brickworks", "storageyard": "Storage yard", "firehouse": "Firehouse", "architect": "Architect", "taxoffice": "Tax office", "mastaba": "Mastaba"}
const OVERLAYS := {"None": "", "Water": "water", "Food": "food", "Fire risk": "fire", "Collapse risk": "collapse", "Tax coverage": "tax", "Employment": "labor"}

var game: Node
var minimap: Control
var categories: Dictionary = {}
var active_category := "Homes"
var overlay_picker: OptionButton
var repair_button: Button
var game_menu: MenuButton
var overseers_button: Button
var new_town_modal: Control
var new_town_empty: Button
var new_town_starter: Button
var overseer_modal: Control
var overseer_text: RichTextLabel
var overseer_topic := "City"
var overseer_tabs: Dictionary = {}
var completion_modal: Control
var completion_text: Label
var completion_continue: Button
var construction_menu: PanelContainer
var price_labels: Dictionary = {}
var policy_panel: VBoxContainer
var tax_rate: SpinBox
var wage_rate: SpinBox
var labor_priority: OptionButton
var policy_apply: Button
var policy_notice: Label
var policy_dirty := false
var help_modal: Control
var inspector_modal: Control
var build_info: Label
var options_menu: MenuButton


func setup(owner: Node) -> void:
	game = owner
	var layer := CanvasLayer.new()
	layer.name = "CityInterface"
	game.add_child(layer)
	game.hud = Control.new()
	game.hud.set_anchors_and_offsets_preset(Control.PRESET_FULL_RECT)
	game.hud.mouse_filter = Control.MOUSE_FILTER_IGNORE
	layer.add_child(game.hud)
	var ribbon := PanelContainer.new()
	ribbon.set_anchors_and_offsets_preset(Control.PRESET_TOP_WIDE)
	ribbon.offset_bottom = 44
	ribbon.add_theme_stylebox_override("panel", game._style(Color("c48d55"), Color("80502e"), 2))
	game.hud.add_child(ribbon)
	var typeface := SystemFont.new()
	typeface.font_names = PackedStringArray(["Palatino Linotype", "Palatino", "Georgia", "DejaVu Serif"])
	typeface.font_weight = 600
	var row := HBoxContainer.new()
	row.add_theme_constant_override("separation", 10)
	ribbon.add_child(row)
	game_menu = MenuButton.new()
	game_menu.text = "File"
	game_menu.flat = true
	game_menu.add_theme_color_override("font_color", Color("392617"))
	row.add_child(game_menu)
	var menu := game_menu.get_popup()
	menu.add_item("New town…", 0)
	menu.add_separator()
	menu.add_item("Save town", 1)
	menu.add_item("Load town", 2)
	menu.add_separator()
	menu.add_item("Quit", 3)
	menu.id_pressed.connect(_game_action)
	overseers_button = game._button("Overseers", show_overseers)
	overseers_button.custom_minimum_size.y = 26
	overseers_button.flat = true
	options_menu = MenuButton.new()
	options_menu.text = "Options"
	options_menu.add_theme_color_override("font_color", Color("392617"))
	options_menu.get_popup().add_item("Sound and controls…", 0)
	options_menu.get_popup().id_pressed.connect(func(_id: int) -> void:
		game._cancel_gesture()
		var settings := game.get_node_or_null("/root/CitySettings")
		if settings != null: settings.open_settings())
	row.add_child(options_menu)
	var help: Button = game._button("Help", func() -> void: game._cancel_gesture(); help_modal.show())
	help.flat = true
	row.add_child(help)
	row.add_child(overseers_button)
	for button: Button in [game_menu, options_menu, help, overseers_button]:
		button.add_theme_font_override("font", typeface)
		button.add_theme_font_size_override("font_size", 18)
		button.add_theme_color_override("font_color", Color("392617"))
	game.stats = game._label("Starting local settlement…", 13, Color("392617"))
	game.stats.size_flags_horizontal = Control.SIZE_EXPAND_FILL
	game.stats.horizontal_alignment = HORIZONTAL_ALIGNMENT_CENTER
	game.stats.add_theme_font_override("font", typeface)
	game.stats.add_theme_font_size_override("font_size", 17)
	row.add_child(game.stats)
	var trim := preload("res://scripts/city_frieze.gd").new()
	game.hud.add_child(trim)
	trim.set_anchors_and_offsets_preset(Control.PRESET_TOP_WIDE)
	trim.offset_top = 39
	trim.offset_bottom = 44
	var paper := TextureRect.new()
	paper.texture = load("res://assets/street/sidebar.png")
	paper.expand_mode = TextureRect.EXPAND_IGNORE_SIZE
	paper.stretch_mode = TextureRect.STRETCH_SCALE
	paper.mouse_filter = Control.MOUSE_FILTER_IGNORE
	paper.set_anchors_and_offsets_preset(Control.PRESET_RIGHT_WIDE)
	paper.offset_left = -210
	paper.offset_top = 44
	game.hud.add_child(paper)
	var panel := PanelContainer.new()
	panel.set_anchors_and_offsets_preset(Control.PRESET_RIGHT_WIDE)
	panel.offset_left = -210
	panel.offset_top = 44
	var style: StyleBoxFlat = game._style(Color.TRANSPARENT, Color.TRANSPARENT, 0)
	style.content_margin_left = 12
	style.content_margin_right = 12
	style.content_margin_top = 10
	style.content_margin_bottom = 32
	panel.add_theme_stylebox_override("panel", style)
	game.hud.add_child(panel)
	var column := VBoxContainer.new()
	column.add_theme_constant_override("separation", 4)
	panel.add_child(column)
	minimap = MiniMap.new()
	minimap.navigate.connect(game._navigate_minimap)
	column.add_child(minimap)
	overlay_picker = OptionButton.new()
	overlay_picker.add_theme_font_size_override("font_size", 12)
	overlay_picker.clip_text = true
	overlay_picker.fit_to_longest_item = false
	for label: String in OVERLAYS: overlay_picker.add_item("Overlay: " + label)
	overlay_picker.item_selected.connect(func(index: int) -> void:
		game._set_overlay(OVERLAYS.values()[index])
		overlay_picker.release_focus())
	column.add_child(overlay_picker)
	var category_grid := GridContainer.new()
	category_grid.columns = 3
	column.add_child(category_grid)
	for name: String in CATEGORIES:
		var button: Button = game._button(name, set_category.bind(name))
		button.toggle_mode = true
		button.custom_minimum_size = Vector2(56, 25)
		button.add_theme_font_size_override("font_size", 11)
		var icon_id := "category_" + name.to_lower()
		if game.assets.has(icon_id):
			button.text = ""
			button.icon = game._texture_for(icon_id)
			button.expand_icon = true
			button.icon_alignment = HORIZONTAL_ALIGNMENT_CENTER
			button.custom_minimum_size.y = 42
		button.tooltip_text = name
		button.accessibility_name = name
		for state_name in ["normal", "hover", "pressed"]:
			var box: StyleBox = button.get_theme_stylebox(state_name).duplicate()
			box.content_margin_left = 3
			box.content_margin_right = 3
			box.content_margin_top = 3
			box.content_margin_bottom = 3
			button.add_theme_stylebox_override(state_name, box)
		categories[name] = button
		category_grid.add_child(button)
	var commands := HBoxContainer.new()
	column.add_child(commands)
	for entry: Array in [["inspect", "Inspect"], ["road", "Road"], ["clear", "Clear"]]:
		var button: Button = game._button(entry[1], game._set_tool.bind(entry[0]))
		button.toggle_mode = true
		button.custom_minimum_size = Vector2(56, 27)
		button.add_theme_font_size_override("font_size", 11)
		game.tool_buttons[entry[0]] = button
		commands.add_child(button)
	construction_menu = PanelContainer.new()
	construction_menu.set_anchors_and_offsets_preset(Control.PRESET_TOP_RIGHT)
	construction_menu.offset_left = -530
	construction_menu.offset_right = -214
	construction_menu.offset_top = 180
	construction_menu.add_theme_stylebox_override("panel", game._style(Color("cba572"), Color("785634"), 2))
	game.hud.add_child(construction_menu)
	game.palette.columns = 1
	construction_menu.add_child(game.palette)
	for category: String in CATEGORIES:
		for kind: String in CATEGORIES[category]:
			var button: Button = game._button("", _choose_building.bind(kind))
			button.toggle_mode = true
			button.custom_minimum_size = Vector2(290, 38)
			button.tooltip_text = LABELS[kind]
			button.accessibility_name = LABELS[kind]
			var margin := MarginContainer.new()
			margin.set_anchors_and_offsets_preset(Control.PRESET_FULL_RECT)
			margin.add_theme_constant_override("margin_left", 10)
			margin.add_theme_constant_override("margin_right", 10)
			margin.mouse_filter = Control.MOUSE_FILTER_IGNORE
			button.add_child(margin)
			var entry := HBoxContainer.new()
			entry.mouse_filter = Control.MOUSE_FILTER_IGNORE
			margin.add_child(entry)
			if game.assets.has(kind) and ResourceLoader.exists(game._asset_path(game.assets[kind].path)):
				var icon := TextureRect.new()
				icon.texture = game._texture_for(kind)
				icon.custom_minimum_size = Vector2(28, 28)
				icon.expand_mode = TextureRect.EXPAND_IGNORE_SIZE
				icon.stretch_mode = TextureRect.STRETCH_KEEP_ASPECT_CENTERED
				icon.mouse_filter = Control.MOUSE_FILTER_IGNORE
				entry.add_child(icon)
			var label: Label = game._label(LABELS[kind], 14, Color("392617"))
			label.size_flags_horizontal = Control.SIZE_EXPAND_FILL
			entry.add_child(label)
			var price: Label = game._label("", 14, Color("392617"))
			price.horizontal_alignment = HORIZONTAL_ALIGNMENT_RIGHT
			price_labels[kind] = price
			entry.add_child(price)
			game.tool_buttons[kind] = button
			game.palette.add_child(button)
	set_category("Homes", false)
	game.tool_buttons.inspect.set_pressed_no_signal(true)
	build_info = game._label("", 13, Color("35281d"))
	build_info.autowrap_mode = TextServer.AUTOWRAP_WORD_SMART
	column.add_child(build_info)
	var spacer := Control.new()
	spacer.size_flags_vertical = Control.SIZE_EXPAND_FILL
	column.add_child(spacer)
	game.status = game._label("", 11, Color("735936"))
	game.status.autowrap_mode = TextServer.AUTOWRAP_WORD_SMART
	game.status.max_lines_visible = 3
	game.status.text_overrun_behavior = TextServer.OVERRUN_TRIM_ELLIPSIS
	column.add_child(game.status)
	var time_row := HBoxContainer.new()
	column.add_child(time_row)
	game.pause_button = game._button("Resume", game._toggle_pause)
	game.pause_button.size_flags_horizontal = Control.SIZE_EXPAND_FILL
	time_row.add_child(game.pause_button)
	game.speed_button = game._button("1×", game._cycle_speed)
	game.speed_button.custom_minimum_size.x = 56
	time_row.add_child(game.speed_button)
	var saves := HBoxContainer.new()
	column.add_child(saves)
	game.save_button = game._button("Save", func() -> void: game._queue_command("save"))
	game.load_button = game._button("Load", game._load_save)
	game.save_button.size_flags_horizontal = Control.SIZE_EXPAND_FILL
	game.load_button.size_flags_horizontal = Control.SIZE_EXPAND_FILL
	saves.add_child(game.save_button)
	saves.add_child(game.load_button)
	var views := HBoxContainer.new()
	column.add_child(views)
	game.neighborhood_button = game._button("Center", func() -> void: game._set_view(game.neighborhood_zoom))
	game.neighborhood_button.size_flags_horizontal = Control.SIZE_EXPAND_FILL
	views.add_child(game.neighborhood_button)
	game.footprints_button = game._button("Plots", game._toggle_footprints)
	game.footprints_button.toggle_mode = true
	game.footprints_button.size_flags_horizontal = Control.SIZE_EXPAND_FILL
	views.add_child(game.footprints_button)
	game.tool_buttons.inspect.tooltip_text = "Right-click a building or walker to inspect. Right-drag to pan."
	game.tool_buttons.road.tooltip_text = "Drag to preview roads. Release to build; right-click or Esc cancels."
	game.pause_button.tooltip_text = "Space: pause or resume"
	_build_modals()


func set_category(name: String, show_menu: bool = true) -> void:
	active_category = name
	construction_menu.visible = show_menu
	for key: String in categories: categories[key].set_pressed_no_signal(key == name)
	for category: String in CATEGORIES:
		for kind: String in CATEGORIES[category]: game.tool_buttons[kind].visible = category == name


func _choose_building(kind: String) -> void:
	construction_menu.hide()
	game._set_tool(kind)


func category_for(kind: String) -> String:
	for category: String in CATEGORIES:
		if kind in CATEGORIES[category]: return category
	return ""


func _game_action(id: int) -> void:
	game_menu.release_focus()
	game._cancel_gesture()
	match id:
		0: new_town_modal.show()
		1: game._queue_command("save")
		2: game._load_save()
		3:
			game.sim.stop()
			game.get_tree().quit()


func _modal(title: String, dimensions: Vector2) -> Array:
	var background := ColorRect.new()
	background.color = Color(0.08, 0.06, 0.03, 0.72)
	background.set_anchors_and_offsets_preset(Control.PRESET_FULL_RECT)
	game.hud.add_child(background)
	var panel := PanelContainer.new()
	panel.set_anchors_and_offsets_preset(Control.PRESET_CENTER)
	panel.offset_left = -dimensions.x * 0.5
	panel.offset_right = dimensions.x * 0.5
	panel.offset_top = -dimensions.y * 0.5
	panel.offset_bottom = dimensions.y * 0.5
	panel.add_theme_stylebox_override("panel", game._style(Color("dfcb9f"), Color("896b3e"), 3))
	background.add_child(panel)
	var column := VBoxContainer.new()
	column.add_theme_constant_override("separation", 12)
	panel.add_child(column)
	var heading := HBoxContainer.new()
	column.add_child(heading)
	var label: Label = game._label(title, 23, Color("35281d"))
	label.size_flags_horizontal = Control.SIZE_EXPAND_FILL
	heading.add_child(label)
	heading.add_child(game._button("Close", background.hide))
	background.hide()
	return [background, column]


func _build_modals() -> void:
	var help := _modal("Building on the Nile", Vector2(540, 325))
	help_modal = help[0]
	var guide: Label = game._label("Branch a road from the Kingdom Road. Drag to mark house plots within two tiles of a connected street. A nearby well lets the first homes grow.\n\nPlace farms on the dark floodplain. After the harvest, carts take grain to a granary and bazaar sellers bring it to homes.\n\nRight-click a building or a walker to inspect it. Open Options to change camera keys and music volume.", 16, Color("392617"))
	guide.autowrap_mode = TextServer.AUTOWRAP_WORD_SMART
	help[1].add_child(guide)
	var inspection := _modal("Inspection", Vector2(540, 520))
	inspector_modal = inspection[0]
	game.title_label = game._label("", 21, Color("35281d"))
	game.title_label.autowrap_mode = TextServer.AUTOWRAP_WORD_SMART
	inspection[1].add_child(game.title_label)
	game.kind_label = game._label("", 15, Color("735936"))
	game.kind_label.autowrap_mode = TextServer.AUTOWRAP_WORD_SMART
	inspection[1].add_child(game.kind_label)
	var scroll := ScrollContainer.new()
	scroll.size_flags_vertical = Control.SIZE_EXPAND_FILL
	scroll.horizontal_scroll_mode = ScrollContainer.SCROLL_MODE_DISABLED
	inspection[1].add_child(scroll)
	game.detail_label = game._label("", 15, Color("35281d"))
	game.detail_label.size_flags_horizontal = Control.SIZE_EXPAND_FILL
	game.detail_label.autowrap_mode = TextServer.AUTOWRAP_WORD_SMART
	scroll.add_child(game.detail_label)
	game.production_bar = ProgressBar.new()
	game.production_bar.custom_minimum_size.y = 12
	game.production_bar.show_percentage = false
	game.production_bar.visible = false
	inspection[1].add_child(game.production_bar)
	repair_button = game._button("Repair", game._repair_selected)
	repair_button.visible = false
	inspection[1].add_child(repair_button)
	var town := _modal("Found a new town", Vector2(540, 240))
	new_town_modal = town[0]
	var text: Label = game._label("Choose an empty riverbank or an established neighborhood. Save your current town first if you want to keep its latest changes.", 15, Color("35281d"))
	text.autowrap_mode = TextServer.AUTOWRAP_WORD_SMART
	town[1].add_child(text)
	new_town_empty = game._button("Empty riverbank", func() -> void: new_town_modal.hide(); game._new_town("empty"))
	new_town_starter = game._button("Growing neighborhood", func() -> void: new_town_modal.hide(); game._new_town("starter"))
	town[1].add_child(new_town_empty)
	town[1].add_child(new_town_starter)
	var overseer := _modal("City overseers", Vector2(680, 500))
	overseer_modal = overseer[0]
	var tabs := HBoxContainer.new()
	overseer[1].add_child(tabs)
	for topic in ["City", "Food", "Labor", "Treasury", "Safety", "Monument"]:
		var button: Button = game._button(topic, _choose_overseer.bind(topic))
		button.add_theme_font_size_override("font_size", 12)
		overseer_tabs[topic] = button
		tabs.add_child(button)
	policy_panel = VBoxContainer.new()
	policy_panel.add_theme_constant_override("separation", 6)
	overseer[1].add_child(policy_panel)
	var rates := HBoxContainer.new()
	policy_panel.add_child(rates)
	rates.add_child(game._label("Taxes", 14, Color("35281d")))
	tax_rate = SpinBox.new()
	tax_rate.min_value = 0
	tax_rate.max_value = 10
	tax_rate.step = 1
	tax_rate.suffix = "Db / resident"
	tax_rate.custom_minimum_size.x = 185
	tax_rate.value_changed.connect(func(_value: float) -> void: policy_dirty = true)
	rates.add_child(tax_rate)
	rates.add_child(game._label("Wages", 14, Color("35281d")))
	wage_rate = SpinBox.new()
	wage_rate.min_value = 0
	wage_rate.max_value = 5
	wage_rate.step = 1
	wage_rate.suffix = "Db / worker"
	wage_rate.custom_minimum_size.x = 185
	wage_rate.value_changed.connect(func(_value: float) -> void: policy_dirty = true)
	rates.add_child(wage_rate)
	var allocation := HBoxContainer.new()
	policy_panel.add_child(allocation)
	allocation.add_child(game._label("Labor priority", 14, Color("35281d")))
	labor_priority = OptionButton.new()
	for priority in ["Balanced", "Food", "Construction", "Services"]: labor_priority.add_item(priority)
	labor_priority.item_selected.connect(func(_index: int) -> void: policy_dirty = true)
	labor_priority.size_flags_horizontal = Control.SIZE_EXPAND_FILL
	allocation.add_child(labor_priority)
	policy_apply = game._button("Apply policy", _apply_policy)
	allocation.add_child(policy_apply)
	policy_notice = game._label("Rates apply at each tax and wage payment. Zero wages stop work and immigration.", 12, Color("72552f"))
	policy_notice.autowrap_mode = TextServer.AUTOWRAP_WORD_SMART
	policy_panel.add_child(policy_notice)
	overseer_text = RichTextLabel.new()
	overseer_text.bbcode_enabled = true
	overseer_text.selection_enabled = true
	overseer_text.size_flags_vertical = Control.SIZE_EXPAND_FILL
	overseer_text.add_theme_color_override("default_color", Color("35281d"))
	overseer_text.add_theme_font_size_override("normal_font_size", 16)
	overseer[1].add_child(overseer_text)
	var completion := _modal("A monument for your city", Vector2(560, 250))
	completion_modal = completion[0]
	completion_text = game._label("", 17, Color("35281d"))
	completion_text.autowrap_mode = TextServer.AUTOWRAP_WORD_SMART
	completion[1].add_child(completion_text)
	completion_continue = game._button("Continue building", completion_modal.hide)
	completion[1].add_child(completion_continue)


func show_overseers() -> void:
	game._cancel_gesture()
	_sync_policy()
	_update_overseer()
	overseer_modal.show()


func _choose_overseer(topic: String) -> void:
	overseer_topic = topic
	if topic == "Treasury" and not policy_dirty: _sync_policy()
	_update_overseer()


func _sync_policy() -> void:
	tax_rate.set_value_no_signal(int(game.state.get("tax_rate", 2)))
	wage_rate.set_value_no_signal(int(game.state.get("wage_rate", 1)))
	var index: int = ["balanced", "food", "construction", "services"].find(str(game.state.get("labor_priority", "balanced")))
	labor_priority.select(maxi(index, 0))
	policy_dirty = false


func _apply_policy() -> void:
	if game.sim.child.is_empty(): return
	policy_apply.disabled = true
	game._queue_command("set_policy", {"tax_rate": int(tax_rate.value), "wage_rate": int(wage_rate.value), "labor_priority": ["balanced", "food", "construction", "services"][labor_priority.selected]})


func policy_acknowledged(reply: Dictionary) -> void:
	policy_apply.disabled = false
	if reply.get("ok", false):
		_sync_policy()
		policy_notice.text = "City policy updated."
	else:
		policy_notice.text = str(reply.get("error", "Policy could not be applied."))


func dismiss_modal() -> bool:
	if construction_menu.visible:
		construction_menu.hide()
		return true
	for modal: Control in [new_town_modal, overseer_modal, completion_modal, inspector_modal, help_modal]:
		if modal.visible:
			modal.hide()
			return true
	return false


func has_open_modal() -> bool:
	for modal: Control in [new_town_modal, overseer_modal, completion_modal, inspector_modal, help_modal]:
		if modal.visible: return true
	return false


func show_inspector() -> void:
	if game.selected.is_empty(): return
	game._cancel_gesture()
	construction_menu.hide()
	game._refresh_inspector()
	inspector_modal.show()


func update_city() -> void:
	game.status.visible = not game.status.text.is_empty()
	build_info.visible = game.active_tool != "inspect"
	if build_info.visible:
		var entry: Dictionary = game._catalog_entry(game.active_tool)
		build_info.text = "%s · %d Db" % [str(entry.get("label", game.active_tool.capitalize())), int(entry.get("cost", 0))]
	if game.selected.is_empty(): inspector_modal.hide()
	if inspector_modal.visible:
		var inspection_panel: Control = inspector_modal.get_child(0)
		var height := clampf(170.0 + game.detail_label.get_minimum_size().y + (35 if repair_button.visible else 0), 270, 520)
		inspection_panel.offset_top = -height * 0.5
		inspection_panel.offset_bottom = height * 0.5
	var visible_area: Rect2 = game.get_viewport_rect()
	visible_area.size.x -= 210
	visible_area.position.y = 44
	visible_area.size.y -= 44
	var inverse: Transform2D = game.get_canvas_transform().affine_inverse()
	var corners := PackedVector2Array()
	for point: Vector2 in [visible_area.position, Vector2(visible_area.end.x, visible_area.position.y), visible_area.end, Vector2(visible_area.position.x, visible_area.end.y)]:
		corners.append(game._tile_at(inverse * point))
	minimap.update_city(game.state, corners)
	for entry: Dictionary in game.state.get("building_catalog", []):
		if price_labels.has(str(entry.kind)):
			price_labels[str(entry.kind)].text = "%d Db" % int(entry.get("cost", 0))
		if game.tool_buttons.has(str(entry.kind)):
			game.tool_buttons[str(entry.kind)].tooltip_text = "%s · %d debens\n%d × %d tiles" % [str(entry.get("label", entry.kind)), int(entry.get("cost", 0)), int(entry.get("width", 1)), int(entry.get("height", 1))]
	if overseer_modal.visible: _update_overseer()


func show_completion() -> void:
	completion_text.text = "Your mastaba is complete. Farmers, traders, craftspeople and builders have supplied a lasting monument. Continue growing your city or found a new town."
	completion_modal.show()


func _update_overseer() -> void:
	var s: Dictionary = game.state
	policy_panel.visible = overseer_topic == "Treasury"
	var lines: Array[String] = ["[b]" + overseer_topic + "[/b]", ""]
	match overseer_topic:
		"City":
			lines.append("Population: %d\nTreasury: %d debens\nGrain: %d\nYear %d · %s" % [int(s.get("population", 0)), int(s.get("treasury", 0)), int(s.get("grain", 0)), int(s.get("year", 1)), str(s.get("season", "Growing"))])
			lines.append("\nFood in town: %d grain" % int(s.get("grain", 0)))
		"Food":
			lines.append("Grain available: %d\nHarvested: %d\nConsumed: %d" % [int(s.get("grain", 0)), int(s.get("grain_produced", 0)), int(s.get("grain_consumed", 0))])
			for b: Dictionary in s.get("buildings", []):
				if b.kind in ["farm", "granary", "bazaar"]:
					lines.append("%s at (%d, %d): %d grain" % [str(b.kind).capitalize(), int(b.x), int(b.y), int(b.get("stock", 0))])
		"Labor":
			lines.append("Available workers: %d\nEmployed: %d\nUnassigned: %d" % [int(s.get("workforce", 0)), int(s.get("employed", 0)), maxi(0, int(s.get("workforce", 0)) - int(s.get("employed", 0)))])
			for b: Dictionary in s.get("buildings", []):
				if int(b.get("workers_needed", 0)) > 0:
					lines.append("%s: %d / %d workers" % [LABELS.get(str(b.kind), str(b.kind).capitalize()), int(b.get("workers", 0)), int(b.get("workers_needed", 0))])
		"Treasury":
			lines.append("Treasury: %d debens\nIncome: %d\nExpenses: %d" % [int(s.get("treasury", 0)), int(s.get("income", 0)), int(s.get("expenses", 0))])
			lines.append("Gold received: %d Db\nGold in transit: %d Db" % [int(s.get("gold_delivered", 0)), int(s.get("gold_in_transit", 0))])
			if not s.get("taxes_enabled", false): lines.append("A staffed palace is needed to collect taxes.")
			lines.append("Last collection: %d Db\nLast wages: %d Db" % [int(s.get("last_income", 0)), int(s.get("last_wages", 0))])
			lines.append("\nTax collectors must reach households. Taxes above 5 Db slow new arrivals. Zero wages halt staffing and immigration. Labor priority directs scarce workers first.")
		"Safety":
			lines.append("Fires: %d\nCollapses: %d" % [int(s.get("fire_count", 0)), int(s.get("collapse_count", 0))])
			for b: Dictionary in s.get("buildings", []):
				if str(b.get("damaged", "")) != "" or int(b.get("fire_risk", 0)) >= 50 or int(b.get("collapse_risk", 0)) >= 50:
					lines.append("%s (%d, %d): fire %d · collapse %d%s" % [LABELS.get(str(b.kind), str(b.kind).capitalize()), int(b.x), int(b.y), int(b.get("fire_risk", 0)), int(b.get("collapse_risk", 0)), " · needs repair" if str(b.get("damaged", "")) != "" else ""])
			lines.append("\nFirehouse and architect patrols protect the roads they can reach. Inspect damaged buildings to repair them.")
		"Monument":
			lines.append("Mastaba progress: %d / %d" % [int(s.get("monument_progress", 0)), int(s.get("monument_goal", 64))])
			for b: Dictionary in s.get("buildings", []):
				if b.kind in ["claypit", "brickworks", "storageyard", "mastaba"]:
					lines.append("%s: clay %d · straw %d · bricks %d" % [LABELS.get(str(b.kind), str(b.kind).capitalize()), int(b.get("clay", 0)), int(b.get("straw", 0)), int(b.get("bricks", 0))])
				if b.kind == "mastaba":
					lines.append("Crew: %d on site · %d en route · %d returning" % [int(b.get("crew_on_site", 0)), int(b.get("crew_en_route", 0)), int(b.get("crew_returning", 0))])
					if not b.get("staging_access", false): lines.append("Workers cannot reach the staging area. Check the road access at the monument.")
			lines.append("\nFarm straw and clay-pit output feed brickworks. Porters carry bricks to the building site; employed builders complete each stage.")
	overseer_text.text = "\n".join(lines)
