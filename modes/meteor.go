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

func (st *State) spawnMeteor(originX, originY float64, spread bool) {
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

	x := originX
	y := originY
	if originX < 0 {
		x = float64(rand.Intn(w))
		y = -float64(rand.Intn(3)) - 1
	}

	dir := -1.0
	if rand.Float64() < 0.5 {
		dir = 1.0
	}
	if spread {
		dir = -1.0 + rand.Float64()*2.0
	}

	speed := 0.55 + rand.Float64()*0.75
	vx := dir * speed * (0.5 + rand.Float64()*0.9)
	vy := speed * (0.85 + rand.Float64()*0.55)
	vx += st.Wind * 0.15

	life := 18 + rand.Intn(28)
	trail := 6 + rand.Intn(8)

	st.Meteors[slot] = Meteor{
		X:        x,
		Y:        y,
		VX:       vx,
		VY:       vy,
		Life:     life,
		MaxLife:  life,
		TrailLen: trail,
		HeadChar: meteorHeadChars[rand.Intn(len(meteorHeadChars))],
		Active:   true,
	}
}

func (st *State) spawnMeteorShower() {
	w := st.Width
	if w <= 0 {
		return
	}
	ox := float64(w/2) + (rand.Float64()-0.5)*float64(w)/3
	oy := -2.0
	n := 4 + rand.Intn(5)
	for i := 0; i < n; i++ {
		st.spawnMeteor(ox+rand.Float64()*6-3, oy-rand.Float64()*2, true)
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
	y := rand.Float64() * float64(h/3+1)

	st.Meteors[slot] = Meteor{
		X:        x,
		Y:        y,
		VX:       vx,
		VY:       0.1 + rand.Float64()*0.2,
		Life:     w + 20,
		MaxLife:  w + 20,
		TrailLen: 18 + rand.Intn(5),
		HeadChar: '◉',
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

func DrawMeteor(screen tcell.Screen, st *State) {
	w, h := st.Width, st.Height
	if w == 0 || h == 0 {
		return
	}

	st.updateWind()
	mult := st.Speed.SpeedMultiplier()

	drawStarfield(screen, st)

	st.MeteorShowerTimer--
	if st.MeteorShowerTimer <= 0 {
		st.spawnMeteorShower()
		st.MeteorShowerTimer = 200 + rand.Intn(250)
	} else if rand.Float64() < 0.035 {
		st.spawnMeteor(-1, -1, false)
	} else if rand.Float64() < 0.008 {
		st.spawnMeteor(float64(rand.Intn(w)), -1, false)
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

		off := headX < -4 || headX > w+4 || headY < -4 || headY > h+4
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
		if sx >= 0 && sx < w && sy >= 0 && sy < h {
			screen.SetContent(sx, sy, sp.Char, nil, sparkStyle)
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
