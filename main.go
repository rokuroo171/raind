package main

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"raind/modes"

	"github.com/gdamore/tcell/v2"
)

type cliOptions struct {
	mode  string
	color string
	speed string
}

func printUsage() {
	fmt.Fprintf(os.Stderr, "raind - terminal weather screensaver\n\n")
	fmt.Fprintf(os.Stderr, "Usage: raind [options]\n\n")
	fmt.Fprintf(os.Stderr, "Options:\n")
	fmt.Fprintf(os.Stderr, "  --mode, -m <string>\n")
	fmt.Fprintf(os.Stderr, "        weather mode: rain, thunder, snow, meteor (default \"rain\")\n")
	fmt.Fprintf(os.Stderr, "  --color, -c <string>\n")
	fmt.Fprintf(os.Stderr, "        drop color: black, red, green, yellow, blue, magenta, cyan, white (default \"cyan\")\n")
	fmt.Fprintf(os.Stderr, "  --speed, -s <string>\n")
	fmt.Fprintf(os.Stderr, "        animation speed: slow, medium, fast (default \"medium\")\n")
	fmt.Fprintf(os.Stderr, "  --help, -h\n")
	fmt.Fprintf(os.Stderr, "        show this help message\n")
	fmt.Fprintf(os.Stderr, "\nRuntime keys: R/T/S/M modes, +/- speed, Q/Esc/Ctrl+C quit\n")
}

var shortFlags = map[string]string{
	"m": "mode",
	"c": "color",
	"s": "speed",
}

func parseCLI(args []string) (cliOptions, error) {
	opts := cliOptions{mode: "rain", color: "cyan", speed: "medium"}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if !strings.HasPrefix(arg, "-") {
			return opts, fmt.Errorf("raind: unexpected argument %q", arg)
		}

		name, value, hasValue, err := parseFlag(arg)
		if err != nil {
			return opts, err
		}
		if name == "help" {
			printUsage()
			os.Exit(0)
		}
		if !hasValue {
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				return opts, fmt.Errorf("raind: %s requires a value", flagLabel(name))
			}
			i++
			value = args[i]
		}
		switch name {
		case "mode":
			opts.mode = value
		case "color":
			opts.color = value
		case "speed":
			opts.speed = value
		default:
			return opts, fmt.Errorf("raind: unknown flag %s", flagLabel(name))
		}
	}
	return opts, nil
}

func flagLabel(name string) string {
	for short, long := range shortFlags {
		if long == name {
			return fmt.Sprintf("--%s, -%s", name, short)
		}
	}
	return "--" + name
}

func parseFlag(arg string) (name, value string, hasValue bool, err error) {
	if strings.HasPrefix(arg, "--") {
		name, value, hasValue = splitFlagBody(strings.TrimPrefix(arg, "--"))
		if name == "" {
			return "", "", false, fmt.Errorf("raind: invalid flag %q", arg)
		}
		return name, value, hasValue, nil
	}

	body := strings.TrimPrefix(arg, "-")
	if body == "" {
		return "", "", false, fmt.Errorf("raind: invalid flag %q", arg)
	}
	raw, value, hasValue := splitFlagBody(body)
	if raw == "h" || raw == "help" {
		return "help", "", false, nil
	}
	long, ok := shortFlags[raw]
	if !ok {
		return "", "", false, fmt.Errorf("raind: unknown flag -%s (try -h for help)", raw)
	}
	return long, value, hasValue, nil
}

func splitFlagBody(body string) (name, value string, hasValue bool) {
	if eq := strings.IndexByte(body, '='); eq >= 0 {
		return body[:eq], body[eq+1:], true
	}
	return body, "", false
}

