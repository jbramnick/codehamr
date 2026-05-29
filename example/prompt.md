# Procedural Galaxy Explorer — Single-File Build

Build a single self-contained `galaxy.html` file that opens directly in a modern browser and renders an explorable procedural galaxy in WebGL. No build step, no external assets, no images, no servers — just one HTML file with inline CSS, an ES-module import map for Three.js loaded from a CDN, and inline JavaScript. Target around 2000–2500 lines.

## High-level vision

The player floats in deep space at the start. A cinematic intro plays: the camera slowly orbits a yellow home star while the title "GALAXY — A PROCEDURAL UNIVERSE" sits centered on screen and a "CLICK TO START" prompt waits at the bottom. Clicking acquires pointer lock, hides the overlay, and hands control to the player. From there the experience is a first-person 6-DOF flight through an effectively infinite procedural universe full of stars, planets, moons, asteroid belts, ringed gas giants, drifting nebulae, and a distant black hole.

Bias every choice toward looking *cinematic* and *atmospheric*. This is not a game with goals — it is a sandbox sci-fi mood piece. The player should feel small, the universe should feel vast, and every direction they look should have something interesting to find.

The reference aesthetic is something between *Elite Dangerous* exterior shots, *Stellaris* galaxy-map close-ups, and *Star Citizen* atmosphere shaders — but built entirely out of math, with the budget of a single HTML file.

## Tech stack and constraints

- One file, `galaxy.html`. Inline CSS and JS only.
- Three.js loaded via `<script type="importmap">` pointing at the unpkg CDN.
- Use the postprocessing addons (EffectComposer, RenderPass, UnrealBloomPass, ShaderPass) for the look.
- Logarithmic depth buffer — you are mixing meter-scale ships with hundreds-of-thousands-of-units cosmic distances, the near/far split is huge.
- ACES filmic tone mapping, low exposure (~0.4). The default is too bright for a starfield.
- sRGB output color space.
- No external textures whatsoever. Everything procedural — generate textures inside `<canvas>` 2D and write your own noise-based GLSL shaders.
- No external audio. Synthesize every sound live with the Web Audio API.
- Keep helpers shared and reusable. Define one mulberry32 PRNG, one smoothstep, one position-hash noise, one orthonormal-basis builder, and reuse them everywhere instead of duplicating noise math across shaders and helpers.
- Reuse scratch `Vector3`/`Quaternion`/`Matrix4`/`Color` instances at module scope inside hot loops. Allocating new ones per frame will tank performance.

## Universe structure — endless streaming

The universe is divided into cubic chunks (around 20k units per side). Maintain a sphere of active chunks around the camera (radius ~10 chunks in each direction). When the camera crosses a chunk boundary, evict far chunks and stream new ones in. Each chunk is deterministic given its integer coordinates — use a hash-seeded PRNG (mulberry32 style) so the same chunk regenerates identically across visits.

Group chunks into larger regions of about 4×4×4 chunks. Each region picks a *theme* deterministically:
- **void** — empty space, no stars
- **cluster** — multiple stars clumped close together
- **normal** — one star per chunk, ordinary sizes
- **giants** — one large supergiant per chunk

The chunk at world origin (0,0,0) must always contain the home star, a sun-like G-class yellow.

## Stars

Each star has a spectral class chosen by random roll: M (red, most common), K (orange), G (yellow), F (white), A (blue-white). Sizes range from small (~50 units) to enormous (~700 units for supergiants).

Render stars in three layered passes for performance:

1. **Far points** — a `THREE.Points` cloud with a custom shader. Stars grow on screen as you approach, fade at the horizon distance, and twinkle subtly via time-based sin. Bright/close stars grow cross-shaped diffraction spikes in the fragment shader.
2. **Glow billboards** — an `InstancedMesh` of quads that always face the camera, with a multi-falloff radial glow (tight core, mid, wide halo) plus four-pointed and eight-pointed star spikes. This sells the "sun" look up close. Pulse the glow gently over time.
3. **Invisible hit spheres** — an `InstancedMesh` of unit spheres with an invisible material, only used as raycast targets so the laser knows which star was clicked. The instance index must map back to the star object so a hit can be identified.

A single directional `starLight` represents the current host star. As the nearest-star tracker swaps systems, move that light to the new star's position, tint it with the star's color, and aim it at the camera. A weak cool ambient adds a tiny floor of bluish fill so the unlit hemisphere of planets is not pitch black.

