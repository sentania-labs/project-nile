extends RefCounted
## City menus and overseers. This class reads state; it never advances simulation.

const ServiceOverlays := preload("res://scripts/service_overlays.gd")
const MiniMap = preload("res://scripts/city_minimap.gd")
const GROUPS := ["Homes", "Food", "Industry", "Safety", "Civic", "Monument", "Religion", "Streets", "Health", "Education", "Entertainment", "Military"]
var groups: Dictionary = {}
var labels: Dictionary = {}
var visible_catalog: Array = []
var build_scroll: ScrollContainer
const OVERLAYS := {"None": "", "Water": "water", "Food": "food", "Fire risk": "fire", "Collapse risk": "collapse", "Tax coverage": "tax", "Disease risk":"disease","Malaria risk":"malaria", "Employment": "labor"}

var burial_controls := preload("res://scripts/burial_controls.gd").new()
var naval_controls := preload("res://scripts/naval_controls.gd").new()
var military_controls := preload("res://scripts/military_controls.gd").new()
var overseer_controls := preload("res://scripts/overseer_controls.gd").new()
var storage_controls := preload("res://scripts/storage_controls.gd").new()
var roadblock := preload("res://scripts/roadblock_controls.gd").new()
var campaign_ui := preload("res://scripts/campaign_interface.gd").new()
var game: Node
var minimap: Control
var categories: Dictionary = {}
var active_category := "Homes"
var overlay_picker: OptionButton
var undo_button: Button
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
var gift_row := HBoxContainer.new()
var gift_buttons: Array[Button]=[]
var governor := HBoxContainer.new()
var salary := OptionButton.new()
var donation := SpinBox.new()
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
	ribbon.add_theme_stylebox_override("panel", _painted_stone(Color("d19870")))
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
	menu.add_item("Save civilization as…", 4)
	menu.add_item("Open civilization…", 5)
	menu.add_item("Restore last manual save", 6)
	menu.add_item("Restore previous manual save", 7)
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
	paper.self_modulate = Color("dbab80")
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
	overlay_picker.alignment = HORIZONTAL_ALIGNMENT_CENTER
	overlay_picker.tooltip_text = "Show service coverage and building risks"
	for state_name: String in ["normal", "hover", "pressed", "focus"]:
		overlay_picker.add_theme_stylebox_override(state_name, game._style(Color("d1a16b") if state_name == "normal" else Color("dfb67e"), Color("785634"), 2))
	for color_name: String in ["font_color", "font_hover_color", "font_pressed_color", "font_hover_pressed_color", "font_focus_color", "icon_normal_color", "icon_hover_color", "icon_pressed_color", "arrow_color"]:
		overlay_picker.add_theme_color_override(color_name, Color("392617"))
	var overlay_menu := overlay_picker.get_popup()
	overlay_menu.add_theme_stylebox_override("panel", game._style(Color("e1c28d"), Color("785634"), 2))
	overlay_menu.add_theme_stylebox_override("hover", game._style(Color("c39259"), Color("9f733f"), 1))
	overlay_menu.add_theme_color_override("font_color", Color("392617"))
	overlay_menu.add_theme_color_override("font_hover_color", Color("25170e"))
	for label: String in OVERLAYS: overlay_picker.add_item("Overlays" if label == "None" else label)
	for group: String in ServiceOverlays.GROUPS:
		var sub := PopupMenu.new()
		sub.name = group
		overlay_menu.add_child(sub)
		var modes: Dictionary = ServiceOverlays.GROUPS[group]
		for label: String in modes:
			sub.add_item(label)
			sub.set_item_metadata(sub.item_count-1,modes[label])
		sub.id_pressed.connect(func(id: int) -> void: game._set_overlay(sub.get_item_metadata(sub.get_item_index(id))))
		overlay_menu.add_submenu_item(group,group)
	overlay_picker.item_selected.connect(func(index: int) -> void:
		if index<OVERLAYS.size(): game._set_overlay(OVERLAYS.values()[index])
		overlay_picker.release_focus())
	column.move_child(minimap,-1)
	column.add_child(overlay_picker)
	column.move_child(overlay_picker,0)
	var mural := TextureButton.new()
	mural.texture_normal = load("res://assets/street/overseers-mural.png")
	mural.stretch_mode=TextureButton.STRETCH_KEEP_ASPECT_COVERED
	mural.ignore_texture_size=true
	mural.custom_minimum_size=Vector2(150,58)
	mural.tooltip_text="City overseers"
	mural.accessibility_name="City overseers"
	mural.pressed.connect(show_overseers)
	column.add_child(mural)
	var category_grid := GridContainer.new()
	category_grid.columns = 4
	column.add_child(category_grid)
	for name: String in GROUPS:
		groups[name] = []
		var button: Button = game._button(name, set_category.bind(name))
		button.toggle_mode = true
		button.custom_minimum_size = Vector2(40, 25)
		button.add_theme_font_size_override("font_size", 11)
		var icon_id := "category_" + name.to_lower()
		if not game.assets.has(icon_id):
			icon_id = {"Religion": "temple_ra", "Streets": "roadblock", "Health": "physician", "Education": "library", "Entertainment": "booth"}.get(name, "")
		if game.assets.has(icon_id):
			button.text = ""
			button.icon = game._texture_for(icon_id)
			button.expand_icon = true
			button.icon_alignment = HORIZONTAL_ALIGNMENT_CENTER
			button.custom_minimum_size.y = 40
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
		button.custom_minimum_size = Vector2(45, 27)
		button.add_theme_font_size_override("font_size", 11)
		game.tool_buttons[entry[0]] = button
		commands.add_child(button)
	undo_button = game._button("↶",func() -> void:
		game._cancel_gesture()
		game._queue_command("undo"))
	undo_button.tooltip_text = "Undo last construction"
	undo_button.custom_minimum_size = Vector2(28,27)
	undo_button.disabled = true
	commands.add_child(undo_button)
	construction_menu = PanelContainer.new()
	construction_menu.set_anchors_and_offsets_preset(Control.PRESET_TOP_RIGHT)
	construction_menu.offset_left = -530
	construction_menu.offset_right = -214
	construction_menu.offset_top = 180
	construction_menu.add_theme_stylebox_override("panel", game._style(Color("cba572"), Color("785634"), 2))
	game.hud.add_child(construction_menu)
	game.palette.columns = 1
	build_scroll = ScrollContainer.new()
	build_scroll.horizontal_scroll_mode = ScrollContainer.SCROLL_MODE_DISABLED
	construction_menu.add_child(build_scroll)
	build_scroll.add_child(game.palette)
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
	campaign_ui.setup(self, row)
	roadblock.setup(self, game)


