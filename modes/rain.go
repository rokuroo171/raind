package modes

import (
	"math"
	"math/rand"

	"github.com/gdamore/tcell/v2"
)

type Mode int

const (
	ModeRain Mode = iota
	ModeThunderstorm
	ModeSnow
	ModeMeteor
)

type SpeedLevel int

const (
	SpeedSlow SpeedLevel = iota
	SpeedMedium
	SpeedFast
)

type Particle struct {
	X, Y       float64
	VX, VY     float64
	Weight     float64
	Len        int
	Char       rune
	Active     bool
	Splash     int
	SplashChar rune
}

type Snowflake struct {
	X, Y    float64
	Drift   float64
	Speed   float64
	Weight  float64
	Char    rune
	Active  bool
}

type Meteor struct {
	X, Y      float64
	VX, VY    float64
	Life      int
	MaxLife   int
	TrailLen  int
	HeadChar  rune
	Active    bool
}

type Star struct {
	X, Y    int
	Char    rune
	Twinkle int
}

type Spark struct {
	X, Y   float64
	VX, VY float64
	Life   int
	Char   rune
	Active bool
}

type BoltPoint struct {
	X, Y int
	Char rune
}

type BoltStrike struct {
	Points     []BoltPoint
	HaloPoints []BoltPoint
	FramesLeft int
	HaloFrame  bool
}

type State struct {
	Width, Height int
	Mode          Mode
	Speed         SpeedLevel
	Color         tcell.Color
	Frame         int

	// rain or thunderstorm
	Wind           float64
	WindTarget     float64
	WindTick       int
	StormIntensity float64
	Particles      []Particle
	StormFlash     int
	Bolts          []BoltStrike

	// snow
	Flakes []Snowflake

	// meteor
	Meteors           []Meteor
	Stars             []Star
	Sparks            []Spark
	MeteorFlash       int
	MeteorShowerTimer int
}

var rainChars = []rune{'|', ':', '·', '′', '¦'}

// converts a CLI string to Mode
func ParseMode(s string) (Mode, bool) {
	switch s {
	case "rain":
		return ModeRain, true
	case "thunder", "thunderstorm":
		return ModeThunderstorm, true
	case "snow":
		return ModeSnow, true
	case "meteor", "meteors", "shooting", "shower":
		return ModeMeteor, true
	default:
		return ModeRain, false
	}
}

// converts a CLI color name to tcell.Color
func ParseColor(s string) (tcell.Color, bool) {
	switch s {
	case "black":
		return tcell.ColorBlack, true
	case "red":
		return tcell.ColorRed, true
	case "green":
		return tcell.ColorGreen, true
	case "yellow":
		return tcell.ColorYellow, true
	case "blue":
		return tcell.ColorBlue, true
	case "magenta":
		return tcell.ColorPurple, true
	case "cyan":
		return tcell.ColorTeal, true
	case "white":
		return tcell.ColorWhite, true
	default:
		return tcell.ColorTeal, false
	}
}

// this function converts a CLI speed name to SpeedLevel.
func ParseSpeed(s string) (SpeedLevel, bool) {
	switch s {
	case "slow":
		return SpeedSlow, true
	case "medium":
		return SpeedMedium, true
	case "fast":
		return SpeedFast, true
	default:
		return SpeedMedium, false
	}
}

// this function returns a scalar for drop motion.
func (s SpeedLevel) SpeedMultiplier() float64 {
	switch s {
	case SpeedSlow:
		return 0.6
	case SpeedFast:
		return 1.6
	default:
		return 1.0
	}
}

// this function returns the sleep duration between frames for a mode.
func FrameDelay(mode Mode, speed SpeedLevel) int {
	base := 45
	switch mode {
	case ModeThunderstorm:
		base = 28
	case ModeSnow:
		base = 80
	case ModeMeteor:
		base = 35
	}
	switch speed {
	case SpeedSlow:
		base = base * 3 / 2
	case SpeedFast:
		base = base * 2 / 3
	}
	if base < 15 {
		base = 15
	}
	return base
}

