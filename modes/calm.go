package modes

import (
	"math"
	"math/rand"
	"time"

	"github.com/gdamore/tcell/v2"
)

type Cloud struct {
	X, Y  float64
	W     int
	Cells []rune
	Speed float64
}

// only shade chars, no solid half-blocks, so clouds drift as a wisp
// instead of scattering blocky pixels across the sky
var cloudChars = []rune{'░', '▒'}

// daytime returns how much daylight a given hour has, 0 at night and 1 at noon.
// Sunrise is 6, sunset is 18, and the curve is a sine so transitions are smooth.
func daytime(hour float64) float64 {
	d := math.Sin(math.Pi * (hour - 6) / 12)
	if d < 0 {
		return 0
	}
	return d
}

func (st *State) initClouds() {
	if st.Width <= 0 || st.Height <= 0 {
		st.Clouds = nil
		return
	}
	n := st.Width / 16
	if n < 2 {
		n = 2
	}
	if n > 8 {
		n = 8
	}
	clouds := make([]Cloud, n)
	for i := range clouds {
		c := &clouds[i]
		c.W = 3 + rand.Intn(4)
		c.Cells = make([]rune, c.W)
		for j := range c.Cells {
			c.Cells[j] = cloudChars[rand.Intn(len(cloudChars))]
		}
		c.X = float64(rand.Intn(st.Width + 20))
		c.Y = float64(2 + rand.Intn(st.City.HorizonY-4))
		c.Speed = 0.02 + rand.Float64()*0.05
	}
	st.Clouds = clouds
}

func (st *State) drawClouds(screen tcell.Screen) {
	style := tcell.StyleDefault.Foreground(tcell.NewHexColor(0x9aa4b8)).Background(tcell.ColorReset)
	if st.Night > 0.5 {
		style = tcell.StyleDefault.Foreground(tcell.NewHexColor(0x4a5266)).Background(tcell.ColorReset)
	}
	for i := range st.Clouds {
		c := &st.Clouds[i]
		c.X += c.Speed
		if c.X > float64(st.Width)+2 {
			c.X = -float64(c.W) - 2
			c.Y = float64(2 + rand.Intn(st.City.HorizonY-4))
		}
		x := int(c.X)
		y := int(c.Y)
		for j := 0; j < c.W; j++ {
			px := x + j
			if px < 0 || px >= st.Width || y < 1 || y >= st.City.HorizonY {
				continue
			}
			screen.SetContent(px, y, c.Cells[j], nil, style)
		}
	}
}

func (st *State) drawSun(screen tcell.Screen) {
	horizon := st.City.HorizonY
	if horizon <= 0 {
		return
	}
	hour := timeHour()
	day := daytime(hour)
	if day > 0.02 {
		x := int(float64(st.Width) * (hour - 6) / 12)
		elev := horizon - 2 - int(day*float64(horizon-6))
		if x < 0 {
			x = 0
		}
		if x >= st.Width {
			x = st.Width - 1
		}
		halo := tcell.StyleDefault.Foreground(tcell.NewHexColor(0xffd98a)).Background(tcell.ColorReset).Attributes(tcell.AttrDim)
		core := tcell.StyleDefault.Foreground(tcell.NewHexColor(0xfff2c8)).Background(tcell.ColorReset)
		for dx := -2; dx <= 2; dx++ {
			for dy := -1; dy <= 1; dy++ {
				if dx == 0 && dy == 0 {
					continue
				}
				sx := x + dx
				sy := elev + dy
				if sx >= 0 && sx < st.Width && sy >= 0 && sy < horizon {
					screen.SetContent(sx, sy, '░', nil, halo)
				}
			}
		}
		if elev >= 0 && elev < horizon {
			screen.SetContent(x, elev, '●', nil, core)
		}
	}
}

func (st *State) drawMoonStars(screen tcell.Screen) {
	horizon := st.City.HorizonY
	if horizon <= 2 {
		return
	}
	if st.Night > 0.45 {
		x := st.Width * 3 / 4
		elev := 3 + int(st.Night*float64(horizon-8))
		moon := tcell.StyleDefault.Foreground(tcell.NewHexColor(0xdfe6f0)).Background(tcell.ColorReset)
		if elev >= 0 && elev < horizon {
			screen.SetContent(x, elev, '☾', nil, moon)
		}
	}
	if st.Night > 0.6 {
		dim := tcell.StyleDefault.Foreground(tcell.NewHexColor(0x9aa4b8)).Background(tcell.ColorReset).Attributes(tcell.AttrDim)
		n := st.Width / 15
		if n < 6 {
			n = 6
		}
		for i := 0; i < n; i++ {
			x := (i * 7919) % st.Width
			y := (i*104729)%(horizon-2) + 1
			if (x+y+st.Frame)%40 < 8 {
				screen.SetContent(x, y, '·', nil, dim)
			}
		}
	}
}

func timeHour() float64 {
	now := time.Now()
	return float64(now.Hour()) + float64(now.Minute())/60
}