func _add_choice(kind: String, caption: String) -> void:
	var button: Button = game._button("", _choose_building.bind(kind))
	button.toggle_mode = true
	button.custom_minimum_size = Vector2(290, 38)
	button.tooltip_text = caption
	button.accessibility_name = caption
	var margin := MarginContainer.new()
	margin.set_anchors_and_offsets_preset(Control.PRESET_FULL_RECT)
	margin.add_theme_constant_override("margin_left", 10)
	margin.add_theme_constant_override("margin_right", 10)
	margin.mouse_filter = Control.MOUSE_FILTER_IGNORE
	button.add_child(margin)
	var entry := HBoxContainer.new()
	entry.mouse_filter = Control.MOUSE_FILTER_IGNORE
	margin.add_child(entry)
	var art := "shrine" if kind.begins_with("shrine_") else kind
	if kind.begins_with("complex_"): art="temple_"+kind.trim_prefix("complex_")
	art = {"policestation":"police","residence":"personalresidence","shipbridge":"bridge"}.get(art,art)
	if game.assets.has(art) and ResourceLoader.exists(game._asset_path(game.assets[art].path)):
		var icon := TextureRect.new()
		icon.texture = game._texture_for(art)
		icon.custom_minimum_size = Vector2(28, 28)
		icon.expand_mode = TextureRect.EXPAND_IGNORE_SIZE
		icon.stretch_mode = TextureRect.STRETCH_KEEP_ASPECT_CENTERED
		icon.mouse_filter = Control.MOUSE_FILTER_IGNORE
		entry.add_child(icon)
	var label: Label = game._label(caption, 14, Color("392617"))
	label.size_flags_horizontal = Control.SIZE_EXPAND_FILL
	entry.add_child(label)
	var price: Label = game._label("", 14, Color("392617"))
	price.horizontal_alignment = HORIZONTAL_ALIGNMENT_RIGHT
	price_labels[kind] = price
	entry.add_child(price)
	game.tool_buttons[kind] = button
	game.palette.add_child(button)


