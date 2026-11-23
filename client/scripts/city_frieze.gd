extends Control

func _ready() -> void:
	mouse_filter = Control.MOUSE_FILTER_IGNORE
	set_anchors_and_offsets_preset(Control.PRESET_BOTTOM_WIDE)
	offset_top = -5


func _draw() -> void:
	var colors := [Color("264f64"), Color("b76c45"), Color("407365"), Color("b7673a")]
	draw_rect(Rect2(Vector2.ZERO, size), Color("503727"))
	for i in range(ceili(size.x / 22)):
		var x := float(i * 22)
		draw_rect(Rect2(x + 2, 1, 13, 3), colors[i % colors.size()])
		draw_rect(Rect2(x + 16, 1, 2, 3), Color("e7c992"))
		draw_rect(Rect2(x + 19, 1, 1, 3), Color("e7c992"))
