package block

import "encoding/json"

type Vec3 struct {
	x float32
	y float32
	z float32
}

func (v *Vec3) UnmarshalJSON(data []byte) error {
	var values []float32
	if err := json.Unmarshal(data, &values); err != nil {
		return err
	}

	v.x = values[0]
	v.y = values[1]
	v.z = values[2]

	return nil
}

type Rect struct {
	x1 float32
	y1 float32
	x2 float32
	y2 float32
}

func (r *Rect) UnmarshalJSON(data []byte) error {
	var values []float32
	if err := json.Unmarshal(data, &values); err != nil {
		return err
	}

	r.x1 = values[0]
	r.y1 = values[1]
	r.x2 = values[2]
	r.y2 = values[3]

	return nil
}

type ModelRotation struct {
	// Center of rotation
	Origin Vec3 `json:"origin"`

	// Axis about which to rotate
	Axis string `json:"axis"`

	// Value from -45 to 45 in 22.5 degree increments
	Angle float32 `json:"angle"`

	// Whether to scale the faces across the whole block
	Rescale bool `json:"rescale"`
}

type ModelFace struct {
	// Area of the texture to use for this face
	Uv Rect `json:"uv"`

	// Resource location of the texture
	Texture string `json:"texture"`

	// Whether to skip rendering the face if there is a neighboring block in the
	// specified direction
	// Valid cullfaces are: down,up,north,south,east,west
	Cullface string `json:"cullface"`

	// Rotate the texture region by this amount
	Rotation int `json:"rotation"`
}

type ModelElement struct {
	// Starting point of the element's cuboid
	From Vec3 `json:"from"`
	// Ending point of the element's cuboid
	To Vec3 `json:"to"`

	Rotation ModelRotation `json:"rotation"`

	// Whether to render shadows
	Shade bool `json:"shade"`

	// Valid face keys are: down,up,north,south,east,west. Not all face keys need to be present
	Faces map[string]ModelFace `json:"faces"`
}

type BlockModel struct {
	// Resource name of the parent block (inherits all non-overridden properties)
	Parent string `json:"parent"`
	// Textures map a texture variable to a resource location (`assets/<namespace>/textures/<resource_name>.jpg`)
	Textures map[string]string `json:"textures"`

	// Elements describe how the block model is rendered, using cubic forms.
	Elements []ModelElement `json:"elements"`
}
