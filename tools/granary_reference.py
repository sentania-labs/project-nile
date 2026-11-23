"""Original Codex Gate 0 response to fixed-asset-task.txt, not historical reconstruction."""
import argparse,sys
from pathlib import Path
import bpy
p=argparse.ArgumentParser(); p.add_argument('--output',required=True,type=Path); a=p.parse_args(sys.argv[sys.argv.index('--')+1:]); a.output.mkdir(parents=True,exist_ok=True)
bpy.ops.wm.read_factory_settings(use_empty=True)
bpy.context.scene.unit_settings.system='METRIC'; bpy.context.scene.unit_settings.scale_length=1.0
colors={'plaster':(0.73,0.54,0.32,1),'mudbrick':(0.43,0.25,0.13,1),'timber':(0.13,0.07,0.035,1),'grain':(0.72,0.44,0.10,1)}
materials={}
for name,color in colors.items():
 m=bpy.data.materials.new(name); m.use_nodes=True; s=m.node_tree.nodes.get('Principled BSDF'); s.inputs['Base Color'].default_value=color; s.inputs['Metallic'].default_value=0; s.inputs['Roughness'].default_value=0.85; materials[name]=m

def cube(name,loc,size,mat,bevel=0):
 bpy.ops.mesh.primitive_cube_add(size=1,location=loc); o=bpy.context.object; o.name=name; o.dimensions=size; bpy.ops.object.transform_apply(location=False,rotation=False,scale=True); o.data.materials.append(materials[mat])
 if bevel:
  b=o.modifiers.new('Soft earthen edges','BEVEL'); b.width=bevel; b.segments=1; bpy.ops.object.modifier_apply(modifier=b.name)
 return o

def cylinder(name,loc,radius,depth,mat):
 bpy.ops.mesh.primitive_cylinder_add(vertices=16,radius=radius,depth=depth,location=loc); o=bpy.context.object; o.name=name; o.data.materials.append(materials[mat]); return o
# Pad fixes the 3x2 tile footprint. Entrance faces -Y.
cube('Earthen foundation',(0,0,.10),(6,4,.20),'mudbrick',.04)
cube('Storage room',(-1.4,.2,1.38),(2.6,3.2,2.36),'plaster',.055)
cube('Flat roof',(-1.4,.2,2.57),(2.72,3.3,.16),'mudbrick',.025)
for name,loc,size in [('West',(-2.68,.2,2.81),(.16,3.3,.38)),('East',(-.12,.2,2.81),(.16,3.3,.38)),('Back',(-1.4,1.77,2.81),(2.72,.16,.38))]: cube('Roof parapet '+name,loc,size,'plaster',.02)
cube('Recessed dark doorway',(-1.4,-1.415,1.02),(.72,.06,1.56),'timber')
for x in [-1.84,-.96]: cube('Door jamb',(x,-1.48,1.07),(.14,.22,1.72),'mudbrick',.015)
cube('Door lintel',(-1.4,-1.48,1.93),(1.04,.25,.16),'timber',.015)
for i in range(3): cube('Loading step '+str(i),(-1.4,-1.84+i*.13,.23+i*.09),(1.15,.24,.12),'plaster',.012)
# Two hollow grain bins built from a ring of upright staves of masonry.
import math
for j,y in enumerate([-.9,.85]):
 x=1.35; cylinder('Bin floor '+str(j),(x,y,.29),.70,.18,'mudbrick')
 for i in range(16):
  t=2*math.pi*i/16; o=cube('Bin wall %d %02d'%(j,i),(x+.63*math.cos(t),y+.63*math.sin(t),.93),(.27,.16,1.22),'plaster',.015); o.rotation_euler.z=t+math.pi/2
 cylinder('Visible grain '+str(j),(x,y,1.35),.54,.10,'grain')
# Small loading bundles, no external textures or copied objects.
for x,y in [(2.35,-1.35),(2.48,-.9),(2.4,-.4)]: cylinder('Grain sack',(x,y,.48),.19,.56,'grain')
for i in range(3): cube('Roof timber '+str(i),(-1.4,-.8+i*.8,2.68),(2.3,.09,.09),'timber',.015)
for o in list(bpy.context.scene.objects):
 bpy.context.view_layer.objects.active=o; bpy.ops.object.select_all(action='DESELECT'); o.select_set(True); bpy.ops.object.transform_apply(location=True,rotation=True,scale=True)
bpy.ops.object.select_all(action='SELECT')
bpy.ops.wm.save_as_mainfile(filepath=str((a.output/'master.blend').resolve()))
bpy.ops.export_scene.gltf(filepath=str((a.output/'raw.glb').resolve()),export_format='GLB',export_yup=True,export_animations=False,use_selection=True)
print('NILE_ORIGINAL_GRANARY_GENERATED')