## Background sky

Three layered backgrounds, all parented to a group that follows the camera so they feel infinitely far:
- **Background star field** — thousands of small static points scattered on a huge sphere, colored by spectral class with subtle size variety.
- **Anchor stars** — a few dozen extra-bright pinpricks with their own twinkle shader, providing recognizable bright landmarks.
- **Milky Way skydome** — a back-side sphere with a fragment shader that paints a dark navy base gradient, a brighter galactic band along a tilted axis, FBM noise for dust clouds and dark lanes, and a warm bulge along one direction. No texture — pure shader math.

## Star systems — lazy planet generation

Generating planets for every star would blow the budget. Instead, on each tick find the nearest star to the camera. When it changes, deactivate the previous system (remove planets from scene) and activate the new one. If a system has never been visited it is built on the fly, deterministically seeded from the star's position.

A system has 4–8 planets and 2–5 asteroid belts. Each planet:
- Type — rocky, gas giant, or ice — with size matched to type.
- Orbit — incrementing radius from the star, orbiting on a random inclined plane (use a random orthonormal basis).
- Slow rotation around its own axis.

The home system always contains one *showcase planet*: a hand-tuned Saturn-like ringed gas giant placed in the second orbit slot so the very first frame after the intro shows a dramatic ringed giant.

### Planet surfaces

Paint vertex colors directly on the sphere geometry — no textures.

- **Gas** — banded latitudinal palette of 3 base colors. Add 3D noise turbulence so band boundaries are not perfect circles. Add storm spots at low latitudes where noise exceeds a threshold. Several palettes available (cream/brown, blue/navy, red, lavender).
- **Ice** — pale blue base with subtle noise variation, brighter at the poles.
- **Rocky** — ocean / land / desert / polar ice chosen by elevation noise and latitude. Most rocky planets have oceans, some are dry.

Crucial detail: when a planet is far enough away its mesh, atmosphere, clouds and rings should fade out smoothly via material opacity, not pop. Use a distance-based smoothstep, e.g. fully visible inside ~9k units and fully gone past ~22k. Otherwise the visible boundary of a system feels artificial.

### Atmospheres and clouds

Gas giants always, many rocky/ice planets sometimes — render an atmosphere shell: a slightly larger sphere with a custom shader on the back side, additive blending, fresnel-style rim term that pulses gently and shimmers from procedural hash noise. The lit hemisphere should be visibly brighter than the unlit side. On the showcase planet, add an "eclipse" boost where the rim near the terminator glows extra.

Gas giants additionally get a cloud shell — another slightly larger sphere with a procedural FBM noise shader giving moving wispy bands. Normal blending, not additive.

### Rings

Gas giants frequently, ice giants sometimes, large rocky planets rarely. Use a `RingGeometry` with custom UVs remapped so the texture stretches radially. The ring texture is procedurally drawn into a 2D canvas — radial density curve, several gaps (Cassini divisions), sine-wave bands modulating brightness, warm-to-cool color shift across radius. The ring shader projects a shadow from the planet body across the back half of the ring.

On top of the flat ring, instance thousands of tiny asteroid rocks distributed in the same plane with a slight thickness, each spinning on a per-instance random axis. Use `onBeforeCompile` shader injection to read a per-instance `aSpinAxis` attribute and animate rotation in the vertex shader.

### Moons

Each planet may have 0–3 moons. Different flavors: cratered grey, icy white, rusty orange, dusty red, blueish, sometimes faintly self-emissive. Paint mottled vertex colors on each moon and give it its own orbital plane around the planet.

### Asteroid belts

Per system, generate a few belts at varying orbital radii, each on a random plane. A belt is an `InstancedMesh` of icosahedron-derived asteroid geometries (slightly deformed per-vertex by noise so they are non-spherical). Sizes follow a long-tail distribution — mostly tiny, occasionally a few big chunks.

## Nebulae

Place a handful (~5) of large drifting nebulae scattered around the home region. Each nebula is built from:
- One or two large sprite layers using a procedurally drawn nebula texture in canvas (radial gradients composited additively — coloured core, secondary accent blobs, sparkle dots, dark punches for shape variety, then masked to a soft circle so the sprite has no hard square edge).
- A few bright satellite "embedded stars" as smaller sprites overlaid on the nebula cloud.

Use a rotating set of color schemes — pink/blue, orange/violet, teal/purple, golden/indigo.

