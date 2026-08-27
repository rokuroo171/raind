package modes

import "testing"

func TestCountsFillNearPlane(t *testing.T) {
	st := &State{Width: 80, Height: 24}
	_, m1, n1, _ := st.countsFor(0.0)
	_, m2, n2, _ := st.countsFor(0.5)
	_, m3, n3, _ := st.countsFor(1.0)
	if n1 > n2 || n2 > n3 {
		t.Errorf("near plane should fill with intensity: %d, %d, %d", n1, n2, n3)
	}
	if m1 > m2 || m2 > m3 {
		t.Errorf("mid plane should fill with intensity: %d, %d, %d", m1, m2, m3)
	}
}

func TestCountsMatchTotal(t *testing.T) {
	st := &State{Width: 80, Height: 24}
	for _, iv := range []float64{0, 0.25, 0.5, 0.75, 1.0} {
		f, m, n, tot := st.countsFor(iv)
		if f+m+n != tot {
			t.Errorf("countsFor(%v) parts %d+%d+%d != total %d", iv, f, m, n, tot)
		}
		if tot < 1 {
			t.Errorf("countsFor(%v) produced no particles", iv)
		}
	}
}

func TestInitParticlesAssignsPlanes(t *testing.T) {
	st := &State{Width: 80, Height: 24}
	st.initParticles(1.0)
	counts := [3]int{}
	for _, p := range st.Particles {
		if p.Plane < PlaneNear || p.Plane > PlaneFar {
			t.Fatalf("bad plane %d", p.Plane)
		}
		counts[p.Plane]++
	}
	if counts[PlaneNear] == 0 || counts[PlaneFar] == 0 {
		t.Errorf("full rain should have near and far planes, got %+v", counts)
	}
}

func TestFarDropsRespawnSilently(t *testing.T) {
	st := &State{Width: 40, Height: 24}
	p := particleForPlane(PlaneFar)
	oldY := p.Y
	p.Y = 23
	st.splashAt(&p, 23)
	// silent: no splash counter, just a respawn at the top
	if p.Splash != 0 {
		t.Errorf("far drop should not splash, got Splash=%d", p.Splash)
	}
	if p.Y != 0 {
		t.Errorf("far drop should respawn at top, got Y=%v (was %v)", p.Y, oldY)
	}
}

func TestNearDropsSplash(t *testing.T) {
	st := &State{Width: 40, Height: 24}
	p := particleForPlane(PlaneNear)
	p.Y = 23
	st.splashAt(&p, 23)
	if p.Splash == 0 {
		t.Error("near drop should throw a visible ripple")
	}
	if p.Y != 0 {
		t.Errorf("splashed near drop should respawn, got Y=%v", p.Y)
	}
}
