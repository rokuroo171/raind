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
	ModeCalm
	ModeAuto
)

type SpeedLevel int

const (
	SpeedSlow SpeedLevel = iota
	SpeedMedium
	SpeedFast
)

// Plane is the depth layer a drop belongs to: 0 near, 1 mid, 2 far.
const (
	PlaneNear = iota
	PlaneMid
	PlaneFar
)

type Particle struct {
	X, Y       float64
	VX, VY     float64
	Weight     float64
	Len        int
	Char       rune
	Plane      int
	Active     bool
	Splash     int
	SplashY    int
	SplashChar rune
}

type Snowflake struct {
	X, Y   float64
	Drift  float64
	Speed  float64
	Weight float64
	Char   rune
	Active bool
}

type Meteor struct {
	X, Y     float64
	VX, VY   float64
	Life     int
	MaxLife  int
	TrailLen int
	HeadChar rune
	Active   bool
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

// BoltChannel is the ionized path of one strike: the main trunk plus side
// branches, re-drawn for each stroke in the flash.
type BoltChannel struct {
	Points   []BoltPoint
	Branches [][]BoltPoint
}

type BoltStrike struct {
	Channel    BoltChannel
	FramesLeft int
	StrokeGap  int
	Strokes    int
	HaloFrame  bool
	Power      float64

	// leader phase: a faint probe descends the channel in steps before the
	// bright return strokes, only where the cell is close enough to see it
	LeaderFrames int
	LeaderLen    int
	StrokeNum    int
}

type State struct {
	Width, Height int
	Mode          Mode
	Speed         SpeedLevel
	Color         tcell.Color
	Frame         int
	FocusMode     bool

	// rain or thunderstorm
	Wind           float64
	WindTarget     float64
	WindLive       bool
	WindTargetLive float64
	WindTick       int
	GustX          float64
	GustWidth      float64
	GustStrength   float64
	GustDir        float64
	GustTimer      int
	StormIntensity float64
	Particles      []Particle
	StormFlash     int
	Bolts          []BoltStrike

	// cloud flash: a strike that lights the cloud interior and never
	// reaches ground, the storm's most common flash
	CloudGlow  int
	CloudGlowX int

	// storm cells and cadence
	CellPos           float64
	CellDist          float64
	CellDrift         float64
	CellApproach      float64
	BurstLeft         int
	BurstTimer        int
	LullTimer         int
	BurstDoubleChance float64

	// snow
	Flakes    []Snowflake
	AccumRow  []int
	RoofAccum []int

	// meteor
	Meteors             []Meteor
	Stars               []Star
	Sparks              []Spark
	MeteorFlash         int
	MeteorShowerTimer   int
	MeteorFireballTimer int

	// the radiant: every trail points back at this drifting point, the
	// perspective origin that makes the shower read as deep space
	RadiantX     float64
	RadiantY     float64
	RadiantDrift float64

	// bolide: a rare heavy meteor whose trail tints the sky for a moment
	BolideFlash  int
	BolideChance float64

	// world
	World   WorldKind
	Coast   CoastScene
	Seed    int64
	City    Skyline
	Night   float64
	Clouds  []Cloud
	RoofTop []int

	// live weather
	Weather     WeatherData
	WeatherLive bool
	Intensity   float64
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
	case "calm", "clear", "sunny", "fair":
		return ModeCalm, true
	case "auto":
		return ModeAuto, true
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

// thisconverts a CLI speed name to SpeedLevel.
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

func FrameDelay(mode Mode, speed SpeedLevel) int {
	base := 45
	switch mode {
	case ModeThunderstorm:
		base = 28
	case ModeSnow:
		base = 80
	case ModeMeteor:
		base = 35
	case ModeCalm:
		base = 100
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

// planeParams gives the feel of one depth plane. Far drops are sparse, slow,
// short, and near-vertical; near drops are the opposite and lean with the wind.
func planeParams(plane int) (vyMin, vyMax float64, maxLen int, windFactor, weightMin, weightMax float64) {
	switch plane {
	case PlaneNear:
		return 0.9, 1.6, 4, 1.3, 0.6, 1.0
	case PlaneMid:
		return 0.55, 1.0, 2, 0.8, 0.3, 0.7
	default:
		return 0.3, 0.6, 1, 0.25, 0.1, 0.4
	}
}

func particleForPlane(plane int) Particle {
	vyMin, vyMax, maxLen, _, wMin, wMax := planeParams(plane)
	w := wMin + rand.Float64()*(wMax-wMin)
	return Particle{
		Weight: w,
		VY:     vyMin + rand.Float64()*(vyMax-vyMin),
		Len:    1 + rand.Intn(maxLen),
		Char:   rainChars[rand.Intn(len(rainChars))],
		Plane:  plane,
		Active: true,
	}
}

// countsFor splits the particle budget across planes by intensity. Light rain
// fills only the far plane, a downpour fills and weights the near plane.
// Multipliers tuned for screensaver density: fills the screen without
// overwhelming slow terminals.
func (st *State) countsFor(intensity float64) (farN, midN, nearN, total int) {
	w := st.Width
	if w <= 0 {
		return 0, 0, 0, 0
	}
	// 2.5× the original density for a richer field of rain
	farN = int(float64(w) * (0.30 + intensity*0.45))
	midN = int(float64(w) * intensity * 0.85)
	nearN = int(float64(w) * intensity * 1.2)
	total = farN + midN + nearN
	if total < 1 {
		farN, midN, nearN, total = 1, 0, 0, 1
		return
	}
	maxN := w * st.Height / 2
	if maxN < 1 {
		maxN = 1
	}
	if total > maxN {
		scale := float64(maxN) / float64(total)
		farN = int(float64(farN) * scale)
		midN = int(float64(midN) * scale)
		nearN = int(float64(nearN) * scale)
		total = farN + midN + nearN
		if total < 1 {
			farN, total = 1, 1
		}
	}
	return
}

func (st *State) initParticles(intensity float64) {
	if st.Width <= 0 || st.Height <= 0 {
		st.Particles = nil
		return
	}
	farN, midN, nearN, total := st.countsFor(intensity)
	particles := make([]Particle, 0, total)
	for _, n := range []struct {
		count int
		plane int
	}{{farN, PlaneFar}, {midN, PlaneMid}, {nearN, PlaneNear}} {
		for i := 0; i < n.count; i++ {
			p := particleForPlane(n.plane)
			p.X = float64(rand.Intn(st.Width))
			p.Y = float64(rand.Intn(st.Height))
			particles = append(particles, p)
		}
	}
	st.Particles = particles
}

func (st *State) targetParticleCount(intensity float64) int {
	if st.Width <= 0 {
		return 0
	}
	_, _, _, total := st.countsFor(intensity)
	return total
}

func (st *State) initMeteors() {
	if st.Width <= 0 || st.Height <= 0 {
		st.Meteors = nil
		st.Stars = nil
		st.Sparks = nil
		return
	}
	nStars := (st.Width * st.Height) / 80
	if nStars < 50 {
		nStars = 50
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
	st.MeteorFireballTimer = 400 + rand.Intn(300)
	st.BolideFlash = 0
	st.BolideChance = 0.18
	// the radiant hangs high in the sky and slowly wanders
	ry := st.City.HorizonY / 7
	if ry < 3 {
		ry = 3
	}
	if ry >= st.City.HorizonY-4 {
		ry = st.City.HorizonY - 4
	}
	if ry < 1 {
		ry = 1
	}
	st.RadiantY = float64(ry)
	st.RadiantX = float64(st.Width)*0.3 + rand.Float64()*float64(st.Width)*0.4
	st.RadiantDrift = 0.0006 + rand.Float64()*0.0009
	if rand.Float64() < 0.5 {
		st.RadiantDrift = -st.RadiantDrift
	}
}

func (st *State) Resize(w, h int) {
	st.Width = w
	st.Height = h
	if w > 0 {
		st.AccumRow = make([]int, w)
	} else {
		st.AccumRow = nil
	}
	switch st.Mode {
	case ModeThunderstorm:
		st.initParticles(st.StormIntensity)
	case ModeRain:
		st.initParticles(st.Intensity)
	case ModeSnow:
		st.initSnow()
	case ModeMeteor:
		st.initMeteors()
	}
	if st.Seed != 0 {
		st.City = NewSkyline(st.Seed, w, h)
		if st.World == WorldCoast {
			st.initCoast(w, h)
		}
		st.initClouds()
		st.updateRooftops()
	}
}

func rainStyle(color tcell.Color, dim bool) tcell.Style {
	s := tcell.StyleDefault.Foreground(color).Background(tcell.ColorReset)
	if dim {
		s = s.Attributes(tcell.AttrDim)
	}
	return s
}

func meteorStyle(color tcell.Color, dim, bold bool) tcell.Style {
	s := tcell.StyleDefault.Foreground(color).Background(tcell.ColorReset)
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

// updateWind eases the current wind toward its target. With live weather,
// snapping to the OLD target would fight the gust system, so it eases
// toward WindTargetLive instead. Without it, the target wanders on a timer
// like before.
func (st *State) updateWind() {
	if st.WindLive {
		diff := st.WindTargetLive - st.Wind
		st.Wind += math.Copysign(0.002, diff)
		if st.Wind >= st.WindTargetLive-0.002 && st.Wind <= st.WindTargetLive+0.002 {
			st.Wind = st.WindTargetLive
		}
		return
	}
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
			rangeScale := 1.0
			if st.FocusMode {
				rangeScale = 0.75
			}
			st.WindTarget = (-0.9 + rand.Float64()*1.8) * rangeScale
		} else {
			rangeScale := 1.0
			if st.FocusMode {
				rangeScale = 0.75
			}
			st.WindTarget = (-0.6 + rand.Float64()*1.2) * rangeScale
		}
	}
}

// updateGust advances the rolling transverse gust band. The new band spawns
// on the upwind edge and crosses the frame, so the whole field leans together
// as it passes and relaxes behind it.
func (st *State) updateGust() {
	st.GustTimer--
	if st.GustTimer <= 0 {
		st.GustStrength = 0.5 + rand.Float64()*0.5
		st.GustWidth = float64(st.Width) * (0.12 + rand.Float64()*0.18)
		if st.GustWidth < 2 {
			st.GustWidth = 2
		}
		if st.Wind >= 0 {
			st.GustDir = 1
			st.GustX = -st.GustWidth
		} else {
			st.GustDir = -1
			st.GustX = float64(st.Width) + st.GustWidth
		}
		st.GustTimer = 300 + rand.Intn(300)
		return
	}
	st.GustX += st.GustDir * 1.3
}

func (st *State) updateRain() {
	if st.Width <= 0 || st.Height <= 0 {
		return
	}
	st.updateWind()
	st.updateGust()
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
		_, _, _, windFactor, _, _ := planeParams(p.Plane)
		// gust is a gaussian bump centered on the band, stronger in storms
		dx := (p.X - st.GustX) / st.GustWidth
		gust := st.GustStrength * math.Exp(-dx*dx*2) * (0.4 + st.StormIntensity*0.6)
		p.VX = (st.Wind + gust) * (0.95 + p.Weight*0.35) * windFactor
		p.X += p.VX * mult
		p.Y += p.VY * mult

		if p.X < 0 {
			p.X += w
		}
		if p.X >= w {
			p.X -= w
		}

		if roof := st.roofAt(int(p.X)); roof >= 0 && p.Y >= float64(roof) {
			st.splashAt(p, roof)
			continue
		}
		if p.Y >= float64(st.Height) {
			st.splashAt(p, st.Height-1)
		}
	}
}

func (st *State) roofAt(x int) int {
	if st.RoofTop != nil && x >= 0 && x < len(st.RoofTop) {
		return st.RoofTop[x]
	}
	return -1
}

// splashAt resolves a drop landing. Far drops wink out silently, mid drops
// leave a faint dot, near drops throw the visible ripple.
func (st *State) splashAt(p *Particle, y int) {
	switch p.Plane {
	case PlaneFar:
		p.respawn(st)
		return
	case PlaneMid:
		p.Splash = 1
		p.SplashChar = '·'
	default:
		p.Splash = 2
		if p.SplashChar == 0 {
			p.SplashChar = '~'
		} else if p.SplashChar == '~' {
			p.SplashChar = '∿'
		} else {
			p.SplashChar = '~'
		}
	}
	p.SplashY = y
	p.respawn(st)
}

// respawn recycles the drop at the top of the screen with fresh parameters
// for its plane, keeping depth zones stable.
func (p *Particle) respawn(st *State) {
	p.Y = 0
	p.X = float64(rand.Intn(st.Width))
	np := particleForPlane(p.Plane)
	p.Weight = np.Weight
	p.VY = np.VY
	p.Len = np.Len
	p.Char = np.Char
}

func rainGlyphForWind(vx float64) rune {
	avx := math.Abs(vx)
	switch {
	case avx < 0.18:
		return '|'
	case avx < 0.45:
		if vx >= 0 {
			return '⟍'
		}
		return '⟋'
	default:
		if vx >= 0 {
			return '\\'
		}
		return '/'
	}
}

func (st *State) drawRain(screen tcell.Screen) {
	w, h := st.Width, st.Height
	if w == 0 || h == 0 {
		return
	}
	style := rainStyle(st.Color, false)
	dimStyle := rainStyle(st.Color, true)

	for i := range st.Particles {
		if st.FocusMode && i%3 == 0 {
			continue
		}
		p := &st.Particles[i]
		if !p.Active {
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
		// near drops render bright, mid and far drops recede into dim
		hStyle := style
		tStyle := dimStyle
		if p.Plane == PlaneNear {
			tStyle = rainStyle(st.Color, false)
		}
		headChar := rainGlyphForWind(p.VX)
		for j := 1; j <= p.Len; j++ {
			tx := p.X - p.VX*float64(j)*1.2
			ty := p.Y - p.VY*float64(j)*0.95
			sx := int(math.Round(tx))
			sy := int(math.Round(ty))
			if sy < 0 {
				break
			}
			sx = sx % w
			if sx < 0 {
				sx += w
			}
			if sx >= 0 && sx < w {
				screen.SetContent(sx, sy, headChar, nil, tStyle)
			}
		}
		if headX >= 0 && headX < w {
			screen.SetContent(headX, headY, headChar, nil, hStyle)
		}
	}
}

// DrawSplashes renders puddle ripples on top of the ground. It runs after
// DrawCity so splashes stay visible in rain and thunder modes.
func DrawSplashes(screen tcell.Screen, st *State) {
	w, h := st.Width, st.Height
	if w == 0 || h == 0 {
		return
	}
	style := rainStyle(st.Color, false)
	bottom := h - 1

	for i := range st.Particles {
		if st.FocusMode && i%3 == 0 {
			continue
		}
		p := &st.Particles[i]
		if !p.Active {
			continue
		}
		if p.Plane == PlaneFar {
			continue
		}
		if p.Splash > 0 {
			sx := int(p.X) % w
			if sx < 0 {
				sx += w
			}
			sy := p.SplashY
			if sy < 0 || sy >= h {
				sy = bottom
			}
			if sx >= 0 && sx < w {
				spStyle := style
				if p.SplashChar == '·' {
					spStyle = rainStyle(st.Color, true)
				}
				screen.SetContent(sx, sy, p.SplashChar, nil, spStyle)
			}
			continue
		}
		if int(p.Y) >= bottom-1 {
			sx := int(p.X) % w
			if sx < 0 {
				sx += w
			}
			if sx >= 0 && sx < w {
				screen.SetContent(sx, bottom, '~', nil, style)
			}
		}
	}
}

func Draw(screen tcell.Screen, st *State) {
	st.updateRain()
	st.drawRain(screen)
}

func (st *State) InitRain() {
	st.StormIntensity = 0
	st.Wind = 0
	st.WindTarget = 0
	st.WindTick = 0
	st.GustTimer = 120 + rand.Intn(120)
	st.Bolts = nil
	st.initParticles(st.Intensity)
}

// InitCalm clears the weather particles so the world alone is the scene.
func (st *State) InitCalm() {
	st.Particles = nil
	st.Bolts = nil
	st.Wind = 0
	st.WindTarget = 0
	st.WindTick = 0
	st.GustTimer = 0
}