## Black hole

Spawn one black hole as a landmark a few tens of thousands of units from home. Composed of:
- An accretion disc — a flat plane with a procedurally drawn disc texture (hot white inner ring fading to orange and magenta outer), additive blending, tilted away from camera-up.
- A second perpendicular disc plane for volume cue.
- A bright lens-flare-style sprite halo around it.
- A pure black core sprite that occludes everything behind, drawn at high render order.

## Sun positioning for shaders

Atmosphere, cloud and ring shaders all need to know the direction from the planet to its star to do correct day/night and shadow lighting. Each frame, recompute that direction per planet and push it into the relevant material uniforms along with the planet's world-space center and radius. The ring shader uses this to project the planet's shadow onto the back half of the ring.

## Flight controls

Pointer-lock mouse look for yaw/pitch. WASD for thrust + strafe. Q/E for roll. Space for brake. H to toggle HUD. LMB to fire laser.

Thrust is not linear — holding W ramps up via a piecewise curve so low speeds are precise and high speeds are fast. Display speed in units of *c* (lightspeed-multiples). Cap somewhere around 30c. Apply a soft auto-brake when very close to a planet so the player cannot fly through one at full speed.

Smooth mouse input by integrating delta then applying with an exponential time constant — avoid jitter and per-frame snapping. Clamp per-event mouse deltas so coalesced OS events do not cause enormous jumps.

Use a free-floating quaternion for orientation, not Euler angles. Applying yaw / pitch / roll as quaternion multiplications keeps the camera fully 6-DOF and avoids gimbal lock — the player should be able to barrel-roll arbitrarily and stay sane.

The first one or two mouse-move events after pointer lock is acquired carry garbage deltas from the lock transition. Drop them, otherwise the view will snap on click-to-start.

## Laser and destruction

Press LMB to fire a green laser. Visually it is a thin elongated additive-blended box attached to the camera, scaled to the hit distance, plus a soft glow sprite at the muzzle. Visible for ~180 ms then fades.

Raycast from the camera forward against both the star hit-mesh and all destructible planet/moon meshes. On hit:
- A star → triggers a **supernova**: massive multi-stage flash, multiple shockwave rings, expanding shells, hundreds of glowing fragments, light rays. The star is removed from the universe. Lasts many seconds.
- A planet or moon → triggers a smaller **explosion**: flash, ring, expanding sphere, dozens of rocky fragments, sparks, rays. The body and its dependents (atmosphere, clouds, rings, moons) are removed.

If the raycast misses everything, do an aim-assist sweep over nearby active stars within a narrow forward cone and snap to the closest one — at high speed even a perfect ray would be nearly impossible to land.

## Explosions

Implement explosions as composite records of many small ephemeral meshes with per-part lifetimes:
- Expanding additive sphere flashes
- Camera-facing rings expanding outward
- Translucent shock spheres rendered front-side only so the camera inside the bubble does not see a white-out
- Big chunks (icosahedron asteroids with emissive material) flying outward with random tumble
- Smaller debris chunks
- Particle sparks as a Points cloud with per-particle velocities
- Line-segment light rays that grow outward from the origin then trail away

Cap concurrent explosion records and dispose all geometries/materials on cleanup so GPU memory does not leak.

The supernova should look distinctly *bigger and more violent* than a planet explosion, not just a rescaled copy. Multiple flash colors hitting in sequence (white → warm → cool), nested shock spheres in different colors, more rays, longer duration. The first second should genuinely fill the screen.

## Audio (Web Audio API only)

All sounds are synthesized live.

- **Ambient pad** — three low detuned sawtooth oscillators through a lowpass, droning continuously at low volume for atmosphere.
- **Engine** — fat detuned sawtooths plus an octave-up square plus a high triangle whine plus filtered noise turbulence. Filter cutoff opens with throttle. An LFO modulates gain for a subtle "alive" wobble. The high whine only kicks in past ~30 % throttle.
- **Laser** — short bandpassed noise burst with descending pitch sweep plus a saw oscillator chirp through a lowpass.
- **Explosion** — long lowpass-filtered noise burst through a waveshaper for crunch, with a low sine sub-thump and several smaller crackles scheduled in the tail.
- **Supernova** — bigger, stereo, longer-tailed version of the explosion with a deep sub rumble.
- **Brake** — short noise sweep plus a low thump.
- **Thrust-on** — only on first press of W/S when stopped, a quick noise whoosh plus a rising sine.

