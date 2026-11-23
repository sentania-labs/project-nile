shader_type canvas_item;
uniform float progress : hint_range(0.0,1.0) = 0.0;
uniform float courses = 24.0;
uniform bool carving = false;
varying vec4 tint;
void vertex(){tint=COLOR;}
void fragment(){
 vec4 painted=texture(TEXTURE,UV);
 // Finished painted masonry appears in physical course order. Fractional
 // progress reveals individual sections, avoiding a whole-building fade.
 float height=max(0.0,(0.95-UV.y)-abs(UV.x-0.5)*0.46);
 float row=min(courses-1.0,floor(height/0.72*courses));
 float column=floor(fract(UV.x+mod(row,2.0)/32.0)*16.0);
 float order=(row*16.0+column+1.0)/(courses*16.0);
 if(carving){order=(floor(UV.x*8.0)+floor(UV.y*6.0)*8.0+1.0)/48.0;}
 painted.a*=step(order,progress);
 COLOR=painted*tint;
}