func randomParticleWeight() float64 {
	return 0.3 + rand.Float64()*0.7
}

func particleFromWeight(w float64) Particle {
	vy := 0.4 + w*0.6
	return Particle{
		Weight: w,
		VY:     vy,
		Len:    1 + int(w*3),
		Char:   rainChars[rand.Intn(len(rainChars))],
		Active: true,
	}
}

func (st *State) initParticles(density float64) {
	if st.Width <= 0 || st.Height <= 0 {
		st.Particles = nil
		return
	}
	n := int(float64(st.Width) * density)
	if n < 1 {
		n = 1
	}
	maxN := st.Width * st.Height / 4
	if maxN < 1 {
		maxN = 1
	}
	if n > maxN {
		n = maxN
	}
	particles := make([]Particle, n)
	for i := range particles {
		w := randomParticleWeight()
		p := particleFromWeight(w)
		p.X = float64(rand.Intn(st.Width))
		p.Y = float64(rand.Intn(st.Height))
		particles[i] = p
	}
	st.Particles = particles
}

func (st *State) targetParticleCount(density float64) int {
	if st.Width <= 0 {
		return 0
	}
	n := int(float64(st.Width) * density)
	if n < 1 {
		n = 1
	}
	maxN := st.Width * st.Height / 4
	if maxN < 1 {
		maxN = 1
	}
	if n > maxN {
		n = maxN
	}
	return n
}

func (st *State) initSnow() {
	if st.Width <= 0 || st.Height <= 0 {
		st.Flakes = nil
		return
	}
	n := (st.Width * st.Height) / 120
	if n < 8 {
		n = 8
	}
	lightChars := []rune{'·', '∗', '❄', '✻'}
	heavyChars := []rune{'|', '•'}
	st.Flakes = make([]Snowflake, n)
	for i := range st.Flakes {
		w := 0.2 + rand.Float64()*0.8
		var ch rune
		if w > 0.7 {
			ch = heavyChars[rand.Intn(len(heavyChars))]
		} else if w < 0.4 {
			ch = lightChars[rand.Intn(len(lightChars))]
		} else {
			all := append(append([]rune{}, lightChars...), heavyChars...)
			ch = all[rand.Intn(len(all))]
		}
		st.Flakes[i] = Snowflake{
			X:      float64(rand.Intn(st.Width)),
			Y:      float64(rand.Intn(st.Height)),
			Weight: w,
			Speed:  0.08 + w*0.18,
			Drift:  (rand.Float64() - 0.5) * (0.25 - w*0.15),
			Char:   ch,
			Active: true,
		}
	}
}

func (st *State) initMeteors() {
	if st.Width <= 0 || st.Height <= 0 {
		st.Meteors = nil
		st.Stars = nil
		st.Sparks = nil
		return
	}
	nStars := (st.Width * st.Height) / 180
	if nStars < 24 {
		nStars = 24
	}
	starChars := []rune{'.', '·', '+', '∙', '˙'}
	st.Stars = make([]Star, nStars)
	for i := range st.Stars {
		st.Stars[i] = Star{
			X:       rand.Intn(st.Width),
			Y:       rand.Intn(st.Height),
			Char:    starChars[rand.Intn(len(starChars))],
			Twinkle: rand.Intn(60),
		}
	}
	st.Meteors = make([]Meteor, 32)
	st.Sparks = make([]Spark, 48)
	st.MeteorFlash = 0
	st.MeteorShowerTimer = 120 + rand.Intn(180)
}

func (st *State) Resize(w, h int) {
	st.Width = w
	st.Height = h
	switch st.Mode {
	case ModeThunderstorm:
		density := 0.35 + st.StormIntensity*0.45
		st.initParticles(density)
	case ModeRain:
		st.initParticles(0.35)
	case ModeSnow:
		st.initSnow()
	case ModeMeteor:
		st.initMeteors()
	}
}

func rainStyle(color tcell.Color, dim bool) tcell.Style {
	s := tcell.StyleDefault.Foreground(color).Background(tcell.ColorBlack)
	if dim {
		s = s.Attributes(tcell.AttrDim)
	}
	return s
}

