extends RefCounted

const GROUPS := {
	"Entertainment": {"All entertainment":"entertainment", "Jugglers":"jugglers", "Musicians":"musicians", "Dancers":"dancers", "Senet players":"senet"},
	"Education": {"All education":"education", "Scribal schools":"school", "Libraries":"library"},
	"Health": {"Herbalists":"herbalist", "Physicians":"physician", "Dentists":"dentist", "Embalmers":"embalmer"},
	"Administration": {"Crime risk":"crime", "Worship":"worship", "Housing desirability":"desirability"}
}
const SERVICES := {"herbalist":0, "physician":1, "dentist":2, "embalmer":3, "school":4, "library":5, "senet":7}
const ACTS := {"jugglers":0, "musicians":1, "dancers":2}
const PROVIDERS := {
	"water":["well","watersupply"], "food":["granary","bazaar"], "tax":["taxoffice"],
	"fire":["firehouse"], "collapse":["architect"], "disease":["physician"], "malaria":["apothecary"],
	"crime":["policestation","courthouse"], "herbalist":["apothecary"], "physician":["physician"], "dentist":["dentist"], "embalmer":["mortuary"],
	"school":["scribalschool"], "library":["library"], "education":["scribalschool","library"],
	"jugglers":["booth","bandstand","pavilion","jugglerschool"], "musicians":["bandstand","pavilion","conservatory"],
	"dancers":["pavilion","danceschool"], "senet":["senethouse"], "entertainment":["booth","bandstand","pavilion","senethouse","zoo"]
}

static func opacity(kind: String, mode: String) -> float:
	if mode.is_empty() or kind in PROVIDERS.get(mode,[]): return 1.0
	if mode=="worship" and (kind.begins_with("shrine_") or kind.begins_with("temple_") or kind.begins_with("complex_")): return 1.0
	return 0.22 if kind=="hut" else 0.45

static func visit(data: Dictionary, field: String, idx: int, tick: int) -> float:
	var until: Array = data.get(field,[])
	if idx >= until.size(): return 0.0
	return clampf((float(until[idx])-tick)/128.0,0,1)

static func value(data: Dictionary, mode: String, tick: int) -> float:
	match mode:
		"fire","collapse": return clampf(float(data.get(mode+"_risk",0))/100,0,1)
		"labor":
			var jobs:=int(data.get("workers_needed",0))
			return 1.0-float(data.get("workers",0))/jobs if jobs>0 else -1.0
	if data.kind != "hut": return -1.0
	if SERVICES.has(mode):return visit(data,"service_until",SERVICES[mode],tick)
	if ACTS.has(mode):return visit(data,"entertainment_until",ACTS[mode],tick)
	match mode:
		"crime","malaria","disease": return clampf(float(data.get(mode+"_risk",0))/100,0,1)
		"desirability":return clampf((float(data.get("desirability",0))+30.0)/100.0,0,1)
		"food":return 1.0 if data.get("food",false) else 0.0
		"water":return 1.0 if data.get("water",false) else 0.0
		"tax":return clampf((float(data.get("tax_until",0))-tick)/128.0,0,1)
		"education":return (visit(data,"service_until",4,tick)+visit(data,"service_until",5,tick))*0.5
		"entertainment":
			# The four household shows contribute different strengths in entertainmentScore.
			var points:=int(visit(data,"service_until",7,tick)*960.0/25.0)
			var divisors:=[50.0,40.0,30.0]
			for i in range(3):points+=int(visit(data,"entertainment_until",i,tick)*960.0/float(divisors[i]))
			return float(points)/113.0
		"worship":
			var strength:=0.0
			for i in range(5):strength=maxf(strength,visit(data,"god_until",i,tick))
			return strength
	return -1.0

static func paint(game: Node, item: Dictionary, mode: String) -> void:
	var score:=value(item.building,mode,int(game.state.tick))
	if score<0: return
	var danger:=mode in ["fire","collapse","malaria","disease","crime","labor"]
	var tint:=Color("9c5530").lerp(Color("e6bf65"),score) if mode=="desirability" else Color("a7673c").lerp(Color("9daf73"),score if not danger else 1.0-score)
	var foundation:=tint
	foundation.a=0.58
	game.overlay.draw_colored_polygon(game._footprint(item.rect),foundation)
	if mode=="desirability" or score<=0.0: return
	var tile:Vector2=item.rect.get_center()
	var at:Vector2=game._iso(tile.x,tile.y)
	var top:=at+Vector2(0,-5.0-42.0*score)
	var l:=Vector2(-4,-2)
	var r:=Vector2(4,-2)
	var crown:=Vector2(0,-4)
	game.overlay.draw_colored_polygon(PackedVector2Array([at+l,at,top,top+l]),tint.darkened(0.26))
	game.overlay.draw_colored_polygon(PackedVector2Array([at,at+r,top+r,top]),tint)
	game.overlay.draw_colored_polygon(PackedVector2Array([top+l,top+crown,top+r,top]),tint.lightened(0.27))
