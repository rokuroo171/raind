# raind v0.2.0 roadmap: The Quiet Coast

Target: v0.2.0 (current release is v0.1.1).

## The story

raind becomes a bounded world instead of a particle field: a horizon, a sea, and a
lighthouse on a rock. Rain winks out on the water, snow dusts the rock, lightning
strikes the lighthouse and flashes the whole sea white. The target audience is the
rice crowd, dwm / niri / hyprland, and they want minimal: negative space is the
product, most of the frame stays empty, the water is the only constant motion.

One line for a rice post: "my terminal shows the weather off my coast."

## Why this direction

cmatrix, cava, pipes.sh, and asciiquarium all throw particles at the screen. None of
them render a world that weather can interact with. Real weather plus a living scene is
the hook that makes raind the answer to "which screensaver do I keep running?"

## What ships in v0.2.0

1. **The coast.** Horizon plus sea. The sea is rendered from traveling phase waves,
   mostly empty, sparse crests. The lighthouse sits close on the left third, the only
   structure, white tower, lantern room, sweeping beam at night. Sun by day and moon by
   night drop a faint reflection column on the water.
2. **Sparse life.** One boat crosses the frame every couple of minutes, a gull or two
   drift over, then the frame is empty again. The aquarium rule: punctuation, not
   constant motion.
3. **Calm scene.** The mode that fills a clear day: clouds crossing, sun position that
   tracks real time of day. Calm is allowed to be calm, the water carries the motion.
4. **Live weather.** Open-Meteo, keyless, no signup, no rate limits at our scale.
   Auto-detect city from IP on first run, optional `--city` override. Refresh every few
   minutes. On fetch failure or offline: fall back to simulated weather with a flat
   grey band at the horizon as a visible tell, so the user always knows whether the
   terminal is telling the truth. Never hang, never block startup on the network.
5. **Storm at sea.** Lightning aims at the lighthouse and flashes the whole sea white
   for a frame or two, the iconic ocean-storm image. Rain winks out at the water with
   a faint ripple, snow fades into the sea and dusts only the rock.
6. **Release mechanics.** Bump `VERSION` default in `install.sh`, tag `v0.2.0`,
   GoReleaser picks the version from the tag.

## Deliberately not in v0.2.0

- **System reactivity** (storm intensity from CPU load, fan speed, builds). Later
  feature, not the headline.
- **Settings UI, Bubble Tea.** Rejected. Bubble Tea is an event-driven TUI framework
  and fights raind's frame loop; two terminal engines in one static binary. raind's UI
  stays flags plus hotkeys.
- **Config file.** `--city` is optional. Zero setup remains the rule, cmatrix-style.
- **Noise.** Rain and thunder sound was seen in the wild (terminal-rain-lightning,
  ffplay-based) and is a real gap, but it drags an external dependency into the
  static binary story. Draft for later, gated behind `--sound` if it ever lands.

## Build order

1. Sea renderer: traveling phase waves, judged alone before anything else lands.
2. Lighthouse, boat, gulls, reflections.
3. Calm scene rework onto the coast.
4. Storm at sea: lighthouse-targeted bolts, whole-sea flash.
5. Live weather hook and offline fallback (done in the city pass, re-verify on coast).

## Draft: more worlds (post-v0.2.0, pending a grill)

Idea: raind supports multiple terrain worlds, selectable with `--world`. The coast is
now the v0.2.0 identity; the city survives as a draft alternate world behind the flag.

Candidates for the grill:

- City: the previous design, skyline silhouette, amber night windows, kept as a draft
  alternate world behind `--world city`.
- Forest: canopy skyline, snow on branches, lightning striking the tallest pine.
- Mountains: jagged ridge, weather rolling over peaks, snow line.
- Desert: dunes, cacti, heat shimmer, clear starlit nights.
- Countryside: rolling hills, a farmhouse, windmill blades.

Architectural note already honored: the terrain is behind a `WorldKind` dispatch with
weather interaction points (strike target, landing height, accumulation rule), so a
new world is a new draw function plus a few per-world branches, not a rewrite.

## Draft: mode revamp (weather-driven depth, post-grill)

Grilled and locked. One depth-layer system as the backend, one mode per commit,
rain first. Every mode answers to live weather and reads as depth. The current modes
are ambient randomness; the revamp makes them weather-honest like the coast world.

- Rain: three depth planes (near thick, fast, bright, wind-driven / mid short, dim /
  far sparse, slow, near-vertical). Fall angle tracks live Open-Meteo wind. Rolling
  transverse gust bands cross the frame and the field relaxes behind them. Splashes
  split by plane: far drops wink out at the sea, mid throws a faint ripple, near
  throws the visible one. Precipitation fills planes: light rain shows only the far
  plane, a downpour weights the near plane.
- Thunder: each bolt is a channel that re-fires 2 to 4 strokes per flash and sprouts
  2 to 3 jittered branches, sea flash scales with bolt power. Frequency and power
  come from live Intensity. Storms stage by distance, a drifting cell flashes low
  and faint then overhead and huge. Fire in bursts, 2 to 4 strikes together, then a
  lull. Intensity sets burst length and lull depth.
- Snow: shares the rolling gust machinery with rain, the whole field leans together
  as a band passes. Accumulation is per surface: city rooftops catch, the lighthouse
  rock dusts, the boat roof whitens, light dusts and whiteouts bank.
- Meteor: all trails point back at a single drifting radiant point so the field
  reads as deep space, not random streaks. Live clarity gates activity: clear nights
  star the shower, overcast nearly silences it, matching the sky outside.

Build order: rain depth engine first (it owns the layer system), then thunder, snow,
meteor, one commit each.