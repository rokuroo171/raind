package modes

import (
	"math"
	"math/rand"

	"github.com/gdamore/tcell/v2"
)

// WorldKind selects the terrain behind the weather.
type WorldKind int

const (
	WorldCoast WorldKind = iota
	WorldCity
)

type Boat struct {
	X      float64
	Y      int
	Dir    float64
	Active bool
	Timer  int
}

type Gull struct {
	X, Y   float64
	Dir    float64
	Speed  float64
	Active bool
	Timer  int
}

type CoastScene struct {
	LighthouseX int
	LighthouseH int
	Boat        Boat
	Gulls       []Gull
}

// initCoast lays out the lighthouse on the left third and arms the boat and
// gull timers. The sea itself is drawn procedurally, no state needed.
func (st *State) initCoast(w, h int) {
	horizon := st.City.HorizonY
	if horizon <= 0 || w <= 0 {
		st.Coast = CoastScene{}
		return
	}
	lh := horizon / 5
	if lh < 4 {
		lh = 4
	}
	if lh > horizon-1 {
		lh = horizon - 1
	}
	st.Coast = CoastScene{
		LighthouseX: w / 5,
		LighthouseH: lh,
		Boat:        Boat{Timer: 90 + rand.Intn(120)},
		Gulls: []Gull{
			{Timer: 60 + rand.Intn(90)},
			{Timer: 90 + rand.Intn(90)},
		},
	}
}

func drawSea(screen tcell.Screen, st *State) {
	w, h := st.Width, st.Height
	horizon := st.City.HorizonY
	if horizon <= 0 || horizon >= h {
		return
	}
	// colour palette: shore foam, mid-wave crest, sparkle, deep trough, shimmer
	foam := tcell.StyleDefault.Foreground(tcell.NewHexColor(0x6a7a8f)).Background(tcell.ColorReset)
	crest := tcell.StyleDefault.Foreground(tcell.NewHexColor(0x3d4f63)).Background(tcell.ColorReset)
	spark := tcell.StyleDefault.Foreground(tcell.NewHexColor(0x4a5c70)).Background(tcell.ColorReset)
	deep := tcell.StyleDefault.Foreground(tcell.NewHexColor(0x1a2230)).Background(tcell.ColorReset)
	shimmer := tcell.StyleDefault.Foreground(tcell.NewHexColor(0x5a6c80)).Background(tcell.ColorReset)
	depth := h - horizon
	frameT := float64(st.Frame)
	for y := horizon; y < h; y++ {
		row := float64(y - horizon)
		rowNorm := row / float64(depth) // 0 at horizon, 1 at bottom
		// three travelling waves at different scales; phase couples weakly to
		// the row so crests roll sideways rather than diagonally
		phase := frameT*0.045 + row*0.06
		for x := 0; x < w; x++ {
			v := math.Sin(float64(x)*0.18+phase) +
				0.35*math.Sin(float64(x)*0.08+frameT*0.025+row*0.15) +
				0.15*math.Sin(float64(x)*0.35+frameT*0.04)
			// choppy ripples near shore
			if row > float64(depth)-3 {
				v += 0.25 * math.Sin(float64(x)*0.5-frameT*0.08)
			}
			var ch rune
			var style tcell.Style
			switch {
			// foam / whitecap
			case v > 1.25:
				ch, style = '≈', foam
			// crest
			case v > 0.9:
				ch, style = '~', crest
			// mid-water sparkle (sparse)
			case v > 0.7 && math.Sin(float64(x)*1.2+frameT*0.08+row*0.4) > 0.7:
				ch, style = '·', spark
			// shimmer dot (very rare)
			case v > 0.6 && math.Sin(float64(x)*0.9+frameT*0.12+row*0.3) > 0.92:
				ch, style = '∙', shimmer
			// deep troughs below the mid-line
			case v < -1.15 && rowNorm > 0.5:
				ch, style = '·', deep
			default:
				continue
			}
			screen.SetContent(x, y, ch, nil, style)
		}
	}
}

// drawSeaFlash turns the whole sea white for the frames of a lightning
// strike, the ocean-storm image the coast owns.
func drawSeaFlash(screen tcell.Screen, st *State) {
	if st.StormFlash <= 0 {
		return
	}
	w, h := st.Width, st.Height
	horizon := st.City.HorizonY
	if horizon <= 0 {
		return
	}
	flash := tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorReset)
	for y := horizon; y < h; y++ {
		for x := 0; x < w; x++ {
			screen.SetContent(x, y, '░', nil, flash)
		}
	}
}

// drawReflection drops a faint light column on the water under the sun by
// day and the moon by night, the one-accent shot of the calm coast.
func drawReflection(screen tcell.Screen, st *State) {
	w, h := st.Width, st.Height
	horizon := st.City.HorizonY
	if horizon <= 0 || horizon+2 >= h {
		return
	}
	style := tcell.StyleDefault.Foreground(tcell.NewHexColor(0x6d7c94)).Background(tcell.ColorReset).Attributes(tcell.AttrDim)
	col := -1
	hour := timeHour()
	if day := daytime(hour); day > 0.25 {
		col = int(float64(w) * (hour - 6) / 12)
	} else if st.Night > 0.5 {
		col = w * 3 / 4
	}
	if col < 0 || col >= w {
		return
	}
	for y := horizon + 1; y < horizon+4 && y < h; y++ {
		ch := '░'
		if y == horizon+3 {
			ch = '·'
		}
		screen.SetContent(col, y, ch, nil, style)
	}
}

