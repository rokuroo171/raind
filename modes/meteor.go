package modes

import (
	"math"
	"math/rand"

	"github.com/gdamore/tcell/v2"
)

var meteorHeadChars = []rune{'✦', '★', '✧', '*', '◆', '⊹'}
var meteorTrailChars = []rune{'█', '▓', '▒', '░', '·', '∘', '′', ':'}
var sparkChars = []rune{'·', '∙', '+', '×', '✦', '*'}

func (st *State) InitMeteor() {
	st.Wind = 0
	st.WindTarget = 0
	st.WindTick = 0
	st.MeteorFlash = 0
	st.initMeteors()
}

// spawnRadiantMeteor fires one meteor from the radiant. Its velocity points
// from the radiant toward a random point in the lower sky, so the trail
// always points back at the radiant, which is what makes a shower read as
// deep space instead of random streaks.
func (st *State) spawnRadiantMeteor() {
	slot := -1
	for i := range st.Meteors {
		if !st.Meteors[i].Active {
			slot = i
			break
		}
	}
	if slot < 0 {
		return
	}

	w := st.Width
	if w <= 0 || st.City.HorizonY < 3 {
		return
	}

	// leave the radiant with a small jitter, aim at the sky below
	sx := st.RadiantX + (rand.Float64()-0.5)*3
	sy := st.RadiantY + (rand.Float64()-0.5)*2
	tx := st.RadiantX + (rand.Float64()-0.5)*float64(w)*1.4
	ty := st.RadiantY + (float64(st.City.HorizonY)-st.RadiantY)*(0.3+rand.Float64()*0.65)
	if tx < 0 {
		tx = 0
	}
	if tx >= float64(w) {
		tx = float64(w - 1)
	}
	if ty <= sy {
		ty = sy + 1
	}

	dx, dy := tx-sx, ty-sy
	lenv := math.Sqrt(dx*dx + dy*dy)
	if lenv == 0 {
		lenv = 1
	}
	speed := 0.55 + rand.Float64()*0.8

	life := 18 + rand.Intn(26)
	trail := 6 + rand.Intn(7)

	st.Meteors[slot] = Meteor{
		X:        sx,
		Y:        sy,
		VX:       dx / lenv * speed,
		VY:       dy / lenv * speed * (1.05 + rand.Float64()*0.3),
		Life:     life,
		MaxLife:  life,
		TrailLen: trail,
		HeadChar: meteorHeadChars[rand.Intn(len(meteorHeadChars))],
		Active:   true,
	}
}

func (st *State) spawnMeteorShower() {
	if st.Width <= 0 {
		return
	}
	// a burst of meteors emerging from the radiant together
	n := 4 + rand.Intn(5)
	for i := 0; i < n; i++ {
		st.spawnRadiantMeteor()
	}
	st.MeteorFlash = 3
}

func (st *State) spawnFireball() {
	slot := -1
	for i := range st.Meteors {
		if !st.Meteors[i].Active {
			slot = i
			break
		}
	}
	if slot < 0 {
		return
	}

	w, h := st.Width, st.Height
	if w <= 0 || h <= 0 {
		return
	}

	fromLeft := rand.Intn(2) == 0
	x := 0.0
	vx := 2.5 + rand.Float64()
	if !fromLeft {
		x = float64(w - 1)
		vx = -vx
	}
	// Fireballs fly across the upper sky and burn out before the horizon,
	// so they never streak through the buildings or onto the road.
	burnupY := st.City.HorizonY - 3
	y := rand.Float64() * float64(burnupY/2+1)
	if y >= float64(burnupY) {
		y = float64(burnupY - 1)
	}

	// a rare bolide: heavier trail, and the sky tints warm for a moment
	bolide := rand.Float64() < st.BolideChance
	trailLen := 18 + rand.Intn(5)
	head := '◉'
	if bolide {
		trailLen = 26 + rand.Intn(6)
		head = '☄'
		st.BolideFlash = 3
	}

	st.Meteors[slot] = Meteor{
		X:        x,
		Y:        y,
		VX:       vx,
		VY:       0.1 + rand.Float64()*0.2,
		Life:     w + 20,
		MaxLife:  w + 20,
		TrailLen: trailLen,
		HeadChar: head,
		Active:   true,
	}
}

