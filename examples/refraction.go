package main

import . "github.com/fogleman/pt/pt"

func main() {
	scene := Scene{}

	glass := ClearMaterial(1.5, 0)

	// add a sphere primitive
	scene.Add(NewSphere(V(-1.5, 0, 0.5), 1, glass))

	// add a mesh sphere
	mesh, err := LoadSTL("examples/sphere.stl", glass)
	if err != nil {
		panic(err)
	}
	mesh.SaveSTL("examples/sphere2.stl")
	mesh.SmoothNormals()
	mesh.Transform(Translate(V(1.5, 0, 0.5)))
	scene.Add(mesh)

	// add the floor
	scene.Add(NewPlane(V(0, 0, -1), V(0, 0, 1), DiffuseMaterial(Color{1, 1, 1})))

	// add the light
	scene.Add(NewSphere(V(0, 0, 5), 1, LightMaterial(Color{1, 1, 1}, 30)))

	camera := LookAt(V(0, -5, 5), V(0, 0, 0), V(0, 0, 1), 50)
	sampler := NewSampler(16, 8)
	sampler.SpecularMode = SpecularModeAll
	IterativeRender("out%03d.png", 1000, &scene, &camera, sampler, 1920/4, 1080/4, -1)
}