The engine should react to throttle level in real time via `setTargetAtTime` smoothing, not retriggered on each keypress.

Audio context creation is gated behind user interaction (most browsers block autoplay), so create and start everything on the first click-to-start. A master gain node feeds everything, kept around 50 % so the supernova does not clip.

## HUD and overlay

A small monospace green HUD in the top-left:
- Controls hint line.
- Speed in *c* with a direction arrow and a small brake indicator when braking.
- Nearest body (star class or planet type) and distance in units.

A green `+` crosshair in the center. The title overlay (large "GALAXY" + small subtitle) is centered and visible only during the intro. The CLICK TO START button is shown near the bottom, becomes CLICK TO RESUME if pointer lock is dropped.

## Color palette and feel

Stars are spectral colors slightly desaturated by random multiplication so a field of stars does not look like a candy bowl. Planet palettes lean rich and saturated. Atmospheres are tinted toward the planet's class (warm cream/gold on the showcase, blue on water worlds, lavender on ice). The HUD and laser are a phosphor green (~`#22ff44`) reminiscent of an old CRT, with text-shadow glow.

Nothing should be pure white except the supernova flash core and the inner accretion ring. Default to colors hovering around 80–90 % brightness so bloom has room to do work.

## Postprocessing

EffectComposer chain:
- RenderPass.
- UnrealBloomPass — gentle (low strength, mid threshold, soft radius). Strong bloom looks cheap; subtle bloom looks expensive.
- A custom finish pass — slight contrast lift, warm/cool tonal split-toning (cool shadows, warm highlights), soft circular vignette.

## Intro behavior

Before the player clicks to start: keep the camera slowly orbiting the home star at the start position's radius, with a gentle vertical bob, looking toward the home star and the showcase ringed planet. Once pointer lock is acquired the title overlay and button hide and the normal `updateShip` flight controls take over.

## Render loop

A single `requestAnimationFrame` loop driven by `THREE.Clock`. Each frame:
- Update shader time uniforms (star shimmer, glow pulse, anchor twinkle, ring rock spin).
- If intro is active, drive the orbital camera; otherwise run the ship update.
- Re-evaluate active chunks against camera position.
- Pin the background-sky group to the camera position.
- Every ~10 frames, scan for the nearest star and swap active systems if it changed.
- Update active system orbits and shader uniforms.
- Tick laser timer and explosions.
- Refresh HUD text.
- `composer.render()`.

Cap `dt` at something like 50 ms so a tab-switch does not produce a huge time step that teleports everything.

## Performance and robustness

- Use `InstancedMesh` for everything you have hundreds or thousands of (stars, glows, ring rocks, belt rocks).
- Set `frustumCulled = false` on objects whose bounding sphere is wrong because they are billboarded or huge.
- Cache chunk results in a map so revisited regions do not regenerate.
- Run the expensive nearest-star scan only every few frames, not every frame.
- Dispose geometry and material when removing destroyed objects.
- Use one shared icosahedron geometry pool for all asteroid instances rather than a unique geometry per rock.

## Scale notes

Work in arbitrary world units, not real meters. A reasonable scale:
- Planet radii: tens to a couple hundred units.
- Star radii: tens to several hundred units.
- Planet orbits: hundreds to a few thousand units from the star.
- System scope: a few tens of thousands of units across.
- Chunk size: ~20k units.
- Far star horizon: ~380k units.
- Skydome radius: ~200k units.

These numbers do not need to be exact, but the *ratios* matter — a planet has to look small next to its star, and a star has to look small next to the gulf between chunks.

## Acceptance feel test

Open the file, click start, then:
- A ringed Saturn-like planet should be clearly visible from the first frame, with moons orbiting it.
- Looking around should reveal the Milky Way band overhead and colorful nebulae in the distance.
- Pushing W should accelerate with a satisfying engine sound that brightens as the throttle opens.
- Pointing at any star and clicking should fire a green beam and trigger a multi-second supernova that fills the view.
- Pointing at a planet and clicking should blow it apart into chunks and sparks, taking its moons and rings with it.
- Flying for several seconds in any direction should keep revealing new stars without the universe feeling empty or repeating.
- Coming back to the same coordinates later should show the same stars in the same places — the universe is deterministic.

Deliver the full galaxy.html file. No commentary, no markdown fences around it.