func drawLighthouse(screen tcell.Screen, st *State) {
	w, h := st.Width, st.Height
	horizon := st.City.HorizonY
	x := st.Coast.LighthouseX
	top := horizon - st.Coast.LighthouseH
	if top < 0 {
		top = 0
	}
	if x < 0 || x >= w-1 || top >= horizon {
		return
	}
	tower := tcell.StyleDefault.Foreground(tcell.NewHexColor(0xe9edf5)).Background(tcell.ColorReset)
	lantern := tcell.StyleDefault.Foreground(tcell.NewHexColor(0xffe9a8)).Background(tcell.ColorReset)
	band := tcell.StyleDefault.Foreground(tcell.NewHexColor(0xb3453a)).Background(tcell.ColorReset)
	rock := tcell.StyleDefault.Foreground(tcell.NewHexColor(0x232a36)).Background(tcell.ColorReset)
	// pure white bold, not dim cream, so the beam reads as light even in
	// terminals that map warm tones to brown
	beam := tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorReset).Attributes(tcell.AttrBold)

	stripe := top + st.Coast.LighthouseH/2
	for y := top; y < horizon; y++ {
		style := tower
		if y < top+2 {
			style = lantern
		} else if y == stripe {
			style = band
		}
		for dx := 0; dx < 2; dx++ {
			screen.SetContent(x+dx, y, '█', nil, style)
		}
	}
	for dx := -2; dx <= 2; dx++ {
		rx := x + dx
		if rx >= 0 && rx < w && horizon < h {
			screen.SetContent(rx, horizon, '▄', nil, rock)
			if horizon+1 < h {
				screen.SetContent(rx, horizon+1, '░', nil, rock)
			}
		}
	}
	// straight beam at lantern height, blinking on a slow cycle
	if st.Night > 0.45 && (st.Frame/36)%3 != 0 {
		for i := 2; i < 14; i++ {
			bx := x + i
			if bx >= w {
				break
			}
			screen.SetContent(bx, top+1, '░', nil, beam)
		}
	}
}

func (st *State) updateBoat(screen tcell.Screen) {
	w, h := st.Width, st.Height
	horizon := st.City.HorizonY
	if w <= 0 || horizon <= 0 || horizon+2 >= h {
		return
	}
	b := &st.Coast.Boat
	if !b.Active {
		b.Timer--
		if b.Timer <= 0 {
			b.Active = true
			b.Dir = 1
			if rand.Float64() < 0.5 {
				b.Dir = -1
			}
			if b.Dir < 0 {
				b.X = float64(w)
			} else {
				b.X = -4
			}
			b.Y = horizon + 2
		}
		return
	}
	b.X += b.Dir * 0.1
	clear := func() {
		b.Active = false
		b.Timer = 140 + rand.Intn(140)
	}
	if b.X < -6 || b.X > float64(w)+6 {
		clear()
		return
	}
	x := int(b.X)
	if x < 1 || x >= w-1 {
		return
	}
	style := tcell.StyleDefault.Foreground(tcell.NewHexColor(0x6b768c)).Background(tcell.ColorReset)
	hull := style
	if st.Mode == ModeSnow {
		// the boat roof whitens under the snowfall
		hull = tcell.StyleDefault.Foreground(tcell.NewHexColor(0xd7dde8)).Background(tcell.ColorReset)
	}
	wake := tcell.StyleDefault.Foreground(tcell.NewHexColor(0x54647c)).Background(tcell.ColorReset).Attributes(tcell.AttrDim)
	screen.SetContent(x, b.Y-1, '│', nil, style)
	screen.SetContent(x-1, b.Y, '_', nil, hull)
	screen.SetContent(x, b.Y, '_', nil, hull)
	screen.SetContent(x+1, b.Y, '_', nil, hull)
	for k := 1; k <= 3; k++ {
		wx := x - int(b.Dir*float64(k))
		if wx >= 0 && wx < w && b.Y+1 < h {
			screen.SetContent(wx, b.Y+1, '·', nil, wake)
		}
	}
}

func (st *State) updateGulls(screen tcell.Screen) {
	w := st.Width
	horizon := st.City.HorizonY
	if w <= 0 || horizon < 8 {
		return
	}
	style := tcell.StyleDefault.Foreground(tcell.NewHexColor(0x7a8499)).Background(tcell.ColorReset).Attributes(tcell.AttrDim)
	for i := range st.Coast.Gulls {
		g := &st.Coast.Gulls[i]
		if !g.Active {
			g.Timer--
			if g.Timer <= 0 {
				g.Active = true
				g.Dir = 1
				if rand.Float64() < 0.5 {
					g.Dir = -1
				}
				if g.Dir < 0 {
					g.X = float64(w)
				} else {
					g.X = -2
				}
				g.Y = float64(3 + rand.Intn(horizon-8))
				g.Speed = 0.05 + rand.Float64()*0.05
			}
			continue
		}
		g.X += g.Dir * g.Speed
		if g.X < -4 || g.X > float64(w)+4 {
			g.Active = false
			g.Timer = 90 + rand.Intn(120)
			continue
		}
		x := int(g.X)
		y := int(g.Y)
		if x < 0 || x >= w || y < 0 || y >= horizon {
			continue
		}
		ch := '⌒'
		if (st.Frame+i)%16 < 8 {
			ch = '⌣'
		}
		screen.SetContent(x, y, ch, nil, style)
	}
}

// drawCoast renders the sea and its sparse life in front of the weather.
func (st *State) drawCoast(screen tcell.Screen) {
	drawSea(screen, st)
	drawReflection(screen, st)
	drawLighthouse(screen, st)
	st.updateBoat(screen)
	st.updateGulls(screen)
	drawSeaFlash(screen, st)
}
