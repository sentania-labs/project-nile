shader_type canvas_item;
uniform float progress : hint_range(0.0,1.0) = 0.0;
uniform float courses = 24.0;
uniform bool carving = false;
uniform float front_x=0.5;
uniform float front_y=0.95;
uniform float left_slope=0.46;
uniform float right_slope=0.46;
uniform float wall_height=0.72;
varying vec4 tint;
void vertex(){tint=COLOR;}
void fragment(){
 vec4 painted=texture(TEXTURE,UV);
 // Finished painted masonry appears in physical course order. Fractional
 // progress reveals individual sections, avoiding a whole-building fade.
 float height=max(0.0,(front_y-UV.y)-abs(UV.x-front_x)*(UV.x<front_x?left_slope:right_slope));
 float row=min(courses-1.0,floor(height/wall_height*courses));
 float column=floor(fract(UV.x+mod(row,2.0)/32.0)*16.0);
 float order=(row*16.0+column+1.0)/(courses*16.0);
 if(carving){order=(floor(UV.x*8.0)+floor(UV.y*6.0)*8.0+1.0)/48.0;}
 painted.a*=step(order,progress);
 COLOR=painted*tint;
}