func (st *State) spawnSparks(x, y float64) {
	n := 5 + rand.Intn(4)
	for k := 0; k < n; k++ {
		slot := -1
		for i := range st.Sparks {
			if !st.Sparks[i].Active {
				slot = i
				break
			}
		}
		if slot < 0 {
			return
		}
		angle := rand.Float64() * math.Pi * 2
		spd := 0.15 + rand.Float64()*0.35
		st.Sparks[slot] = Spark{
			X:      x,
			Y:      y,
			VX:     math.Cos(angle) * spd,
			VY:     math.Sin(angle) * spd,
			Life:   3 + rand.Intn(4),
			Char:   sparkChars[rand.Intn(len(sparkChars))],
			Active: true,
		}
	}
}

func drawStarfield(screen tcell.Screen, st *State) {
	w, h := st.Width, st.Height
	dim := meteorStyle(st.Color, true, false)
	bright := meteorStyle(st.Color, false, false)

	for i := range st.Stars {
		s := &st.Stars[i]
		s.Twinkle++
		style := dim
		if s.Twinkle%40 < 8 {
			style = bright
		}
		if s.X >= 0 && s.X < w && s.Y >= 0 && s.Y < h {
			screen.SetContent(s.X, s.Y, s.Char, nil, style)
		}
	}
}

func drawMeteorFlash(screen tcell.Screen, st *State) {
	if st.MeteorFlash <= 0 {
		return
	}
	w, h := st.Width, st.Height
	flash := meteorStyle(st.Color, true, false)
	for x := 0; x < w; x++ {
		screen.SetContent(x, 0, '‾', nil, flash)
		if h > 1 {
			screen.SetContent(x, h-1, '_', nil, flash)
		}
	}
}

// meteorVisibility scales the shower rate with the actual sky: an overcast
// or stormy sky hides most meteors, a clear sky shows them all, and night
// reads best. Offline simulation stays near full so the mode never lies.
func (st *State) meteorVisibility() float64 {
	v := 1.0
	if st.WeatherLive && !st.Weather.Offline {
		switch st.Weather.Condition {
		case CondCloudy:
			v = 0.12
		case CondRain, CondSnow, CondThunder:
			v = 0.2
		}
	}
	return v * (0.35 + st.Night*0.65)
}

// drawBolideFlash tints the whole sky dim warm for the frames of a bolide,
// the heavy meteor's signature.
func drawBolideFlash(screen tcell.Screen, st *State) {
	if st.BolideFlash <= 0 {
		return
	}
	w := st.Width
	tint := tcell.StyleDefault.Foreground(tcell.NewHexColor(0xffe9c9)).Background(tcell.ColorReset).Attributes(tcell.AttrDim)
	for y := 0; y < st.City.HorizonY && y < st.Height; y++ {
		for x := 0; x < w; x += 2 {
			screen.SetContent(x, y, '░', nil, tint)
		}
	}
	st.BolideFlash--
}

