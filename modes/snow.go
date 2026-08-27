package modes

import (
	"math"
	"math/rand"

	"github.com/gdamore/tcell/v2"
)

func (st *State) InitSnow() {
	st.Wind = 0
	st.WindTarget = 0
	st.WindTick = 0
	st.GustTimer = 120 + rand.Intn(120)
	st.initSnow()
}

func (st *State) initSnow() {
	if st.Width <= 0 || st.Height <= 0 {
		st.Flakes = nil
		st.AccumRow = nil
		st.RoofAccum = nil
		return
	}
	st.AccumRow = make([]int, st.Width)
	st.RoofAccum = make([]int, st.Width)
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

// snowMaxAccum returns the ground bank height for the current intensity, so
// light snow leaves a thin dusting and a whiteout builds real banks.
func (st *State) snowMaxAccum(h int) int {
	m := int(float64(h) * 0.16 * (0.5 + st.Intensity*0.5))
	if m < 1 {
		m = 1
	}
	return m
}

// snowGroundDeposit layers snow on the ground. On the coast the only ground
// is the lighthouse rock, which catches a thin dusting; in the city the bank
// builds into neighbors for a windswept drift.
func (st *State) snowGroundDeposit(col, maxAccum, w int) {
	if w == 0 {
		return
	}
	if st.World == WorldCoast {
		if col < st.Coast.LighthouseX-2 || col > st.Coast.LighthouseX+2 {
			return
		}
		limit := 1 + int(st.Intensity*3)
		if limit > 4 {
			limit = 4
		}
		if st.AccumRow[col] < limit {
			st.AccumRow[col]++
		}
		return
	}
	best := col
	bestHeight := maxAccum + 1
	for dx := -1; dx <= 1; dx++ {
		c := col + dx
		if c < 0 || c >= w {
			continue
		}
		if st.AccumRow[c] < bestHeight {
			best = c
			bestHeight = st.AccumRow[c]
		}
	}
	if bestHeight < maxAccum {
		st.AccumRow[best]++
	}
}

// snowRoofDeposit layers snow on the roofed column with the least snow
// nearby, so city rooftops catch drifts in windswept shapes. Snow never
// lands on a column without a roof.
func (st *State) snowRoofDeposit(col, w int) {
	if st.World != WorldCity || w == 0 {
		return
	}
	limit := 2 + int(st.Intensity*3)
	if limit > 6 {
		limit = 6
	}
	best := col
	bestVal := limit + 1
	for dx := -1; dx <= 1; dx++ {
		c := col + dx
		if c < 0 || c >= w || st.roofAt(c) < 0 {
			continue
		}
		if st.RoofAccum[c] < bestVal {
			best = c
			bestVal = st.RoofAccum[c]
		}
	}
	if bestVal < limit {
		st.RoofAccum[best]++
	}
}

func DrawSnow(screen tcell.Screen, st *State) {
	w, h := st.Width, st.Height
	if w == 0 || h == 0 {
		return
	}
	st.updateWind()
	st.updateGust()
	mult := st.Speed.SpeedMultiplier()
	if st.FocusMode {
		mult *= 0.85
	}
	style := tcell.StyleDefault.Foreground(st.Color).Background(tcell.ColorReset)

	lightChars := []rune{'·', '∗', '❄', '✻'}
	heavyChars := []rune{'|', '•'}
	maxAccum := st.snowMaxAccum(h)

	if len(st.AccumRow) != w {
		st.AccumRow = make([]int, w)
	}
	if len(st.RoofAccum) != w {
		st.RoofAccum = make([]int, w)
	}

	drawAuroraSky(screen, st)

	resetFlake := func(f *Snowflake) {
		f.Y = 0
		f.X = float64(rand.Intn(w))
		wgt := 0.2 + rand.Float64()*0.8
		f.Weight = wgt
		f.Speed = 0.08 + wgt*0.18
		f.Drift = (rand.Float64() - 0.5) * (0.25 - wgt*0.15)
		if wgt > 0.7 {
			f.Char = heavyChars[rand.Intn(len(heavyChars))]
		} else if wgt < 0.4 {
			f.Char = lightChars[rand.Intn(len(lightChars))]
		} else {
			all := append(append([]rune{}, lightChars...), heavyChars...)
			f.Char = all[rand.Intn(len(all))]
		}
	}
	smoothAccum := func() {
		if w < 2 {
			return
		}
		for pass := 0; pass < 2; pass++ {
			for x := 1; x < w; x++ {
				if st.AccumRow[x]-st.AccumRow[x-1] > 1 {
					st.AccumRow[x]--
					st.AccumRow[x-1]++
				}
			}
			for x := w - 2; x >= 0; x-- {
				if st.AccumRow[x]-st.AccumRow[x+1] > 1 {
					st.AccumRow[x]--
					st.AccumRow[x+1]++
				}
			}
		}
		// crest cleanup: soften single-cell spikes and fill tiny pits
		for x := 1; x < w-1; x++ {
			left := st.AccumRow[x-1]
			cur := st.AccumRow[x]
			right := st.AccumRow[x+1]
			if left == right {
				if cur > left+1 {
					st.AccumRow[x] = left + 1
				}
				if cur+1 < left {
					st.AccumRow[x] = left - 1
				}
			}
		}
	}

	for i := range st.Flakes {
		f := &st.Flakes[i]
		if !f.Active {
			continue
		}
		x := int(f.X)
		y := int(f.Y)
		if x >= 0 && x < w && y >= 0 && y < h {
			screen.SetContent(x, y, f.Char, nil, style)
		}
		f.Y += f.Speed * mult
		// rolling gust leans the whole field together as the band passes,
		// the same system rain uses
		gdx := (f.X - st.GustX) / st.GustWidth
		gust := st.GustStrength * math.Exp(-gdx*gdx*2) * (0.5 + st.Intensity*0.5)
		f.X += f.Drift*mult + (st.Wind+gust)*(1.0-f.Weight)*0.3*mult
		if int(f.X) < 0 {
			f.X = float64(w - 1)
		}
		if int(f.X) >= w {
			f.X = 0
		}
		col := int(f.X)
		if col < 0 {
			col = 0
		}
		if col >= w {
			col = w - 1
		}
		accum := st.AccumRow[col]
		if accum > maxAccum {
			accum = maxAccum
			st.AccumRow[col] = accum
		}
		// city rooftops catch snow before the flake can fall past them
		if st.World == WorldCity {
			if roof := st.roofAt(col); roof >= 0 && int(f.Y) >= roof {
				st.snowRoofDeposit(col, w)
				resetFlake(f)
				continue
			}
		}
		if int(f.Y) >= h {
			st.snowGroundDeposit(col, maxAccum, w)
			resetFlake(f)
			continue
		}
		if accum > 0 {
			topAccumY := h - accum
			if int(f.Y) >= topAccumY {
				st.snowGroundDeposit(col, maxAccum, w)
				resetFlake(f)
				continue
			}
		}
	}
	if st.World == WorldCity {
		smoothAccum()
	}
}

// DrawSnowForeground renders what snow has accumulated in front of the
// world: the ground bank and rooftop caps in the city, the rock dusting on
// the coast, and the pine tree.
func DrawSnowForeground(screen tcell.Screen, st *State) {
	w, h := st.Width, st.Height
	if w == 0 || h == 0 {
		return
	}
	if st.World == WorldCoast {
		drawRockSnow(screen, st)
		return
	}
	accumStyle := tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorReset).Attributes(tcell.AttrDim)
	accumTopStyle := tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorReset)
	maxAccum := st.snowMaxAccum(h)

	// ground bank, windswept from the deposit logic
	for x := 0; x < w; x++ {
		height := st.AccumRow[x]
		if height > maxAccum {
			height = maxAccum
			st.AccumRow[x] = height
		}
		for k := 0; k < height; k++ {
			y := h - 1 - k
			if y < 0 {
				break
			}
			ch := '·'
			style := accumStyle
			if k == height-1 {
				ch = '▁'
				style = accumTopStyle
			} else if k >= height-2 {
				ch = '▄'
			}
			screen.SetContent(x, y, ch, nil, style)
		}
	}

	// rooftop caps: white panels over the top of each snowed building
	white := tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorReset)
	for x := 0; x < w; x++ {
		rd := st.RoofAccum[x]
		if rd <= 0 {
			continue
		}
		roof := st.roofAt(x)
		if roof < 0 {
			continue
		}
		for k := 0; k < rd; k++ {
			y := roof + k
			if y >= st.City.HorizonY || y >= h {
				break
			}
			screen.SetContent(x, y, '▄', nil, white)
		}
	}
	drawPine(screen, st)
}