func sync_catalog() -> void:
	var entries: Array = game.state.get("building_catalog", [])
	if entries == visible_catalog: return
	visible_catalog = entries
	for group: String in groups: groups[group] = []
	for kind: String in price_labels: game.tool_buttons[kind].hide()
	for entry: Dictionary in entries:
		var kind := str(entry.kind)
		if kind == "road": continue
		var group := str(entry.category)
		match kind:
			"hut", "well", "watersupply": group = "Homes"
			"roadblock", "plaza", "garden","smallstatue","mediumstatue","largestatue", "bridge","shipbridge","ferry","irrigationditch": group = "Streets"
			_:
				group = {"Services":"Safety", "Government":"Civic", "Labor":"Civic", "Monuments":"Monument", "Housing":"Homes", "Beautification":"Streets"}.get(group,group)
		if not groups.has(group): continue
		groups[group].append(kind)
		labels[kind] = str(entry.label)
		if not game.tool_buttons.has(kind): _add_choice(kind, str(entry.label))
		price_labels[kind].text = "%d Db" % int(entry.cost)
		game.tool_buttons[kind].tooltip_text = "%s · %d Db · %d × %d" % [entry.label, int(entry.cost), int(entry.width), int(entry.height)]
	for group: String in groups:
		var unlocked: bool = not groups[group].is_empty()
		categories[group].disabled = not unlocked
		categories[group].modulate = Color.WHITE if unlocked else Color(0.51,0.39,0.28,0.56)
	set_category(active_category, construction_menu.visible and not groups[active_category].is_empty())


func set_category(name: String, show_menu: bool = true) -> void:
	active_category = name
	construction_menu.visible = show_menu
	for key: String in categories: categories[key].set_pressed_no_signal(key == name)
	for kind: String in price_labels: game.tool_buttons[kind].visible = kind in groups[name]
	build_scroll.custom_minimum_size = Vector2(302, mini(groups[name].size() * 42, 370))
	build_scroll.scroll_vertical = 0


func _choose_building(kind: String) -> void:
	construction_menu.hide()
	game._set_tool(kind)


func category_for(kind: String) -> String:
	for category: String in groups:
		if kind in groups[category]: return category
	return ""


func _game_action(id: int) -> void:
	game_menu.release_focus()
	game._cancel_gesture()
	match id:
		0: new_town_modal.show()
		1: game._queue_command("save")
		2: game._load_save()
		4: game.saves.show_dialog(true)
		5: game.saves.show_dialog(false)
		6: game._queue_command("pause"); game._queue_command("load_manual")
		7: game._queue_command("pause"); game._queue_command("load_previous")
		3:
			game.sim.stop()
			game.get_tree().quit()


func _painted_stone(tint: Color) -> StyleBoxTexture:
	var texture: Texture2D = load("res://assets/street/sidebar.png")
	var stone := StyleBoxTexture.new()
	stone.texture = texture
	var pixels := texture.get_size()
	stone.region_rect = Rect2(pixels*Vector2(0.15,0.3),pixels*Vector2(0.7,0.23))
	stone.modulate_color=tint
	stone.axis_stretch_horizontal=StyleBoxTexture.AXIS_STRETCH_MODE_TILE
	stone.axis_stretch_vertical=StyleBoxTexture.AXIS_STRETCH_MODE_TILE
	for side in [SIDE_LEFT,SIDE_RIGHT]: stone.set_content_margin(side,12)
	for side in [SIDE_TOP,SIDE_BOTTOM]: stone.set_content_margin(side,7)
	return stone


