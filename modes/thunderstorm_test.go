package modes

import (
	"math/rand"
	"testing"
)

func TestGenerateBoltStopsAtTarget(t *testing.T) {
	st := &State{Width: 40, Height: 24}
	st.City = NewSkyline(3, 40, 24)
	stop := st.City.HorizonY - 3
	bolt := st.generateBolt(40, 24, 7, stop, 0, 0.5, false)
	if len(bolt.Channel.Points) == 0 {
		t.Fatal("expected bolt points")
	}
	last := bolt.Channel.Points[len(bolt.Channel.Points)-1]
	if last.Y != stop-1 {
		t.Fatalf("bolt did not stop at target: last Y %d, want %d", last.Y, stop-1)
	}
	for _, p := range bolt.Channel.Points {
		if p.Y < 0 || p.Y >= stop {
			t.Fatalf("bolt point out of range: %+v", p)
		}
	}
	for _, br := range bolt.Channel.Branches {
		for _, p := range br {
			if p.X < 0 || p.X >= 40 || p.Y < 0 || p.Y >= stop {
				t.Fatalf("branch point out of range: %+v", p)
			}
		}
	}
	if bolt.Strokes < 2 || bolt.Strokes > 4 {
		t.Fatalf("expected 2-4 strokes, got %d", bolt.Strokes)
	}
	if bolt.FramesLeft <= 0 {
		t.Fatal("bolt should start visible")
	}
}

func TestCloseBoltHasLeader(t *testing.T) {
	st := &State{Width: 40, Height: 24}
	st.City = NewSkyline(3, 40, 24)
	b := st.generateBolt(40, 24, 7, st.City.HorizonY-1, 1, 0.8, true)
	if b.LeaderFrames <= 0 {
		t.Fatal("close bolt should start in the leader phase")
	}
	if b.LeaderLen < 1 {
		t.Fatal("leader should have descended at least one point")
	}
}

func TestDistantBoltSkipsLeader(t *testing.T) {
	st := &State{Width: 40, Height: 24}
	st.City = NewSkyline(3, 40, 24)
	b := st.generateBolt(40, 24, 7, st.City.HorizonY-1, 1, 0.2, false)
	if b.LeaderFrames != 0 {
		t.Fatal("distant bolt should flash immediately without a leader")
	}
	if b.FramesLeft <= 0 {
		t.Fatal("distant bolt should start visible")
	}
}

func TestBranchesStartInLowerHalf(t *testing.T) {
	rand.Seed(7)
	st := &State{Width: 40, Height: 24}
	st.City = NewSkyline(3, 40, 24)
	for i := 0; i < 40; i++ {
		b := st.generateBolt(40, 24, 19, st.City.HorizonY-1, 1, 0.8, true)
		pts := b.Channel.Points
		if len(pts) < 8 {
			continue
		}
		// trunk Y advances exactly one per point, so a branch origin at
		// index i sits at startY + i
		minStart := len(pts) * 2 / 5
		for _, br := range b.Channel.Branches {
			if len(br) == 0 {
				continue
			}
			if br[0].Y < pts[0].Y+minStart-1 {
				t.Fatalf("branch origin too high: Y %d, want >= %d", br[0].Y, pts[0].Y+minStart-1)
			}
		}
	}
}

func TestCloudFlashLightsNoBolt(t *testing.T) {
	st := &State{Width: 40, Height: 24, StormIntensity: 0.5, CellPos: 0.5, CellDist: 0.3}
	st.City = NewSkyline(3, 40, 24)
	st.cloudFlash()
	if st.CloudGlow <= 0 {
		t.Fatal("cloud flash should set a glow timer")
	}
	if st.CloudGlowX < 0 || st.CloudGlowX >= 40 {
		t.Fatalf("cloud glow column out of range: %d", st.CloudGlowX)
	}
	if st.StormFlash <= 0 {
		t.Fatal("cloud flash should flash the sky")
	}
	if len(st.Bolts) != 0 {
		t.Fatal("cloud flash should not spawn a ground bolt")
	}
}

func TestBurstFiresThenLulls(t *testing.T) {
	st := &State{Width: 40, Height: 24, StormIntensity: 0.6, BurstDoubleChance: 0}
	st.City = NewSkyline(3, 40, 24)
	st.initCoast(40, 24)
	st.BurstLeft = 0
	st.BurstTimer = 0
	st.LullTimer = 0
	// a lull just ended: the cadence arms a burst of 2 to 4 strikes
	st.updateCadence()
	if st.BurstLeft <= 0 {
		t.Fatal("expected a burst to be armed")
	}
	fired := 0
	for st.BurstLeft > 0 {
		before := len(st.Bolts)
		st.updateCadence()
		if len(st.Bolts) > before {
			fired++
		}
	}
	if fired < 4 || fired > 8 {
		t.Fatalf("burst should fire 4-8 strikes, got %d", fired)
	}
	if st.LullTimer <= 0 {
		t.Fatal("finished burst should enter a lull")
	}
}