// drawRockSnow dusts the lighthouse rock white where flakes landed on it.
func drawRockSnow(screen tcell.Screen, st *State) {
	w, h := st.Width, st.Height
	horizon := st.City.HorizonY
	if horizon <= 0 || horizon >= h {
		return
	}
	white := tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorReset)
	dim := tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorReset).Attributes(tcell.AttrDim)
	lx := st.Coast.LighthouseX
	for x := lx - 2; x <= lx+2; x++ {
		if x < 0 || x >= w {
			continue
		}
		a := st.AccumRow[x]
		if a <= 0 {
			continue
		}
		screen.SetContent(x, horizon, '▀', nil, white)
		if a >= 2 && horizon+1 < h {
			screen.SetContent(x, horizon+1, '▄', nil, dim)
		}
	}
}

func drawAuroraSky(screen tcell.Screen, st *State) {
	w, h := st.Width, st.Height
	if w <= 0 || h <= 2 {
		return
	}
	bandTop := h / 3
	if bandTop < 3 {
		bandTop = 3
	}
	pulseBoost := 0.0
	if st.Frame%480 < 24 {
		pulseBoost = 0.18
	}
	if st.FocusMode {
		pulseBoost *= 0.5
	}
	for x := 0; x < w; x++ {
		wave := math.Sin(float64(x)*0.085+float64(st.Frame)*0.018)*0.55 +
			math.Sin(float64(x)*0.03-float64(st.Frame)*0.011)*0.45
		height := int((wave + 1.0) * 0.5 * float64(bandTop) * (0.82 + pulseBoost))
		if height < 2 {
			continue
		}
		for y := 0; y < height; y++ {
			intensity := 1.0 - float64(y)/float64(height)
			if intensity < 0.15 {
				continue
			}
			// sparse curtain: skip most mid-band cells for a softer glow
			skip := 4
			if intensity > 0.65 {
				skip = 3
			}
			if st.FocusMode {
				skip++
			}
			if (x+y+st.Frame)%skip != 0 {
				continue
			}
			phase := math.Sin(float64(x)*0.07 + float64(y)*0.12 + float64(st.Frame)*0.02)
			color := tcell.ColorTeal
			if phase > 0.35 {
				color = tcell.ColorGreen
			} else if phase < -0.2 {
				color = tcell.ColorPurple
			}
			style := tcell.StyleDefault.Foreground(color).Background(tcell.ColorReset).Attributes(tcell.AttrDim)
			ch := '·'
			if intensity > 0.75 && (x+st.Frame)%5 == 0 {
				ch = '░'
			}
			screen.SetContent(x, y, ch, nil, style)
		}
	}
}

