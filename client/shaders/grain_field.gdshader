shader_type canvas_item;
render_mode unshaded;

uniform float growth = 1.0;
uniform float flood_level = 0.0;
uniform int orientation = 0;
uniform vec2 tile_size = vec2(96.0, 48.0);
varying vec2 projected;
varying vec4 tint;

void vertex() {
	projected = (MODEL_MATRIX * vec4(VERTEX, 0.0, 1.0)).xy;
	tint = COLOR;
}

void fragment() {
	vec4 pixel = texture(TEXTURE, UV);
	// The open shelter and its jars stand on a raised platform.
	float shelter = smoothstep(0.55, 0.62, UV.x) * (1.0 - smoothstep(0.35, 0.44, UV.y));
	float grain = smoothstep(0.07, 0.25, pixel.g - pixel.b)
		* smoothstep(0.0, 0.24, pixel.r - pixel.b) * (1.0 - shelter);
	float light = dot(pixel.rgb, vec3(0.35, 0.5, 0.15));
	vec3 soil = vec3(0.24, 0.16, 0.10) * (0.8 + light * 0.48);
	vec3 shoots = vec3(0.34, 0.45, 0.13) * (0.42 + light * 0.85);
	vec3 crop = mix(soil, shoots, smoothstep(0.01, 0.35, growth));
	crop = mix(crop, pixel.rgb, smoothstep(0.46, 0.92, growth));
	vec2 axes = vec2(projected.x / tile_size.x + projected.y / tile_size.y,
		projected.y / tile_size.y - projected.x / tile_size.x);
	float x = axes.x;
	float y = axes.y;
	if (orientation == 1) { x = axes.y; y = -axes.x; }
	if (orientation == 2) { x = -axes.x; y = -axes.y; }
	if (orientation == 3) { x = -axes.y; y = axes.x; }
	float bank = 5.83 + 0.1 * sin(y * 0.77) + 0.07 * sin(y * 2.49);
	float depth = mix(bank, 9.82 + 0.15 * sin(y * 0.57), flood_level) - x;
	float submerged = smoothstep(-0.08, 0.30, depth) * (1.0 - shelter * 0.83);
	COLOR = vec4(mix(pixel.rgb, crop, grain), pixel.a * (1.0 - submerged)) * tint;
}