// TestCadenceNeverStarvates walks the state machine through a full cycle:
// lull, burst arms, burst fires, lull again, then a lull ending re-arms a
// burst on its own. This is the regression for the storm that never struck.
func TestCadenceNeverStarvates(t *testing.T) {
	st := &State{Width: 40, Height: 24, StormIntensity: 0.6, BurstDoubleChance: 0}
	st.City = NewSkyline(3, 40, 24)
	st.initCoast(40, 24)
	st.BurstLeft = 0
	st.BurstTimer = 0
	st.LullTimer = 1

	arms := 0
	armed := false
	for frame := 0; frame < 20000; frame++ {
		st.updateCadence()
		if st.BurstLeft > 0 && !armed {
			arms++
		}
		armed = st.BurstLeft > 0
	}
	if arms < 3 {
		t.Fatalf("lulls kept re-arming bursts, got %d arms in 20k frames", arms)
	}

	// normalize: if we are mid-lull, force it to end so a burst is armed
	if st.BurstLeft == 0 {
		st.LullTimer = 0
		st.updateCadence()
	}
	if st.BurstLeft <= 0 {
		t.Fatal("a lull ending should arm a burst")
	}
	// walk the burst to completion; it must end in a lull
	for st.BurstLeft > 0 {
		st.updateCadence()
	}
	if st.LullTimer <= 0 {
		t.Fatal("a finished burst should start a lull")
	}
	// walk the lull to completion, then one more tick arms the next burst
	for st.LullTimer > 0 {
		st.updateCadence()
	}
	st.updateCadence()
	if st.BurstLeft <= 0 {
		t.Fatal("an ending lull should re-arm a burst")
	}
}

// TestDoubleBurstChainsStrikes pins the irregular-cadence feature: when a
// burst ends, a double burst can follow almost immediately instead of a lull.
func TestDoubleBurstChainsStrikes(t *testing.T) {
	st := &State{Width: 40, Height: 24, StormIntensity: 0.6, BurstDoubleChance: 1.0}
	st.City = NewSkyline(3, 40, 24)
	st.initCoast(40, 24)
	st.BurstLeft = 1
	st.BurstTimer = 1
	st.LullTimer = 0

	// one strike fires, the burst ends, and with chance 1 the cadence arms a
	// follow-up burst immediately instead of a lull
	st.updateCadence()
	if st.BurstLeft <= 0 {
		t.Fatal("double-burst should arm a follow-up burst, not silence")
	}
	if st.LullTimer != 0 {
		t.Fatalf("double-burst should not start a lull, got %d", st.LullTimer)
	}
	// the chain keeps firing; cap at 40 frames and require several strikes
	base := len(st.Bolts)
	fired := 0
	for iter := 0; iter < 40 && st.BurstLeft > 0; iter++ {
		before := len(st.Bolts)
		st.updateCadence()
		if len(st.Bolts) > before {
			fired++
		}
	}
	if fired < 2 || len(st.Bolts) <= base {
		t.Fatalf("chained bursts should keep firing, got %d new strikes", fired)
	}
}

func TestRooftopsWithinBounds(t *testing.T) {
	st := &State{Width: 40, Height: 24, World: WorldCity}
	st.City = NewSkyline(5, 40, 24)
	st.updateRooftops()
	if len(st.RoofTop) != 40 {
		t.Fatalf("rooftop width %d, want 40", len(st.RoofTop))
	}
	found := false
	for x, roof := range st.RoofTop {
		if roof == -1 {
			continue
		}
		found = true
		if roof < 0 || roof >= st.City.HorizonY {
			t.Fatalf("roof at %d out of bounds: %d (horizon %d)", x, roof, st.City.HorizonY)
		}
	}
	if !found {
		t.Fatal("expected at least one roofed column")
	}
}

func TestCoastRooftopsLandOnWater(t *testing.T) {
	st := &State{Width: 40, Height: 24, World: WorldCoast}
	st.City = NewSkyline(5, 40, 24)
	st.initCoast(40, 24)
	st.updateRooftops()
	if len(st.RoofTop) != 40 {
		t.Fatalf("rooftop width %d, want 40", len(st.RoofTop))
	}
	for x, roof := range st.RoofTop {
		if roof != st.City.HorizonY {
			t.Fatalf("coast roof at %d = %d, want sea level %d", x, roof, st.City.HorizonY)
		}
	}
	if st.Coast.LighthouseX <= 0 || st.Coast.LighthouseX >= 40 {
		t.Fatalf("lighthouse x %d out of range", st.Coast.LighthouseX)
	}
	if st.Coast.LighthouseH <= 0 || st.Coast.LighthouseH >= st.City.HorizonY {
		t.Fatalf("lighthouse height %d out of range (horizon %d)", st.Coast.LighthouseH, st.City.HorizonY)
	}
}
