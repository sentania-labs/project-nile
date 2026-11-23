extends VBoxContainer
var game: Node
var resource := OptionButton.new()
var loads := SpinBox.new()
var dispatch := Button.new()
var stock := Label.new()
var keys: Array[String] = []
func setup(owner: Node) -> void:
	game = owner
	var row := HBoxContainer.new()
	add_child(row)
	row.add_child(resource)
	loads.min_value=1
	loads.max_value=1
	loads.suffix="loads"
	row.add_child(loads)
	dispatch.text="Dispatch burial provisions"
	row.add_child(dispatch)
	add_child(stock)
	stock.autowrap_mode=TextServer.AUTOWRAP_WORD_SMART
	resource.item_selected.connect(func(_i: int): sync("Monument"))
	dispatch.pressed.connect(func():
		if not keys.is_empty(): game._queue_command("dispatch_burial",{"resource":keys[resource.selected],"limit":int(loads.value)})
		dispatch.release_focus())
func sync(topic: String) -> void:
	var required: Dictionary=game.state.get("burial_requirements",{})
	visible=topic=="Monument" and not required.is_empty()
	if not visible:return
	var wanted: Array[String]=[]
	for r: String in required: wanted.append(r)
	wanted.sort()
	if keys!=wanted:
		keys=wanted
		resource.clear()
		for r: String in keys:resource.add_item(r.capitalize())
	var r:=keys[maxi(0,resource.selected)]
	var delivered:=int(game.state.get("burial_delivered",{}).get(r,0))
	var available:=int(game.state.get("burial_available",{}).get(r,0))
	var remaining:=int(required[r])-delivered
	var cap:=mini(remaining/100,available/100)
	loads.max_value=maxi(1,cap)
	dispatch.disabled=cap==0 or game.sim.child.is_empty()
	stock.text="%d / %d loads provided · %d loads available in storage" %[delivered/100,int(required[r])/100,available/100]
