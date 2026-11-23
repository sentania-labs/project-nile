shader_type canvas_item;
// Each authoritative delivered-and-built unit adds another piece of a course.
// The V-shaped baseline follows the painted isometric walls. Sixteen shallow
// courses keep the rising masonry legible without large rectangular cutouts.
uniform int completed : hint_range(0, 64) = 0;
varying vec4 tint;
void vertex() {
	tint = COLOR;
}
void fragment() {
	vec4 painted = texture(TEXTURE, UV);
	float elevation = max(0.0, (1.0 - UV.y) - abs(UV.x - 0.5) * 0.48);
	float row = min(15.0, floor(elevation / 0.82 * 16.0));
	float column = min(3.0, floor(fract(UV.x + mod(row, 2.0) / 8.0) * 4.0));
	float order = row * 4.0 + (mod(row, 2.0) < 0.5 ? column : 3.0 - column);
	painted.a *= step(order + 1.0, float(completed));
	COLOR = painted * tint;
}
