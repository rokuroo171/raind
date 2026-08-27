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

	target := st.targetParticleCount(st.StormIntensity)
	if target > 0 {
		cur := len(st.Particles)
		lo := target * 8 / 10
		hi := target*12/10 + 1
		if cur < lo || cur > hi {
			st.initParticles(st.StormIntensity)
		}
	}

	st.updateStormCell()
	st.updateCadence()

	// the sky flash is a soft dim shimmer, not a solid sheet, so the bolt
	// stays visible against it instead of washing out in white-on-white
	if st.StormFlash > 0 {
		flashRows := st.StormFlash
		if st.Height < flashRows {
			flashRows = st.Height
		}
		flashStyle := tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorReset).Attributes(tcell.AttrDim)
		for y := 0; y < flashRows; y++ {
			for x := 0; x < st.Width; x++ {
				screen.SetContent(x, y, '░', nil, flashStyle)
			}
		}
		st.StormFlash--
	}

	// the cloud flash: no bolt, just a diffuse bloom lighting the cloud
	// interior at the cell position for a moment
	if st.CloudGlow > 0 {
		glow := tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorReset).Attributes(tcell.AttrDim)
		cx := st.CloudGlowX
		for dy := 1; dy <= 3; dy++ {
			for dx := -3 + dy; dx <= 3-dy; dx++ {
				x, y := cx+dx, dy
				if x >= 0 && x < st.Width && y < st.Height {
					screen.SetContent(x, y, '░', nil, glow)
				}
			}
		}
		st.CloudGlow--
	}

	st.drawLightning(screen)
	st.updateRain()
	st.drawThunderDepthLayer(screen)
	st.drawRain(screen)
}

// updateStormCell drifts the storm position across the sky and eases it
// closer then farther, so distant bolts start low and overhead bolts hammer
// the frame.
func (st *State) updateStormCell() {
	st.CellPos += st.CellDrift
	if st.CellPos < 0.05 {
		st.CellPos = 0.05
		st.CellDrift = -st.CellDrift
	}
	if st.CellPos > 0.95 {
		st.CellPos = 0.95
		st.CellDrift = -st.CellDrift
	}
	st.CellDist += st.CellApproach
	if st.CellDist < 0.15 {
		st.CellDist = 0.15
		st.CellApproach = -st.CellApproach
	}
	if st.CellDist > 0.85 {
		st.CellDist = 0.85
		st.CellApproach = -st.CellApproach
	}
}

// updateCadence drives the burst-and-lull rhythm. The storm sleeps through a
// lull, then fires 2 to 4 strikes close together, then sleeps again.
// Intensity shortens the lull and lengthens the burst.
func (st *State) updateCadence() {
	if st.LullTimer > 0 {
		st.LullTimer--
		return
	}
	if st.BurstLeft == 0 {
		// a lull just ended: arm a new burst of 2 to 4 strikes
		st.BurstLeft = 2 + rand.Intn(3)
		st.BurstTimer = 6 + rand.Intn(10)
		return
	}
	st.BurstTimer--
	if st.BurstTimer <= 0 {
		st.spawnLightning()
		st.BurstLeft--
		if st.BurstLeft == 0 {
			// the burst is over. Most of the time the storm goes quiet for an
			// irregular, jittered gap; sometimes a second burst follows quickly,
			// like a real storm's double flash.
			if rand.Float64() < st.BurstDoubleChance {
				st.BurstLeft = 2 + rand.Intn(2)
				st.BurstTimer = 4 + rand.Intn(8)
				return
			}
			base := float64(250 + rand.Intn(450))
			lull := int(base * (1.4 - st.StormIntensity*0.7) * (0.75 + rand.Float64()*0.5))
			if lull < 30 {
				lull = 30
			}
			st.LullTimer = lull
		} else {
			st.BurstTimer = 6 + rand.Intn(10)
		}
	}
}