func drawPine(screen tcell.Screen, st *State) {
	w, h := st.Width, st.Height
	if w < 18 || h < 10 {
		return
	}
	tree := []string{
		"    ^    ",
		"   /|\\   ",
		"  //|\\\\  ",
		" ///|\\\\\\ ",
		"////|\\\\\\\\",
		"   |||   ",
		"   |||   ",
	}
	startX := w - len(tree[0]) - 2
	startY := h - len(tree) - 2
	leafStyleDim := tcell.StyleDefault.Foreground(tcell.ColorGreen).Background(tcell.ColorReset).Attributes(tcell.AttrDim)
	leafStyle := tcell.StyleDefault.Foreground(tcell.ColorGreen).Background(tcell.ColorReset)
	trunkStyle := tcell.StyleDefault.Foreground(tcell.ColorMaroon).Background(tcell.ColorReset).Attributes(tcell.AttrDim)

	for y, row := range tree {
		for x, ch := range row {
			if ch == ' ' {
				continue
			}
			px := startX + x
			py := startY + y
			if px < 0 || px >= w || py < 0 || py >= h {
				continue
			}
			style := leafStyleDim
			if ch == '|' {
				style = trunkStyle
			} else if (px+py)%3 == 0 {
				style = leafStyle
			}
			screen.SetContent(px, py, ch, nil, style)
		}
	}
}