func _modal(title: String, dimensions: Vector2, on_close: Callable = Callable()) -> Array:
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
	panel.add_theme_stylebox_override("panel", _painted_stone(Color("efc69d")))
	background.add_child(panel)
	var column := VBoxContainer.new()
	column.add_theme_constant_override("separation", 12)
	panel.add_child(column)
	var heading := HBoxContainer.new()
	column.add_child(heading)
	var label: Label = game._label(title, 23, Color("35281d"))
	label.size_flags_horizontal = Control.SIZE_EXPAND_FILL
	heading.add_child(label)
	heading.add_child(game._button("Close", on_close if on_close.is_valid() else background.hide))
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
	inspection[1].add_child(storage_controls)
	storage_controls.setup(game)
	var town := _modal("Found a new town", Vector2(540, 240))
	new_town_modal = town[0]
	var text: Label = game._label("Choose an empty riverbank or an established neighborhood. Save your current town first if you want to keep its latest changes.", 15, Color("35281d"))
	text.autowrap_mode = TextServer.AUTOWRAP_WORD_SMART
	town[1].add_child(text)
	new_town_empty = game._button("Empty riverbank", func() -> void: new_town_modal.hide(); game._new_town("empty"))
	new_town_starter = game._button("Growing neighborhood", func() -> void: new_town_modal.hide(); game._new_town("starter"))
	town[1].add_child(game._button("Nile Valley · 96 × 96", func() -> void: new_town_modal.hide(); game._new_town("valley")))
	town[1].add_child(new_town_empty)
	town[1].add_child(new_town_starter)
	var overseer := _modal("City overseers", Vector2(680, 500))
	overseer_modal = overseer[0]
	var tabs := HFlowContainer.new()
	overseer[1].add_child(tabs)
	for topic in ["City", "Food", "Labor", "Treasury", "Safety", "Monument", "Trade", "Religion", "Military", "River"]:
		var button: Button = game._button(topic, _choose_overseer.bind(topic))
		button.add_theme_font_size_override("font_size", 12)
		overseer_tabs[topic] = button
		tabs.add_child(button)
	overseer[1].add_child(burial_controls)
	burial_controls.setup(game)
	overseer[1].add_child(overseer_controls)
	overseer_controls.setup(game)
	overseer[1].add_child(military_controls)
	military_controls.setup(game)
	overseer[1].add_child(naval_controls)
	naval_controls.setup(game)
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
	policy_panel.add_child(governor)
	policy_panel.add_child(gift_row)
	gift_row.add_child(game._label("Gift to Pharaoh",14,Color("35281d")))
	for i in range(3):
		var gift: Button=game._button("",func() -> void: game._queue_command("royal_gift",{"rank":i}))
		gift_row.add_child(gift)
		gift_buttons.append(gift)
	governor.add_child(game._label("Monthly salary",14,Color("35281d")))
	for amount in [0,2,5,8,12,20,30,40,60,80,100]: salary.add_item("%d Db" % amount)
	salary.item_selected.connect(func(rank: int) -> void: game._queue_command("set_salary",{"rank":rank}))
	governor.add_child(salary)
	donation.step = 1
	donation.suffix = "Db"
	governor.add_child(donation)
	governor.add_child(game._button("Donate to city",func() -> void: game._queue_command("donate",{"amount":int(donation.value)})))
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
	var monthly := bool(game.state.get("monthly_finance", false))
	tax_rate.max_value = 25 if monthly else 10
	wage_rate.max_value = 100 if monthly else 5
	tax_rate.suffix = "%" if monthly else "Db / resident"
	wage_rate.suffix = "Db / 10 workers / year" if monthly else "Db / worker"
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
	if campaign_ui.panel.visible:
		campaign_ui.dismiss()
		return true
	if construction_menu.visible:
		construction_menu.hide()
		return true
	for modal: Control in [new_town_modal, overseer_modal, completion_modal, inspector_modal, help_modal, roadblock.panel]:
		if modal != null and modal.visible:
			modal.hide()
			return true
	return false


func has_open_modal() -> bool:
	if campaign_ui.panel != null and campaign_ui.panel.visible: return true
	for modal: Control in [new_town_modal, overseer_modal, completion_modal, inspector_modal, help_modal, roadblock.panel]:
		if modal != null and modal.visible: return true
	return false


func show_inspector() -> void:
	if game.selected.is_empty(): return
	game._cancel_gesture()
	construction_menu.hide()
	game._refresh_inspector()
	inspector_modal.show()