func meteorStyle(color tcell.Color, dim, bold bool) tcell.Style {
	s := tcell.StyleDefault.Foreground(color).Background(tcell.ColorBlack)
	var attr tcell.AttrMask
	if dim {
		attr |= tcell.AttrDim
	}
	if bold {
		attr |= tcell.AttrBold
	}
	if attr != 0 {
		s = s.Attributes(attr)
	}
	return s
}

func (st *State) updateWind() {
	if st.Wind < st.WindTarget {
		st.Wind += 0.002
		if st.Wind > st.WindTarget {
			st.Wind = st.WindTarget
		}
	} else if st.Wind > st.WindTarget {
		st.Wind -= 0.002
		if st.Wind < st.WindTarget {
			st.Wind = st.WindTarget
		}
	}
	st.WindTick++
	if st.WindTick >= 180+rand.Intn(121) {
		st.WindTick = 0
		if st.Mode == ModeThunderstorm {
			st.WindTarget = -0.9 + rand.Float64()*1.8
		} else {
			st.WindTarget = -0.6 + rand.Float64()*1.2
		}
	}
}

func (st *State) updateRain() {
	if st.Width <= 0 || st.Height <= 0 {
		return
	}
	st.updateWind()
	mult := st.Speed.SpeedMultiplier()
	w := float64(st.Width)

	for i := range st.Particles {
		p := &st.Particles[i]
		if !p.Active {
			continue
		}
		if p.Splash > 0 {
			p.Splash--
			continue
		}
		p.VX = st.Wind * 0.4 * p.Weight
		p.X += p.VX
		p.Y += p.VY * mult

		if p.X < 0 {
			p.X += w
		}
		if p.X >= w {
			p.X -= w
		}

		if p.Y >= float64(st.Height) {
			p.Splash = 2
			if p.SplashChar == 0 {
				p.SplashChar = '~'
			} else if p.SplashChar == '~' {
				p.SplashChar = '∿'
			} else {
				p.SplashChar = '~'
			}
			p.Y = 0
			p.X = float64(rand.Intn(st.Width))
			wgt := randomParticleWeight()
			np := particleFromWeight(wgt)
			p.Weight = np.Weight
			p.VY = np.VY
			p.Len = np.Len
			p.Char = np.Char
		}
	}
}

func (st *State) drawRain(screen tcell.Screen, splash bool) {
	w, h := st.Width, st.Height
	if w == 0 || h == 0 {
		return
	}
	style := rainStyle(st.Color, false)
	trailStyle := rainStyle(st.Color, true)
	bottom := h - 1

	for i := range st.Particles {
		p := &st.Particles[i]
		if !p.Active {
			continue
		}
		if p.Splash > 0 && splash {
			sx := int(p.X) % w
			if sx < 0 {
				sx += w
			}
			if sx >= 0 && sx < w {
				screen.SetContent(sx, bottom, p.SplashChar, nil, style)
			}
			continue
		}
		headY := int(p.Y)
		if headY < 0 || headY >= h {
			continue
		}
		headX := int(p.X) % w
		if headX < 0 {
			headX += w
		}
		for j := 1; j <= p.Len; j++ {
			sy := headY - j
			if sy < 0 {
				break
			}
			sx := headX - int(math.Round(float64(j)*p.VX*0.5))
			sx = sx % w
			if sx < 0 {
				sx += w
			}
			if sx >= 0 && sx < w {
				screen.SetContent(sx, sy, p.Char, nil, trailStyle)
			}
		}
		if headX >= 0 && headX < w {
			screen.SetContent(headX, headY, p.Char, nil, style)
		}
		if splash && headY >= bottom-1 {
			screen.SetContent(headX, bottom, '~', nil, style)
		}
	}
}

func Draw(screen tcell.Screen, st *State) {
	st.updateRain()
	st.drawRain(screen, true)
}

func (st *State) InitRain() {
	st.StormIntensity = 0
	st.Wind = 0
	st.WindTarget = 0
	st.WindTick = 0
	st.Bolts = nil
	st.initParticles(0.35)
}
