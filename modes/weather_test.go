package modes

import (
	"os"
	"testing"
)

func TestConditionFromCode(t *testing.T) {
	cases := []struct {
		code int
		want Condition
	}{
		{0, CondClear},
		{1, CondCloudy},
		{2, CondCloudy},
		{3, CondCloudy},
		{45, CondCloudy},
		{48, CondCloudy},
		{51, CondRain},
		{61, CondRain},
		{80, CondRain},
		{71, CondSnow},
		{77, CondSnow},
		{85, CondSnow},
		{95, CondThunder},
		{99, CondThunder},
	}
	for _, c := range cases {
		if got := conditionFromCode(c.code); got != c.want {
			t.Errorf("conditionFromCode(%d) = %v, want %v", c.code, got, c.want)
		}
	}
}

func TestModeForWeather(t *testing.T) {
	cases := []struct {
		cond Condition
		want Mode
	}{
		{CondClear, ModeCalm},
		{CondCloudy, ModeCalm},
		{CondRain, ModeRain},
		{CondSnow, ModeSnow},
		{CondThunder, ModeThunderstorm},
	}
	for _, c := range cases {
		if got := ModeForWeather(c.cond); got != c.want {
			t.Errorf("ModeForWeather(%v) = %v, want %v", c.cond, got, c.want)
		}
	}
}

// TestLiveEndpoints hits the real Open-Meteo and ipwho.is APIs. Skipped
// unless RAIND_LIVE_TEST is set, so the offline suite never needs network.
func TestLiveEndpoints(t *testing.T) {
	if os.Getenv("RAIND_LIVE_TEST") == "" {
		t.Skip("set RAIND_LIVE_TEST=1 to hit the live weather APIs")
	}
	lat, lon, city, err := DetectCity()
	if err != nil {
		t.Fatalf("DetectCity: %v", err)
	}
	if city == "" {
		t.Error("DetectCity returned an empty city")
	}
	wd, err := fetchForecast(lat, lon)
	if err != nil {
		t.Fatalf("fetchForecast(%v, %v): %v", lat, lon, err)
	}
	if wd.Offline {
		t.Error("fetchForecast marked the result offline")
	}
	l2, n2, name, err := GeocodeCity("Tokyo")
	if err != nil {
		t.Fatalf("GeocodeCity: %v", err)
	}
	if name == "" || l2 == 0 || n2 == 0 {
		t.Errorf("GeocodeCity returned empty data: %q %v %v", name, l2, n2)
	}
	t.Logf("ip city=%q condition=%v temp=%.1f\n", city, wd.Condition, wd.Temperature)
}

func TestIntensityStaysInRange(t *testing.T) {
	conds := []Condition{CondClear, CondCloudy, CondRain, CondSnow, CondThunder}
	for _, cond := range conds {
		for _, p := range []float64{0, 5, 30} {
			wd := WeatherData{Condition: cond, Precipitation: p}
			if i := wd.Intensity(); i < 0 || i > 1 {
				t.Errorf("Intensity(%v, precip=%v) = %v, out of range", cond, p, i)
			}
			if cond == CondClear || cond == CondCloudy {
				if i := wd.Intensity(); i != 0 {
					t.Errorf("Intensity(%v) = %v, want 0", cond, i)
				}
			}
		}
	}
}