// boltTarget picks where a strike goes. Close cells aim at the tallest
// structure, the lighthouse on the coast, the hero tower in the city.
// Distant or random cells strike open water and sky near the cell position.
func (st *State) boltTarget() (aim, stop int) {
	aim, stop = -1, -1
	if st.CellDist < 0.5 || rand.Float64() < 0.35 {
		if st.World == WorldCoast {
			aim = st.Coast.LighthouseX
			stop = st.City.HorizonY - 1 - st.Coast.LighthouseH
			if stop < 2 {
				stop = st.City.HorizonY - 1
			}
		} else if len(st.City.Buildings) > 0 && st.City.Tallest >= 0 {
			b := st.City.Buildings[st.City.Tallest]
			aim = b.X + b.W/2
			stop = st.City.HorizonY - 1 - b.H
		}
	}
	if aim < 0 {
		w := st.Width
		cx := int(st.CellPos*float64(w)) + rand.Intn(w/8) - w/16
		if cx < 0 {
			cx = 0
		}
		if cx >= w {
			cx = w - 1
		}
		aim = cx
		stop = st.City.HorizonY - 1
	}
	return aim, stop
}

func (st *State) spawnLightning() {
	w, h := st.Width, st.Height
	if w <= 0 || h <= 2 {
		return
	}
	// roughly a third of storm events are cloud flashes: the stroke lights
	// the cloud from inside and never reaches ground
	if rand.Float64() < 0.33 {
		st.cloudFlash()
		return
	}
	power := (1 - st.CellDist) * (0.5 + st.StormIntensity*0.5)
	nBolts := 1
	if st.StormIntensity > 0.55 && rand.Float64() < 0.40 {
		nBolts = 2 + rand.Intn(2)
	}
	aim, stop := st.boltTarget()
	startY := 1 + int(float64(st.City.HorizonY)*st.CellDist*7/10)
	// the stepped leader is only visible up close, a distant flash just
	// pops on with no probe
	leader := st.CellDist < 0.6
	for b := 0; b < nBolts; b++ {
		a, s := aim, stop
		if rand.Float64() < 0.25 {
			a, s = -1, -1
		}
		st.Bolts = append(st.Bolts, st.generateBolt(w, h, a, s, startY, power, leader))
	}
}

// cloudFlash lights the cloud interior without a ground bolt: a brief
// bloom at the cell position and a soft sky flash, counted in the burst.
func (st *State) cloudFlash() {
	w := st.Width
	if w <= 0 {
		return
	}
	cx := int(st.CellPos*float64(w)) + rand.Intn(w/8) - w/16
	if cx < 0 {
		cx = 0
	}
	if cx >= w {
		cx = w - 1
	}
	st.CloudGlow = 5 + rand.Intn(5)
	st.CloudGlowX = cx
	power := (1 - st.CellDist) * (0.5 + st.StormIntensity*0.5)
	st.StormFlash = max(st.StormFlash, 2+int(power*3))
}

