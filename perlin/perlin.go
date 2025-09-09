package perlin

import (
	"math"
	"math/rand"
)

// Perlin holds the duplicated permutation table (512 entries).
type Perlin struct {
	p []int
}

// NewPerlin creates a Perlin instance seeded deterministically.
func NewPerlin(seed int64) *Perlin {
	r := rand.New(rand.NewSource(seed))
	base := r.Perm(256)

	p := &Perlin{p: make([]int, 512)}
	for i := 0; i < 256; i++ {
		p.p[i] = base[i]
		p.p[256+i] = base[i]
	}
	return p
}

// fade implements the Perlin fade curve 6t^5 - 15t^4 + 10t^3.
func fade(t float64) float64 {
	return t*t*t*(t*(t*6-15)+10)
}

// lerp performs linear interpolation.
func lerp(t, a, b float64) float64 {
	return a + t*(b-a)
}

// grad converts a hash into one of 4 diagonal gradients and returns the dot product.
func (p *Perlin) grad(hash int, x, y float64) float64 {
	h := hash & 3
	switch h {
	case 0:
		return x + y
	case 1:
		return -x + y
	case 2:
		return x - y
	default: // case 3
		return -x - y
	}
}

// Noise2DRaw returns 2D Perlin noise approximately in [-1, 1].
// x,y are world coords; freq is frequency multiplier (larger freq -> more detail).
func (p *Perlin) Noise2DRaw(x, y, freq float64) float64 {
	xf := x * freq
	yf := y * freq

	xi := int(math.Floor(xf)) & 255
	yi := int(math.Floor(yf)) & 255

	xf = xf - math.Floor(xf)
	yf = yf - math.Floor(yf)

	u := fade(xf)
	v := fade(yf)

	aa := p.p[p.p[xi]+yi]
	ab := p.p[p.p[xi]+yi+1]
	ba := p.p[p.p[xi+1]+yi]
	bb := p.p[p.p[xi+1]+yi+1]

	x1 := lerp(u, p.grad(aa, xf, yf), p.grad(ba, xf-1, yf))
	x2 := lerp(u, p.grad(ab, xf, yf-1), p.grad(bb, xf-1, yf-1))

	return lerp(v, x1, x2)
}

// Noise2D returns normalized Perlin noise in [0,1] (wrapper around Noise2DRaw).
func (p *Perlin) Noise2D(x, y, freq float64) float64 {
	return (p.Noise2DRaw(x, y, freq) + 1.0) * 0.5
}

// FBM2DRaw returns fractal brownian motion using raw Perlin noise in approx [-1,1].
// octaves is integer number of octaves; persistence < 1 reduces amplitude each octave;
// lacunarity > 1 increases frequency each octave.
func (p *Perlin) FBM2DRaw(x, y, baseFreq float64, octaves int, persistence, lacunarity float64) float64 {
	total := 0.0
	amplitude := 1.0
	frequency := baseFreq
	maxAmp := 0.0

	for i := 0; i < octaves; i++ {
		total += p.Noise2DRaw(x, y, frequency) * amplitude
		maxAmp += amplitude
		amplitude *= persistence
		frequency *= lacunarity
	}

	if maxAmp == 0 {
		return 0
	}
	return total / maxAmp
}

// FBM2D is a compatibility wrapper similar to your original FBM2D signature.
// It accepts octaves/persistence/lacunarity as floats like before and returns [0,1].
func (p *Perlin) FBM2D(x, y, baseFreq, octaves, persistence, lacunarity float64) float64 {
	octs := int(octaves)
	raw := p.FBM2DRaw(x, y, baseFreq, octs, persistence, lacunarity)
	// normalize to [0,1]
	return (raw + 1.0) * 0.5
}

// NoiseFlow returns a signed 2D flow vector in approximately [-1,1] per component.
// Use this to perturb coordinates (e.g. tectonic/flow field). The returned values are raw (signed).
func (p *Perlin) NoiseFlow(x, y, freq float64) (float64, float64) {
	// Use raw noise for signed flow and offset second component to decorrelate.
	xFlow := p.Noise2DRaw(x, y, freq)
	yFlow := p.Noise2DRaw(x+100.0, y+100.0, freq)
	return xFlow, yFlow
}
