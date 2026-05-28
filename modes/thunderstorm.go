package modes

import (
	"math"
	"math/rand"

	"github.com/gdamore/tcell/v2"
)

func DrawThunderstorm(screen tcell.Screen, st *State) {
	if st.StormIntensity < 1.0 {
		st.StormIntensity += 0.0008
		if st.StormIntensity > 1.0 {
			st.StormIntensity = 1.0
		}
	}

	density := 0.35 + st.StormIntensity*0.45
	target := st.targetParticleCount(density)
	if target > 0 {
		cur := len(st.Particles)
		lo := target * 8 / 10
		hi := target*12/10 + 1
		if cur < lo || cur > hi {
			st.initParticles(density)
		}
	}

	if st.StormIntensity > 0.3 {
		chance := 0.004 + st.StormIntensity*0.008
		if rand.Float64() < chance {
			st.spawnLightning()
		}
	}

	if st.StormFlash > 0 {
		flashRows := 2
		if st.Height < flashRows {
			flashRows = st.Height
		}
		flashStyle := tcell.StyleDefault.Background(tcell.ColorWhite)
		for y := 0; y < flashRows; y++ {
			for x := 0; x < st.Width; x++ {
				screen.SetContent(x, y, ' ', nil, flashStyle)
			}
		}
		st.StormFlash--
	}

	st.drawLightning(screen)
	st.updateRain()
	st.drawThunderDepthLayer(screen)
	st.drawRain(screen, true)
}

func (st *State) drawThunderDepthLayer(screen tcell.Screen) {
	w, h := st.Width, st.Height
	if w == 0 || h == 0 {
		return
	}
	depthStyle := rainStyle(st.Color, true)
	for i := range st.Particles {
		if i%2 == 1 {
			continue
		}
		p := &st.Particles[i]
		if !p.Active || p.Splash > 0 {
			continue
		}
		headX := int(math.Round(p.X - p.VX*1.4))
		headY := int(math.Round(p.Y - p.VY*0.7))
		if headY < 0 || headY >= h {
			continue
		}
		headX = headX % w
		if headX < 0 {
			headX += w
		}
		ch := rainGlyphForWind(p.VX * 0.75)
		screen.SetContent(headX, headY, ch, nil, depthStyle)
	}
}

func (st *State) spawnLightning() {
	w, h := st.Width, st.Height
	if w <= 0 || h <= 2 {
		return
	}
	nBolts := 1
	r := rand.Float64()
	if r < 0.15 {
		nBolts = 3
	} else if r < 0.50 {
		nBolts = 2
	}
	frames := 3 + rand.Intn(3)
	for b := 0; b < nBolts; b++ {
		st.Bolts = append(st.Bolts, st.generateBolt(w, h, frames))
	}
	st.StormFlash = frames
}

func (st *State) generateBolt(w, h, frames int) BoltStrike {
	x := rand.Intn(w)
	y := 0
	dir := 0 // -1 left, 0 down, 1 right
	var points []BoltPoint
	chars := map[[2]int]rune{
		{-1, -1}: '└', {-1, 0}: '┘', {-1, 1}: '┐',
		{0, -1}: '/', {0, 1}: '\\',
		{1, -1}: '┌', {1, 0}: '┐', {1, 1}: '┘',
	}
	for y < h-1 {
		ch := '|'
		if len(points) > 0 {
			prev := points[len(points)-1]
			dx := x - prev.X
			dy := y - prev.Y
			if c, ok := chars[[2]int{dx, dy}]; ok {
				ch = c
			} else if dx < 0 {
				ch = '/'
			} else if dx > 0 {
				ch = '\\'
			}
		}
		points = append(points, BoltPoint{X: x, Y: y, Char: ch})

		// weighted step: continue direction more often
		roll := rand.Float64()
		nextDir := dir
		if roll < 0.2 {
			nextDir = -1
		} else if roll < 0.4 {
			nextDir = 1
		} else if roll < 0.55 {
			nextDir = 0
		}
		dir = nextDir
		y++
		switch dir {
		case -1:
			x--
		case 1:
			x++
		}
		if x < 0 {
			x = 0
		}
		if x >= w {
			x = w - 1
		}
	}
	return BoltStrike{Points: points, FramesLeft: frames, HaloFrame: true}
}

func (st *State) drawLightning(screen tcell.Screen) {
	w, h := st.Width, st.Height
	if w == 0 || h == 0 {
		return
	}
	boltStyle := tcell.StyleDefault.
		Foreground(tcell.ColorWhite).
		Background(tcell.ColorBlack).
		Attributes(tcell.AttrBold)
	haloStyle := tcell.StyleDefault.
		Foreground(tcell.ColorBlack).
		Background(tcell.ColorWhite)

	var remaining []BoltStrike
	for i := range st.Bolts {
		b := &st.Bolts[i]
		if b.FramesLeft <= 0 {
			continue
		}
		if b.HaloFrame {
			st.StormFlash = 4
			for _, pt := range b.Points {
				for dx := -3; dx <= 3; dx++ {
					if dx == 0 {
						continue
					}
					hx := pt.X + dx
					if hx >= 0 && hx < w && pt.Y >= 0 && pt.Y < h {
						screen.SetContent(hx, pt.Y, ' ', nil, haloStyle)
					}
				}
			}
			b.HaloFrame = false
		} else {
			for _, pt := range b.Points {
				if pt.X >= 0 && pt.X < w && pt.Y >= 0 && pt.Y < h {
					screen.SetContent(pt.X, pt.Y, pt.Char, nil, boltStyle)
				}
			}
		}
		b.FramesLeft--
		if b.FramesLeft > 0 {
			remaining = append(remaining, *b)
		}
	}
	st.Bolts = remaining
}

func (st *State) InitThunderstorm() {
	st.StormIntensity = 0.1
	st.Wind = 0
	st.WindTarget = -0.9 + rand.Float64()*1.8
	st.WindTick = 0
	st.Bolts = nil
	st.StormFlash = 0
	st.initParticles(0.5)
}
