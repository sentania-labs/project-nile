shader_type canvas_item;
uniform int completed : hint_range(0, 24000) = 0;
uniform bool base_only = false;
varying vec4 tint;
void vertex() { tint = COLOR; }
void fragment() {
 vec4 painted=texture(TEXTURE,UV);
 // The painted base follows the same front-corner anchor as its footprint.
 float height=max(0.0,(0.95-UV.y)-abs(UV.x-0.5)*0.46);
 float course=base_only?min(5.0,floor(height/0.36*6.0)):min(11.0,floor(height/0.39*12.0));
 float sections=course<6.0?16.0:4.0;
 float column=min(sections-1.0,floor(fract(UV.x+mod(course,2.0)/sections/2.0)*sections));
 float order=course<6.0?course*16.0+column:96.0+(course-6.0)*4.0+column;
 painted.a*=step((order+1.0)*200.0,float(completed));
 COLOR=painted*tint;
}