// generateBolt builds the full channel of one strike: the jittered main
// trunk plus 2 to 3 side branches, with enough strokes to re-fire the same
// channel a few times for the flicker of a real flash.
func (st *State) generateBolt(w, h, aimX, stopY, startY int, power float64, leader bool) BoltStrike {
	if aimX < 0 {
		aimX = rand.Intn(w)
	}
	if stopY < 2 || stopY > h-1 {
		stopY = h - 1
		if st.City.HorizonY > 1 {
			stopY = st.City.HorizonY - 1
		}
	}
	if startY < 0 {
		startY = 0
	}
	if startY >= stopY {
		startY = 0
	}
	x, y := aimX, startY
	dir := 0
	var points []BoltPoint
	for y < stopY {
		ch := '|'
		if len(points) > 0 {
			prev := points[len(points)-1]
			dx := x - prev.X
			dy := y - prev.Y
			switch {
			case dx < 0 && dy < 0:
				ch = '└'
			case dx < 0 && dy == 0:
				ch = '┘'
			case dx < 0 && dy > 0:
				ch = '┐'
			case dx == 0 && dy < 0:
				ch = '/'
			case dx == 0 && dy > 0:
				ch = '\\'
			case dx > 0 && dy < 0:
				ch = '┌'
			case dx > 0 && dy == 0:
				ch = '┐'
			case dx > 0 && dy > 0:
				ch = '┘'
			}
		}
		points = append(points, BoltPoint{X: x, Y: y, Char: ch})

		// pull the bolt back toward the target column, then random-walk
		dx := aimX - x
		roll := rand.Float64()
		nextDir := dir
		if dx > 1 && roll < 0.5 {
			nextDir = 1
		} else if dx < -1 && roll >= 0.5 {
			nextDir = -1
		} else if roll < 0.2 {
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

	var branches [][]BoltPoint
	if len(points) > 4 {
		nBr := 2 + rand.Intn(2)
		for k := 0; k < nBr; k++ {
			// branches probe for ground from the lower half of the channel,
			// where the leader's field is strongest
			lo := len(points) * 2 / 5
			hi := len(points) - 3
			if hi <= lo {
				lo = len(points) / 2
				hi = len(points) - 1
			}
			startIdx := lo + rand.Intn(hi-lo+1)
			if startIdx >= len(points) {
				continue
			}
			side := 1
			if rand.Float64() < 0.5 {
				side = -1
			}
			bx := points[startIdx].X
			by := points[startIdx].Y
			var br []BoltPoint
			for s := 0; s < 4+rand.Intn(3); s++ {
				bx += side * (1 + rand.Intn(2))
				by += 1 + rand.Intn(2)
				if by >= stopY || bx < 0 || bx >= w {
					break
				}
				ch := '\\'
				if side < 0 {
					ch = '/'
				}
				br = append(br, BoltPoint{X: bx, Y: by, Char: ch})
			}
			if len(br) > 0 {
				branches = append(branches, br)
			}
		}
	}

	bolt := BoltStrike{
		Channel:    BoltChannel{Points: points, Branches: branches},
		FramesLeft: 2,
		Strokes:    2 + rand.Intn(3),
		HaloFrame:  true,
		Power:      power,
	}
	if leader && len(points) > 4 {
		// the channel ionizes slowly: the leader reveals itself in steps
		bolt.LeaderFrames = 12 + rand.Intn(8)
		bolt.LeaderLen = 2
	}
	return bolt
}

func (st *State) drawLightning(screen tcell.Screen) {
	w, h := st.Width, st.Height
	if w == 0 || h == 0 {
		return
	}
	haloStyle := tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorReset).Attributes(tcell.AttrDim)
	var remaining []BoltStrike
	for i := range st.Bolts {
		b := &st.Bolts[i]
		if b.LeaderFrames > 0 {
			// the stepped leader probes downward in jumps, twitching, then
			// the return stroke blasts up the fully ionized channel
			b.LeaderLen += 1 + rand.Intn(2)
			if b.LeaderLen >= len(b.Channel.Points) {
				b.LeaderFrames = 0
				b.StrokeNum = 1
				b.FramesLeft = 2
				b.HaloFrame = true
			} else {
				b.LeaderFrames--
				// flicker: the probe blinks out for a beat as it finds ground
				if (st.Frame+b.LeaderFrames)%3 != 0 {
					drawLeader(screen, w, h, b)
				}
			}
			remaining = append(remaining, *b)
			continue
		}
		if b.FramesLeft > 0 {
			if b.HaloFrame {
				st.StormFlash = max(st.StormFlash, 1+int(b.Power*2))
				haloW := 1 + int(b.Power*2)
				for _, pt := range b.Channel.Points {
					for dx := -haloW; dx <= haloW; dx++ {
						if dx == 0 {
							continue
						}
						hx := pt.X + dx
						if hx >= 0 && hx < w && pt.Y >= 0 && pt.Y < h {
							screen.SetContent(hx, pt.Y, '░', nil, haloStyle)
						}
					}
				}
				b.HaloFrame = false
			}
			b.drawBolt(screen, w, h)
			b.FramesLeft--
			if b.FramesLeft == 0 && b.Strokes > 1 {
				// return strokes re-fire fast: 1-2 frames, ~28-56ms, the real flicker
				b.StrokeGap = 1 + rand.Intn(2)
			}
			remaining = append(remaining, *b)
			continue
		}
		if b.StrokeGap > 0 {
			b.StrokeGap--
			if b.StrokeGap == 0 {
				b.Strokes--
				b.StrokeNum++
				b.FramesLeft = 2
				b.HaloFrame = true
			}
			remaining = append(remaining, *b)
			continue
		}
	}
	st.Bolts = remaining
}

// drawLeader renders the faint probe as it descends: the trunk and branches
// revealed so far, dim and twitchy, the quiet act before the flash.
func drawLeader(screen tcell.Screen, w, h int, b *BoltStrike) {
	style := tcell.StyleDefault.Foreground(tcell.NewHexColor(0x9fb2cc)).Background(tcell.ColorReset).Attributes(tcell.AttrDim)
	n := b.LeaderLen
	if n > len(b.Channel.Points) {
		n = len(b.Channel.Points)
	}
	for _, pt := range b.Channel.Points[:n] {
		if pt.X >= 0 && pt.X < w && pt.Y >= 0 && pt.Y < h {
			screen.SetContent(pt.X, pt.Y, pt.Char, nil, style)
		}
	}
	// a branch appears once the probe has passed its origin
	deepY := b.Channel.Points[n-1].Y
	for _, br := range b.Channel.Branches {
		if len(br) == 0 || br[0].Y > deepY {
			continue
		}
		for _, pt := range br {
			if pt.X >= 0 && pt.X < w && pt.Y >= 0 && pt.Y < h {
				screen.SetContent(pt.X, pt.Y, pt.Char, nil, style)
			}
		}
	}
}

func (b *BoltStrike) drawBolt(screen tcell.Screen, w, h int) {
	boltStyle := tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorReset).Attributes(tcell.AttrBold)
	for _, pt := range b.Channel.Points {
		if pt.X >= 0 && pt.X < w && pt.Y >= 0 && pt.Y < h {
			screen.SetContent(pt.X, pt.Y, pt.Char, nil, boltStyle)
		}
	}
	// branches belong to the first stroke only, re-strikes race down the
	// clean ionized channel
	if b.StrokeNum <= 1 {
		for _, br := range b.Channel.Branches {
			for _, pt := range br {
				if pt.X >= 0 && pt.X < w && pt.Y >= 0 && pt.Y < h {
					screen.SetContent(pt.X, pt.Y, pt.Char, nil, boltStyle)
				}
			}
		}
	}
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

func (st *State) InitThunderstorm() {
	st.StormIntensity = 0.1 + st.Intensity*0.9
	st.Wind = 0
	st.WindTarget = -0.9 + rand.Float64()*1.8
	st.WindTick = 0
	st.GustTimer = 60 + rand.Intn(120)
	st.Bolts = nil
	st.StormFlash = 0
	st.CloudGlow = 0
	st.CellPos = 0.2 + rand.Float64()*0.6
	st.CellDist = 0.55 + rand.Float64()*0.2
	st.CellDrift = 0.0004 + rand.Float64()*0.0008
	if rand.Float64() < 0.5 {
		st.CellDrift = -st.CellDrift
	}
	st.CellApproach = -(0.0002 + rand.Float64()*0.0005)
	st.BurstLeft = 0
	st.BurstTimer = 0
	st.LullTimer = 40 + rand.Intn(80)
	st.BurstDoubleChance = 0.25
	st.initParticles(st.StormIntensity)
}
