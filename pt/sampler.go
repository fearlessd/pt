package pt

import (
	"math"
	"math/rand"
)

type SpecularMode int

const (
	SpecularModeNaive = iota
	SpecularModeFirst
	SpecularModeAll
)

type BounceMode int

const (
	BounceModeAny = iota
	BounceModeDiffuse
	BounceModeSpecular
)

type Sampler interface {
	Sample(scene *Scene, ray Ray, rnd *rand.Rand) Color
}

func NewSampler(firstHitSamples, maxBounces int) *DefaultSampler {
	return &DefaultSampler{firstHitSamples, maxBounces, true, SpecularModeNaive}
}

type DefaultSampler struct {
	FirstHitSamples int
	MaxBounces      int
	DirectLighting  bool
	SpecularMode    SpecularMode
}

func (s *DefaultSampler) Sample(scene *Scene, ray Ray, rnd *rand.Rand) Color {
	return s.sample(scene, ray, true, s.FirstHitSamples, s.MaxBounces, rnd)
}

func (s *DefaultSampler) sample(scene *Scene, ray Ray, emission bool, samples, depth int, rnd *rand.Rand) Color {
	if depth < 0 {
		return Color{}
	}
	hit := scene.Intersect(ray)
	if !hit.Ok() {
		if scene.Texture != nil {
			d := ray.Direction
			u := math.Atan2(d.Z, d.X) + scene.TextureAngle
			v := math.Atan2(d.Y, Vector{d.X, 0, d.Z}.Length())
			u = (u + math.Pi) / (2 * math.Pi)
			v = (v + math.Pi/2) / math.Pi
			return scene.Texture.Sample(u, v)
		}
		return scene.Color
	}
	info := hit.Info(ray)
	material := info.Material
	result := Color{}
	if material.Emittance > 0 {
		if s.DirectLighting && !emission {
			return Color{}
		}
		result = result.Add(material.Color.MulScalar(material.Emittance * float64(samples)))
	}
	n := int(math.Sqrt(float64(samples)))
	var ma, mb BounceMode
	if s.SpecularMode == SpecularModeAll || (s.SpecularMode == SpecularModeFirst && n > 1) {
		ma = BounceModeDiffuse
		mb = BounceModeSpecular
	} else {
		ma = BounceModeAny
		mb = BounceModeAny
	}
	for u := 0; u < n; u++ {
		for v := 0; v < n; v++ {
			for mode := ma; mode <= mb; mode++ {
				fu := (float64(u) + rnd.Float64()) / float64(n)
				fv := (float64(v) + rnd.Float64()) / float64(n)
				newRay, reflected, p := ray.Bounce(&info, fu, fv, mode, rnd)
				if mode == BounceModeAny {
					p = 1
				}
				if p > 0 && reflected {
					// specular
					indirect := s.sample(scene, newRay, reflected, 1, depth-1, rnd)
					tinted := indirect.Mix(material.Color.Mul(indirect), material.Tint)
					result = result.Add(tinted.MulScalar(p))
				}
				if p > 0 && !reflected {
					// diffuse
					indirect := s.sample(scene, newRay, reflected, 1, depth-1, rnd)
					direct := Color{}
					if s.DirectLighting {
						direct = s.directLight(scene, info.Ray, rnd)
					}
					result = result.Add(material.Color.Mul(direct.Add(indirect)).MulScalar(p))
				}
			}
		}
	}
	return result.DivScalar(float64(n * n))
}

func (s *DefaultSampler) directLight(scene *Scene, n Ray, rnd *rand.Rand) Color {
	nLights := len(scene.Lights)
	if nLights == 0 {
		return Color{}
	}

	// pick a random light
	light := scene.Lights[rand.Intn(nLights)]

	// get bounding sphere center and radius
	var center Vector
	var radius float64
	switch t := light.(type) {
	case *Sphere:
		radius = t.Radius
		center = t.Center
	default:
		// get bounding sphere from bounding box
		box := t.BoundingBox()
		radius = box.OuterRadius()
		center = box.Center()
	}

	// get random point in disk
	var point Vector
	for {
		x := rnd.Float64()*2 - 1
		y := rnd.Float64()*2 - 1
		if x*x+y*y <= 1 {
			l := center.Sub(n.Origin).Normalize()
			u := l.Cross(RandomUnitVector(rnd)).Normalize()
			v := l.Cross(u)
			point = point.Add(u.MulScalar(x * radius))
			point = point.Add(v.MulScalar(y * radius))
			point = point.Add(center)
			break
		}
	}

	// construct ray toward light point
	ray := Ray{n.Origin, point.Sub(n.Origin).Normalize()}

	// get cosine term
	diffuse := ray.Direction.Dot(n.Direction)
	if diffuse <= 0 {
		return Color{}
	}

	// check for light visibility
	hit := scene.Intersect(ray)
	if !hit.Ok() || hit.Shape != light {
		return Color{}
	}

	// compute solid angle (hemisphere coverage)
	hyp := center.Sub(n.Origin).Length()
	opp := radius
	theta := math.Asin(opp / hyp)
	adj := opp / math.Tan(theta)
	d := math.Cos(theta) * adj
	r := math.Sin(theta) * adj
	coverage := (r * r) / (d * d)

	// TODO: fix issue where hyp < opp (point inside sphere)
	if hyp < opp {
		coverage = 1
	}
	coverage = math.Min(coverage, 1)

	// get material properties from light
	material := MaterialAt(light, point)

	// combine factors
	m := material.Emittance * diffuse * coverage * float64(nLights)
	return material.Color.MulScalar(m)
}