func DrawMeteor(screen tcell.Screen, st *State) {
	w, h := st.Width, st.Height
	if w == 0 || h == 0 {
		return
	}

	st.updateWind()
	mult := st.Speed.SpeedMultiplier()

	// the radiant wanders slowly, so the shower itself drifts over minutes
	st.RadiantX += st.RadiantDrift
	if st.RadiantX < float64(w)*0.15 {
		st.RadiantX = float64(w) * 0.15
		st.RadiantDrift = -st.RadiantDrift
	}
	if st.RadiantX > float64(w)*0.85 {
		st.RadiantX = float64(w) * 0.85
		st.RadiantDrift = -st.RadiantDrift
	}

	drawStarfield(screen, st)
	drawBolideFlash(screen, st)

	vis := st.meteorVisibility()
	st.MeteorShowerTimer--
	if st.MeteorShowerTimer <= 0 {
		if rand.Float64() < vis {
			st.spawnMeteorShower()
		}
		base := float64(200 + rand.Intn(250))
		st.MeteorShowerTimer = int(base / max(vis, 0.05))
	} else if rand.Float64() < 0.035*vis {
		st.spawnRadiantMeteor()
	}
	st.MeteorFireballTimer--
	if st.MeteorFireballTimer <= 0 {
		st.spawnFireball()
		st.MeteorFireballTimer = 400 + rand.Intn(300)
	}

	headStyle := meteorStyle(st.Color, false, true)
	midStyle := meteorStyle(st.Color, false, false)
	trailStyle := meteorStyle(st.Color, true, false)
	sparkStyle := meteorStyle(st.Color, false, true)

	for i := range st.Meteors {
		m := &st.Meteors[i]
		if !m.Active {
			continue
		}

		m.X += m.VX * mult
		m.Y += m.VY * mult
		m.Life--

		headX := int(math.Round(m.X))
		headY := int(math.Round(m.Y))

		// Meteors burn up in the atmosphere before reaching the skyline,
		// so they never streak into the city or the street.
		burnupY := st.City.HorizonY - 3
		if burnupY < 1 {
			burnupY = 1
		}
		if headY >= burnupY {
			if headX >= 0 && headX < w {
				st.spawnSparks(m.X, m.Y)
			}
			m.Active = false
			continue
		}

		for j := m.TrailLen; j >= 1; j-- {
			t := float64(j) / float64(m.TrailLen)
			tx := m.X - m.VX*float64(j)*0.92
			ty := m.Y - m.VY*float64(j)*0.92
			sx := int(math.Round(tx))
			sy := int(math.Round(ty))
			if sx < 0 || sx >= w || sy < 0 || sy >= h {
				continue
			}
			ch := meteorTrailChars[int(t*float64(len(meteorTrailChars)-1))]
			if ch == 0 {
				ch = '·'
			}
			style := trailStyle
			if j <= 2 {
				style = midStyle
				idx := j
				if idx >= len(meteorTrailChars) {
					idx = len(meteorTrailChars) - 1
				}
				ch = meteorTrailChars[idx]
			}
			if j == 1 {
				ch = '░'
				style = midStyle
			}
			screen.SetContent(sx, sy, ch, nil, style)
		}

		if headX >= 0 && headX < w && headY >= 0 && headY < h {
			screen.SetContent(headX, headY, m.HeadChar, nil, headStyle)
		}

		off := headX < -4 || headX > w+4 || headY < -4
		if m.Life <= 0 || off {
			if headX >= 0 && headX < w && headY >= 0 && headY < h {
				st.spawnSparks(m.X, m.Y)
			}
			m.Active = false
		}
	}

	for i := range st.Sparks {
		sp := &st.Sparks[i]
		if !sp.Active {
			continue
		}
		sx := int(math.Round(sp.X))
		sy := int(math.Round(sp.Y))
		// Keep the explosion above the skyline so sparks never rain into
		// the city or onto the road.
		if sy < st.City.HorizonY-3 {
			if sx >= 0 && sx < w && sy >= 0 && sy < h {
				screen.SetContent(sx, sy, sp.Char, nil, sparkStyle)
			}
		}
		sp.X += sp.VX * mult
		sp.Y += sp.VY * mult
		sp.Life--
		if sp.Life <= 0 {
			sp.Active = false
		}
	}

	drawMeteorFlash(screen, st)
	if st.MeteorFlash > 0 {
		st.MeteorFlash--
	}
}
