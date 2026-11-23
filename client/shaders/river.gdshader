shader_type canvas_item;
render_mode unshaded;

uniform vec4 atlas_region;
uniform vec2 tile_size = vec2(96.0, 48.0);
uniform int orientation = 0;
uniform float flood_level = 0.0;
varying vec2 projected;

void vertex() {
	projected = VERTEX;
}

void fragment() {
	vec2 axes = vec2(projected.x / tile_size.x + projected.y / tile_size.y,
		projected.y / tile_size.y - projected.x / tile_size.x);
	vec2 land = axes;
	if (orientation == 1) { land = vec2(axes.y, -axes.x); }
	if (orientation == 2) { land = -axes; }
	if (orientation == 3) { land = vec2(-axes.y, axes.x); }
	float bank = 5.83 + 0.10 * sin(land.y * 0.77) + 0.07 * sin(land.y * 2.49);
	float edge = mix(bank, 9.82 + 0.15 * sin(land.y * 0.57), flood_level);
	float depth = edge - land.x;
	float wave = sin(land.y * 2.7 + TIME * 1.4 + sin(land.x * 3.8));
	depth += 0.026 * wave;
	vec2 drift = vec2(sin(land.y * 2.7 + TIME) * 0.005, sin(TIME * 0.49 + land.x) * 0.004);
	vec2 uv = 1.0 - abs(mod(UV + drift, vec2(2.0)) - 1.0);
	vec2 inset = TEXTURE_PIXEL_SIZE * 0.5;
	vec3 blue = texture(TEXTURE, atlas_region.xy + inset + uv * (atlas_region.zw - 2.0 * inset)).rgb;
	float shallow = 1.0 - smoothstep(0.05, 1.7, depth);
	vec3 color = mix(blue, vec3(0.35, 0.49, 0.38), shallow * 0.74);
	float rim = (1.0 - smoothstep(0.01, 0.105, abs(depth - 0.065))) * (0.65 + 0.35 * wave);
	color += vec3(0.16, 0.16, 0.09) * rim;
	COLOR = vec4(color, smoothstep(-0.06, 0.13, depth));
}