func update_city() -> void:
	undo_button.disabled = not game.last_reply.get("can_undo",false)
	game.status.visible = not game.status.text.is_empty()
	build_info.visible = game.active_tool != "inspect"
	if build_info.visible:
		var entry: Dictionary = game._catalog_entry(game.active_tool)
		build_info.text = "%s · %d Db" % [str(entry.get("label", game.active_tool.capitalize())), int(entry.get("cost", 0))]
	if game.selected.is_empty(): inspector_modal.hide()
	roadblock.sync()
	if inspector_modal.visible:
		storage_controls.bind(game.selected.get("building", {}))
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
	military_controls.sync(overseer_topic if overseer_modal.visible else "")
	naval_controls.sync(overseer_topic if overseer_modal.visible else "")
	if overseer_modal.visible: _update_overseer()


func show_completion() -> void:
	completion_text.text = "Your monument is complete. Farmers, traders, craftspeople and builders have supplied a lasting monument. Continue growing your city or found a new town."
	completion_modal.show()


func _update_overseer() -> void:
	var s: Dictionary = game.state
	burial_controls.sync(overseer_topic)
	overseer_controls.sync(overseer_topic)
	military_controls.sync(overseer_topic)
	naval_controls.sync(overseer_topic)
	policy_panel.visible = overseer_topic == "Treasury"
	governor.visible = game._has_any_kind(["residence","familymansion","dynastymansion"])
	gift_row.visible=governor.visible or int(s.get("savings",0))>0
	for i in range(gift_buttons.size()):
		var cost:=int(s.get("royal_gift_costs",[20,50,100])[i])
		gift_buttons[i].text="%s · %d" % [["Modest","Generous","Lavish"][i],cost]
		gift_buttons[i].disabled=cost>int(s.get("savings",0))
	salary.select(int(s.get("salary_rank",0)))
	donation.max_value = float(s.get("savings",0))
	var lines: Array[String] = ["[b]" + overseer_topic + "[/b]", ""]
	match overseer_topic:
		"Trade":
			lines.append("Prices shown per 100 units, purchase / sale.")
			lines.append("Spent on imports: %d Db     Earned from exports: %d Db" % [int(s.get("import_spent",0)),int(s.get("export_earned",0))])
			for r: Dictionary in s.get("resource_catalog", []):
				var imports := int(s.get("resource_imported",{}).get(r.id,0))
				var exports := int(s.get("resource_exported",{}).get(r.id,0))
				if imports + exports > 0: lines.append("%s · %d imported · %d exported" % [r.label,imports,exports])
		"Religion":
			var moods: Array = s.get("favor", [])
			for i in moods.size(): lines.append("%s · %d favor" % [["Osiris","Ra","Ptah","Seth","Bast"][i],int(moods[i])])
			if s.has("festival"): lines.append("A festival for %s is being prepared." % s.festival.god)
			else: lines.append("Festivals held: %d" % int(s.get("festivals_held",0)))
		"City":
			lines.append("Population: %d\nTreasury: %d debens\nGrain: %d\nYear %d · %s" % [int(s.get("population", 0)), int(s.get("treasury", 0)), int(s.get("grain", 0)), int(s.get("year", 1)), str(s.get("season", "Growing"))])
			lines.append("Culture %d     Prosperity %d     Monuments %d     Kingdom %d" % [int(s.get("culture",0)),int(s.get("prosperity",0)),int(s.get("monument_rating",0)),int(s.get("kingdom_standing",0))])
			if int(s.get("zoo_coverage",0))>0: lines.append("Zoo serves %d residents" % int(s.zoo_coverage))
		"Food":
			lines.append("Grain available: %d\nHarvested: %d\nConsumed: %d" % [int(s.get("grain", 0)), int(s.get("grain_produced", 0)), int(s.get("grain_consumed", 0))])
			for b: Dictionary in s.get("buildings", []):
				if b.kind in ["farm", "granary", "bazaar"]:
					lines.append("%s at (%d, %d): %d grain" % [str(b.kind).capitalize(), int(b.x), int(b.y), int(b.get("stock", 0))])
		"Labor":
			lines.append("Available workers: %d\nEmployed: %d\nUnassigned: %d" % [int(s.get("workforce", 0)), int(s.get("employed", 0)), maxi(0, int(s.get("workforce", 0)) - int(s.get("employed", 0)))])
			for b: Dictionary in s.get("buildings", []):
				if int(b.get("workers_needed", 0)) > 0:
					lines.append("%s: %d / %d workers" % [labels.get(str(b.kind), str(b.kind).capitalize()), int(b.get("workers", 0)), int(b.get("workers_needed", 0))])
		"Treasury":
			lines.append("Treasury: %d debens\nIncome: %d\nExpenses: %d" % [int(s.get("treasury", 0)), int(s.get("income", 0)), int(s.get("expenses", 0))])
			lines.append("Gold received: %d Db\nGold in transit: %d Db" % [int(s.get("gold_delivered", 0)), int(s.get("gold_in_transit", 0))])
			if not s.get("taxes_enabled", false): lines.append("A staffed palace is needed to collect taxes.")
			lines.append("Kingdom rating: %d / 100" % int(s.get("kingdom_standing",50)))
			if governor.visible: lines.append("Personal savings: %d Db" % int(s.get("savings",0)))
			lines.append("Last collection: %d Db\nLast wages: %d Db" % [int(s.get("last_income", 0)), int(s.get("last_wages", 0))])
			lines.append("\nTax collectors must reach households. Taxes above 5 Db slow new arrivals. Zero wages halt staffing and immigration. Labor priority directs scarce workers first.")
		"Safety":
			lines.append("City sentiment %d / 100 · Arrests %d · Losses to theft %d Db" % [int(s.get("sentiment",60)),int(s.get("arrests",0)),int(s.get("stolen",0))])
			lines.append("Fires: %d\nCollapses: %d" % [int(s.get("fire_count", 0)), int(s.get("collapse_count", 0))])
			for b: Dictionary in s.get("buildings", []):
				if str(b.get("damaged", "")) != "" or int(b.get("fire_risk", 0)) >= 50 or int(b.get("collapse_risk", 0)) >= 50:
					lines.append("%s (%d, %d): fire %d · collapse %d%s" % [labels.get(str(b.kind), str(b.kind).capitalize()), int(b.x), int(b.y), int(b.get("fire_risk", 0)), int(b.get("collapse_risk", 0)), " · needs repair" if str(b.get("damaged", "")) != "" else ""])
			lines.append("\nFirehouse and architect patrols protect the roads they can reach. Inspect damaged buildings to repair them.")
		"Monument":
			lines.append("Monument rating: %d / 100" % int(s.get("monument_rating",0)))
			for b: Dictionary in s.get("buildings", []):
				if b.kind in ["claypit", "brickworks", "storageyard", "mastaba"]:
					lines.append("%s: clay %d · straw %d · bricks %d" % [labels.get(str(b.kind), str(b.kind).capitalize()), int(b.get("clay", 0)), int(b.get("straw", 0)), int(b.get("bricks", 0))])
				if int(b.get("construction_required",0))>0:
					lines.append("%s · %s"%[labels.get(str(b.kind),str(b.kind).capitalize()),str(b.get("stage","")).capitalize()])
					lines.append("Crew: %d on site · %d en route · %d returning" % [int(b.get("crew_on_site", 0)), int(b.get("crew_en_route", 0)), int(b.get("crew_returning", 0))])
					if not b.get("staging_access", false): lines.append("Workers cannot reach the staging area. Check the road access at the monument.")
			lines.append("\nMastabas need bricks. Pyramids need core blocks, ramps and guild workers. Cased pyramids also need limestone and polishing. Sphinxes are carved from bedrock. Inspect a site for its current need.")
	overseer_text.text = "\n".join(lines)

func select_overlay(kind: String) -> void:
	var index:=OVERLAYS.values().find(kind)
	if index>=0:
		overlay_picker.select(index)
		overlay_picker.text = "Overlays" if kind.is_empty() else str(OVERLAYS.keys()[index])
		return
	for modes: Dictionary in ServiceOverlays.GROUPS.values():
		var label: Variant=modes.find_key(kind)
		if label!=null:
			overlay_picker.select(-1)
			overlay_picker.text=label
			return