func main() {
	opts, err := parseCLI(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(2)
	}

	mode, ok := modes.ParseMode(opts.mode)
	if !ok {
		fmt.Fprintf(os.Stderr, "raind: unknown mode %q\n", opts.mode)
		os.Exit(2)
	}
	color, ok := modes.ParseColor(opts.color)
	if !ok {
		fmt.Fprintf(os.Stderr, "raind: unknown color %q\n", opts.color)
		os.Exit(2)
	}
	speed, ok := modes.ParseSpeed(opts.speed)
	if !ok {
		fmt.Fprintf(os.Stderr, "raind: unknown speed %q\n", opts.speed)
		os.Exit(2)
	}

	screen, err := tcell.NewScreen()
	if err != nil {
		fmt.Fprintf(os.Stderr, "raind: %v\n", err)
		os.Exit(1)
	}
	defer screen.Fini()

	if err := screen.Init(); err != nil {
		fmt.Fprintf(os.Stderr, "raind: %v\n", err)
		os.Exit(1)
	}

	screen.SetStyle(tcell.StyleDefault.Background(tcell.ColorBlack))
	screen.Clear()
	screen.Show()

	w, h := screen.Size()
	state := &modes.State{
		Width:  w,
		Height: h,
		Mode:   mode,
		Speed:  speed,
		Color:  color,
	}
	initMode(state)

	quit := make(chan struct{})
	var closeQuit sync.Once
	requestQuit := func() {
		closeQuit.Do(func() { close(quit) })
	}

	delay := time.Duration(modes.FrameDelay(state.Mode, state.Speed)) * time.Millisecond
	ticker := time.NewTicker(delay)
	defer ticker.Stop()

	go pollEvents(screen, state, ticker, requestQuit)

	for {
		select {
		case <-quit:
			return
		case <-ticker.C:
			w, h := screen.Size()
			if w != state.Width || h != state.Height {
				state.Resize(w, h)
			}
			state.Frame++
			screen.Clear()
			drawMode(screen, state)
			screen.Show()
		}
	}
}

func initMode(st *modes.State) {
	switch st.Mode {
	case modes.ModeThunderstorm:
		st.InitThunderstorm()
	case modes.ModeSnow:
		st.InitSnow()
	case modes.ModeMeteor:
		st.InitMeteor()
	default:
		st.InitRain()
	}
}

func drawMode(screen tcell.Screen, st *modes.State) {
	switch st.Mode {
	case modes.ModeThunderstorm:
		modes.DrawThunderstorm(screen, st)
	case modes.ModeSnow:
		modes.DrawSnow(screen, st)
	case modes.ModeMeteor:
		modes.DrawMeteor(screen, st)
	default:
		modes.Draw(screen, st)
	}
}

func pollEvents(screen tcell.Screen, state *modes.State, ticker *time.Ticker, quit func()) {
	for {
		ev := screen.PollEvent()
		switch e := ev.(type) {
		case *tcell.EventKey:
			if e.Key() == tcell.KeyCtrlC {
				quit()
				return
			}
			if e.Key() == tcell.KeyEscape {
				quit()
				return
			}
			if e.Key() == tcell.KeyRune {
				switch e.Rune() {
				case 'q', 'Q':
					quit()
					return
				case 'r', 'R':
					state.Mode = modes.ModeRain
					state.InitRain()
					resetTicker(ticker, state)
				case 't', 'T':
					state.Mode = modes.ModeThunderstorm
					state.InitThunderstorm()
					resetTicker(ticker, state)
				case 's', 'S':
					state.Mode = modes.ModeSnow
					state.InitSnow()
					resetTicker(ticker, state)
				case 'm', 'M':
					state.Mode = modes.ModeMeteor
					state.InitMeteor()
					resetTicker(ticker, state)
				case '+', '=':
					bumpSpeed(state, 1)
					resetTicker(ticker, state)
				case '-', '_':
					bumpSpeed(state, -1)
					resetTicker(ticker, state)
				}
			}
		case *tcell.EventResize:
			w, h := screen.Size()
			state.Resize(w, h)
		case *tcell.EventError:
			quit()
			return
		}
	}
}

func resetTicker(ticker *time.Ticker, state *modes.State) {
	delay := time.Duration(modes.FrameDelay(state.Mode, state.Speed)) * time.Millisecond
	ticker.Reset(delay)
}

func bumpSpeed(st *modes.State, delta int) {
	s := int(st.Speed) + delta
	if s < int(modes.SpeedSlow) {
		s = int(modes.SpeedSlow)
	}
	if s > int(modes.SpeedFast) {
		s = int(modes.SpeedFast)
	}
	st.Speed = modes.SpeedLevel(s)
}
