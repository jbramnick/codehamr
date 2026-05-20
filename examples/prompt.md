# BUILD: `galaxy.html` — Procedural Space Game (v3, bug-hardened)

You are a senior Three.js engineer. Produce **one self-contained HTML file** named `galaxy.html`. Output **only the file content**, starting with `<!DOCTYPE html>`. No prose, no markdown fences.

This is v3. v2 produced: `ReferenceError: easyFade is not defined` (invented helper functions), asteroids made of triangle holes (vertex splitting), giant white-flash explosions (uncapped sizes), no deep-space mood (no tone mapping, too much ambient). Every section below is rewritten to make those bugs unrepresentable.

---

## 0. NEW STRICT RULES — READ FIRST

**A. NO UNDEFINED IDENTIFIERS.**
Every function and variable you call must be defined **in this file**. Do not invent helpers like `easyFade`, `lerp`, `randRange`, `disposeMesh`, `smoothStep` and assume they exist. Either inline the math or define them at the top of the script. Before writing any function call, mentally check: "is this defined above?" If not, write it.

**B. NO IMPORTS BEYOND THIS LIST.**
```js
import * as THREE from 'three';
import { EffectComposer } from 'three/addons/postprocessing/EffectComposer.js';
import { RenderPass } from 'three/addons/postprocessing/RenderPass.js';
import { UnrealBloomPass } from 'three/addons/postprocessing/UnrealBloomPass.js';
```
Do not import `BufferGeometryUtils`, `OrbitControls`, `GLTFLoader`, or anything else.

**C. NO `new` IN HOT PATHS.**
Inside `animate()` or any function it calls every frame, do not allocate `new Vector3`, `new Color`, `new Quaternion`. Allocate once at module scope, reuse with `.set()` / `.copy()`.

**D. PROVIDED HELPERS — USE THESE, DON'T REINVENT.**
Put these at the top of the script. Every later section references them by name.
```js
const TAU = Math.PI * 2;
const lerp = (a, b, t) => a + (b - a) * t;
const clamp = (v, a, b) => v < a ? a : (v > b ? b : v);
const randRange = (r, a, b) => a + r() * (b - a);
const randPick = (r, arr) => arr[Math.floor(r() * arr.length)];

// Stable position-based pseudo-noise — CRITICAL for asteroids.
// Two identical input positions always return the same value.
function posNoise(x, y, z) {
  const s = Math.sin(x*12.9898 + y*78.233 + z*37.719) * 43758.5453;
  return s - Math.floor(s);  // 0..1
}

// Temp vectors — reuse, never allocate in render loop.
const _v1 = new THREE.Vector3();
const _v2 = new THREE.Vector3();
const _v3 = new THREE.Vector3();
const _q1 = new THREE.Quaternion();
const _m1 = new THREE.Matrix4();
const _c1 = new THREE.Color();
```

---

## 1. HTML SKELETON — COPY EXACTLY

```html
<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<title>Galaxy</title>
<style>
  html, body { margin:0; padding:0; overflow:hidden; background:#000; height:100%; }
  canvas { display:block; }
  #hud {
    position:fixed; top:12px; left:12px;
    font-family:'Courier New', monospace;
    color:#22ff44; font-size:13px; line-height:1.5;
    text-shadow:0 0 6px #22ff44; pointer-events:none; user-select:none;
  }
  #crosshair {
    position:fixed; top:50%; left:50%; transform:translate(-50%,-50%);
    color:#22ff44; font-family:monospace; font-size:20px;
    text-shadow:0 0 6px #22ff44; pointer-events:none;
  }
  #click-to-start {
    position:fixed; top:50%; left:50%; transform:translate(-50%,-50%);
    color:#22ff44; font-family:monospace; font-size:18px;
    text-shadow:0 0 6px #22ff44;
    background:rgba(0,0,0,0.7); padding:16px 24px; border:1px solid #22ff44;
    cursor:pointer; z-index:10;
  }
</style>
<script type="importmap">
{ "imports": {
  "three": "https://unpkg.com/three@0.160.0/build/three.module.js",
  "three/addons/": "https://unpkg.com/three@0.160.0/examples/jsm/"
}}
</script>
</head>
<body>
<div id="hud"></div>
<div id="crosshair">+</div>
<div id="click-to-start">CLICK TO START</div>
<script type="module">
/* all code here */
</script>
</body>
</html>
```

---

## 2. RENDERER + TONE MAPPING — THIS IS HOW DEEP SPACE LOOKS

Without these settings, everything looks like a cheap WebGL demo. Use **exactly** these values:

```js
const renderer = new THREE.WebGLRenderer({
  antialias: true,
  logarithmicDepthBuffer: true,
  powerPreference: 'high-performance'
});
renderer.setPixelRatio(Math.min(devicePixelRatio, 2));
renderer.setSize(innerWidth, innerHeight);
renderer.setClearColor(0x000000);
renderer.toneMapping = THREE.ACESFilmicToneMapping;
renderer.toneMappingExposure = 0.55;             // dim for high contrast
renderer.outputColorSpace = THREE.SRGBColorSpace;
document.body.appendChild(renderer.domElement);

const scene = new THREE.Scene();
scene.fog = null;                                 // no fog — kills deep space feel
const camera = new THREE.PerspectiveCamera(70, innerWidth/innerHeight, 0.1, 200000);
scene.add(camera);                                // children of camera need this
```

---

## 3. LIGHTING — HIGH CONTRAST DEEP SPACE

```js
// Almost no ambient — space is black.
scene.add(new THREE.AmbientLight(0x0a0a18, 0.08));

// The active star is a powerful directional light.
const starLight = new THREE.DirectionalLight(0xffffff, 3.5);
starLight.position.set(0, 0, 0);
scene.add(starLight);
// In updateNearStars(), update starLight.position and starLight.color from the nearest star.
```

**Do not** add extra fill lights. Hard shadows from one direction is the look we want.

---

## 4. SCALES (unchanged from v2 — they work)

| Object | Size | Distance |
|---|---|---|
| Stars | radius 50–400 | spread in cube of ±40000, ~3000 stars |
| Planets | radius 5–80 | orbit 200–2000 from star |
| Moons | radius 1–15 | orbit 30–200 from planet |
| Asteroids | radius 0.3–4 | belts 500–5000 from star |
| Nebulae | size 3000–15000 | scattered |
| Camera near / far | 0.1 / 200000 | |

---

## 5. STARS — DATA + POINTS + LOD MESHES

(Same approach as v2: data array, points-shader for far, real meshes for nearest 6.)

Reuse v2 §5 structure. Key reminders:
- Point shader must clamp `gl_PointSize` between 2 and 10. **No exception.**
- Vary brightness: multiply each star's color by `(0.6 + r()*0.6)` so the field isn't uniformly bright.
- After generating, place the camera 800 units from a G/K-class star (guaranteed home).

For the **near-star mesh**: use `MeshBasicMaterial` with the star's color (not white) for the inner sphere, `corona` is a back-side sphere at scale 2.0 with fresnel additive shader. Bloom will glow them. **Do not** scale the inner sphere larger than the star's actual radius.

---

## 6. PLANETS — KEEP IT SIMPLE

Each planet: `SphereGeometry(radius, 48, 48)`, `MeshStandardMaterial({ color, roughness: 0.85, metalness: 0.0 })`. For visual variety, *modulate vertex colors* using position-noise:

```js
function makePlanet(r, starRadius, orbit) {
  const radius = randRange(r, 5, 80);
  const geom = new THREE.SphereGeometry(radius, 48, 48);
  // Add vertex color variation via position-based noise — bands and patches.
  const pos = geom.attributes.position;
  const colors = new Float32Array(pos.count * 3);
  const baseHue = r();
  const planetColor = new THREE.Color().setHSL(baseHue, 0.5, 0.4);
  for (let i = 0; i < pos.count; i++) {
    const x = pos.getX(i), y = pos.getY(i), z = pos.getZ(i);
    const n = posNoise(x*0.1, y*0.1, z*0.1);
    const c = planetColor.clone().multiplyScalar(0.6 + n*0.8);
    colors[i*3]   = c.r;
    colors[i*3+1] = c.g;
    colors[i*3+2] = c.b;
  }
  geom.setAttribute('color', new THREE.BufferAttribute(colors, 3));
  const mat = new THREE.MeshStandardMaterial({ vertexColors:true, roughness:0.9, metalness:0.0 });
  const mesh = new THREE.Mesh(geom, mat);
  return { mesh, orbit, phase: r()*TAU, speed: 0.05 + r()*0.1, radius };
}
```

Atmosphere (40% chance): a back-side sphere at 1.04× radius, `MeshBasicMaterial({ color: atmoColor, transparent:true, opacity:0.15, side:THREE.BackSide, blending:THREE.AdditiveBlending, depthWrite:false })`.

Rings (50% chance, only for planets with radius > 40): `RingGeometry(r*1.5, r*2.5, 64)` rotated `Math.PI/2` on X, double-sided, additive, alpha 0.3. **Do not** spawn asteroid instances inside rings in v3 — too expensive. The ring mesh alone reads as "rings."

---

## 7. ASTEROIDS — THE GEOMETRY FIX

**v2 failed because `IcosahedronGeometry` is non-indexed.** Each face has its own 3 vertices, not shared with neighbors. When you randomly displace vertices, every face moves independently → cracks.

**Fix: position-based noise.** Vertices that share the same position get the same displacement. This works on non-indexed geometry too. Use the `posNoise` from §0.D.

```js
function makeAsteroidGeometry(seed) {
  const r = mulberry32(seed);
  const detail = 1 + Math.floor(r()*2);          // 1 or 2 (low poly is fine)
  const geom = new THREE.IcosahedronGeometry(1, detail);
  const pos = geom.attributes.position;
  for (let i = 0; i < pos.count; i++) {
    const x = pos.getX(i), y = pos.getY(i), z = pos.getZ(i);
    // posNoise is deterministic in (x,y,z) — duplicate vertices get identical noise.
    const n1 = posNoise(x*3.0, y*3.0, z*3.0);
    const n2 = posNoise(x*7.0+5, y*7.0+5, z*7.0+5);
    const bump = 0.7 + n1*0.4 + n2*0.15;          // 0.7..1.25
    pos.setXYZ(i, x*bump, y*bump, z*bump);
  }
  pos.needsUpdate = true;
  geom.computeVertexNormals();                    // MANDATORY
  return geom;
}

// Build 10 unique shapes:
const asteroidGeoms = [];
for (let i = 0; i < 10; i++) asteroidGeoms.push(makeAsteroidGeometry(1000 + i));

// Single shared material:
const asteroidMat = new THREE.MeshStandardMaterial({
  color: 0x5a4838, roughness: 0.95, metalness: 0.05, flatShading: true
});
```

**Why this works:**
- Duplicate vertices (the ones at face borders that look the same to the eye) get the *same* `posNoise(x,y,z)` because the input is the same → they move together → no cracks.
- `flatShading:true` gives faceted look that hides any tiny gaps.

For belts:
```js
function makeBelt(starPos, innerR, outerR, count) {
  const geom = asteroidGeoms[Math.floor(Math.random()*asteroidGeoms.length)];
  const inst = new THREE.InstancedMesh(geom, asteroidMat, count);
  for (let i = 0; i < count; i++) {
    const a = Math.random()*TAU;
    const rr = innerR + Math.random()*(outerR-innerR);
    const yOff = (Math.random()-0.5) * (outerR-innerR) * 0.15;
    _v1.set(starPos.x + Math.cos(a)*rr, starPos.y + yOff, starPos.z + Math.sin(a)*rr);
    _q1.setFromAxisAngle(_v2.set(Math.random(),Math.random(),Math.random()).normalize(), Math.random()*TAU);
    const s = randRange(Math.random, 0.3, 3.5);
    _v3.set(s, s, s);
    _m1.compose(_v1, _q1, _v3);
    inst.setMatrixAt(i, _m1);
  }
  inst.instanceMatrix.needsUpdate = true;
  return inst;
}
```

Belts per active star: 1–2. Count per belt: 2000–4000.

---

## 8. NEBULAE — ALWAYS-ON

Spawn 8–12 nebulae at module load (not lazily). Reuse v2 §7 spawn logic.

For each nebula sprite, generate a `CanvasTexture` *once*, share across that nebula's sprites:

```js
function makeNebulaTexture(color1, color2, seed) {
  const r = mulberry32(seed);
  const size = 256;
  const cnv = document.createElement('canvas'); cnv.width = size; cnv.height = size;
  const ctx = cnv.getContext('2d');
  ctx.fillStyle = '#000'; ctx.fillRect(0,0,size,size);
  ctx.globalCompositeOperation = 'lighter';
  for (let i = 0; i < 40; i++) {
    const x = r()*size, y = r()*size;
    const rad = 20 + r()*70;
    const t = r();
    const c = (t < 0.5 ? color1 : color2);
    const grad = ctx.createRadialGradient(x,y,0, x,y,rad);
    grad.addColorStop(0, `rgba(${(c.r*255)|0},${(c.g*255)|0},${(c.b*255)|0},0.7)`);
    grad.addColorStop(1, `rgba(${(c.r*255)|0},${(c.g*255)|0},${(c.b*255)|0},0)`);
    ctx.fillStyle = grad;
    ctx.fillRect(x-rad,y-rad,rad*2,rad*2);
  }
  // edge fade
  ctx.globalCompositeOperation = 'destination-in';
  const fade = ctx.createRadialGradient(size/2,size/2,size*0.2, size/2,size/2,size*0.5);
  fade.addColorStop(0,'rgba(0,0,0,1)'); fade.addColorStop(1,'rgba(0,0,0,0)');
  ctx.fillStyle = fade; ctx.fillRect(0,0,size,size);
  const tex = new THREE.CanvasTexture(cnv);
  tex.colorSpace = THREE.SRGBColorSpace;
  return tex;
}
```

Sprite material: `SpriteMaterial({ map: tex, blending:THREE.AdditiveBlending, depthWrite:false, transparent:true, opacity:0.4 })`. Size 3000–8000. Cluster 40–120 per nebula.

Guarantee: first 3 nebulae spawn within 8000–16000 of home star.

---

## 9. CONTROLS — IDENTICAL TO v2 §9

Copy v2 §9 verbatim. It worked.

---

## 10. LASER — SIMPLE BOX MESH

v2 used `CylinderGeometry` with translate/rotate trickery. Drop it. Use a `BoxGeometry` — simpler, no orientation surprises:

```js
const laserGeom = new THREE.BoxGeometry(0.4, 0.4, 1);   // unit length on Z
const laserMat = new THREE.MeshBasicMaterial({
  color: 0x22ff44, transparent:true, opacity:0.95,
  blending:THREE.AdditiveBlending, depthWrite:false
});
const laser = new THREE.Mesh(laserGeom, laserMat);
laser.visible = false;
camera.add(laser);                          // child of camera — local coords

// Glow sprite under the laser
const laserGlowTex = (function() {
  const c = document.createElement('canvas'); c.width=64; c.height=64;
  const x = c.getContext('2d');
  const g = x.createRadialGradient(32,32,0, 32,32,32);
  g.addColorStop(0,'rgba(80,255,120,1)');
  g.addColorStop(0.4,'rgba(40,255,80,0.4)');
  g.addColorStop(1,'rgba(0,0,0,0)');
  x.fillStyle = g; x.fillRect(0,0,64,64);
  return new THREE.CanvasTexture(c);
})();
const laserGlow = new THREE.Sprite(new THREE.SpriteMaterial({
  map: laserGlowTex, blending:THREE.AdditiveBlending, depthWrite:false, opacity:0.9
}));
laserGlow.visible = false;
camera.add(laserGlow);

let laserTimer = 0;
const destructibles = [];                   // array, populated by buildSystem

function fireLaser() {
  const origin = _v1; camera.getWorldPosition(origin);
  const dir = _v2.set(0,0,-1).applyQuaternion(camera.quaternion);
  const ray = new THREE.Raycaster(origin, dir, 0, 100000);
  const hits = ray.intersectObjects(destructibles, false);
  let dist = 5000;
  if (hits.length > 0) {
    dist = hits[0].distance;
    explode(hits[0].point, hits[0].object);
    removeDestructible(hits[0].object);
  }
  // Place laser: forward from camera by dist/2, scale Z to dist
  laser.position.set(0, -1.5, -dist*0.5 - 2);
  laser.scale.set(1, 1, dist);
  laser.visible = true;
  laserGlow.position.set(0, -1.5, -3);
  laserGlow.scale.set(8, 8, 1);
  laserGlow.visible = true;
  laserTimer = 0.18;
}

function updateLaser(dt) {
  if (laserTimer > 0) {
    laserTimer -= dt;
    const a = clamp(laserTimer / 0.18, 0, 1);
    laser.material.opacity = a * 0.95;
    laserGlow.material.opacity = a * 0.9;
    if (laserTimer <= 0) { laser.visible = false; laserGlow.visible = false; }
  }
}
```

---

## 11. EXPLOSIONS — CAPPED, COMPLETE, DEFINED

v2 failed with `easyFade is not defined` and absurdly large explosions. Use these capped sizes and these helpers only.

### 11.1 Size caps (absolute, not multiplied)
```js
const EXP_FLASH_MAX = 200;            // never exceed
const EXP_RING_MAX = 600;
const EXP_SPHERE_MAX = 500;
const EXP_FRAG_BIG_DIST = 300;
const EXP_FRAG_SMALL_DIST = 600;
const EXP_SPARK_DIST = 400;
```
Inside `explode()`, derive *target* sizes from object radius but clamp to these.

### 11.2 Explosion record + pool
```js
const explosions = [];                // active records
const MAX_EXPLOSIONS = 6;

function explode(point, targetObj) {
  if (explosions.length >= MAX_EXPLOSIONS) {
    cleanupExplosion(explosions.shift());
  }
  const R = clamp(targetObj.geometry?.boundingSphere?.radius || 10, 1, 100);
  const rec = {
    t: 0,
    duration: 8.0,
    point: point.clone(),
    radius: R,
    parts: []
  };

  // (a) Flash sphere
  const flashSize = clamp(R * 3, 5, EXP_FLASH_MAX);
  const flashGeom = new THREE.SphereGeometry(1, 16, 16);
  const flashMat = new THREE.MeshBasicMaterial({
    color: 0xfff2aa, transparent:true, opacity:1,
    blending:THREE.AdditiveBlending, depthWrite:false
  });
  const flash = new THREE.Mesh(flashGeom, flashMat);
  flash.position.copy(point);
  flash.userData = { type:'flash', maxSize:flashSize, lifetime:0.5 };
  scene.add(flash); rec.parts.push(flash);

  // (b) Ring shockwave
  const ringSize = clamp(R * 12, 20, EXP_RING_MAX);
  const ringGeom = new THREE.RingGeometry(0.9, 1.0, 64);
  const ringMat = new THREE.MeshBasicMaterial({
    color: 0xffaa44, transparent:true, opacity:1, side:THREE.DoubleSide,
    blending:THREE.AdditiveBlending, depthWrite:false
  });
  const ring = new THREE.Mesh(ringGeom, ringMat);
  ring.position.copy(point);
  // Face camera
  ring.lookAt(camera.position);
  ring.userData = { type:'ring', maxSize:ringSize, lifetime:1.5 };
  scene.add(ring); rec.parts.push(ring);

  // (c) Sphere shockwave (back-side)
  const sphSize = clamp(R * 9, 15, EXP_SPHERE_MAX);
  const sphGeom = new THREE.SphereGeometry(1, 24, 24);
  const sphMat = new THREE.MeshBasicMaterial({
    color: 0x88bbff, transparent:true, opacity:0.6, side:THREE.BackSide,
    blending:THREE.AdditiveBlending, depthWrite:false
  });
  const sph = new THREE.Mesh(sphGeom, sphMat);
  sph.position.copy(point);
  sph.userData = { type:'sphere', maxSize:sphSize, lifetime:1.2 };
  scene.add(sph); rec.parts.push(sph);

  // (d) Big slow fragments — REAL closed meshes, not loose triangles
  const bigCount = 8 + Math.floor(Math.random()*12);
  for (let i = 0; i < bigCount; i++) {
    const geom = asteroidGeoms[Math.floor(Math.random()*asteroidGeoms.length)];
    const mat = new THREE.MeshStandardMaterial({
      color: 0x553322, emissive: 0xff5522, emissiveIntensity: 0.6,
      roughness: 0.9, flatShading: true
    });
    const m = new THREE.Mesh(geom, mat);
    const s = R * (0.15 + Math.random()*0.35);
    m.scale.set(s,s,s);
    m.position.copy(point);
    const dir = new THREE.Vector3(Math.random()-0.5, Math.random()-0.5, Math.random()-0.5).normalize();
    m.userData = {
      type:'bigFrag',
      vel: dir.multiplyScalar(2 + Math.random()*6),
      angVel: new THREE.Vector3((Math.random()-0.5)*1, (Math.random()-0.5)*1, (Math.random()-0.5)*1),
      lifetime: 10 + Math.random()*5
    };
    scene.add(m); rec.parts.push(m);
  }

  // (e) Medium fast fragments
  const medCount = 30 + Math.floor(Math.random()*30);
  for (let i = 0; i < medCount; i++) {
    const geom = asteroidGeoms[Math.floor(Math.random()*asteroidGeoms.length)];
    const mat = new THREE.MeshBasicMaterial({ color: 0xffaa44 });
    const m = new THREE.Mesh(geom, mat);
    const s = R * (0.04 + Math.random()*0.1);
    m.scale.set(s,s,s);
    m.position.copy(point);
    const dir = new THREE.Vector3(Math.random()-0.5, Math.random()-0.5, Math.random()-0.5).normalize();
    m.userData = {
      type:'medFrag',
      vel: dir.multiplyScalar(8 + Math.random()*20),
      angVel: new THREE.Vector3(Math.random()*4, Math.random()*4, Math.random()*4),
      lifetime: 3 + Math.random()*3
    };
    scene.add(m); rec.parts.push(m);
  }

  // (f) Sparks: THREE.Points with radial-burst velocities
  const sparkCount = 400;
  const sparkPos = new Float32Array(sparkCount*3);
  const sparkVel = new Float32Array(sparkCount*3);
  // 8 spike directions for burst look
  const spikes = [];
  for (let i = 0; i < 8; i++) {
    spikes.push(new THREE.Vector3(Math.random()-0.5, Math.random()-0.5, Math.random()-0.5).normalize());
  }
  for (let i = 0; i < sparkCount; i++) {
    sparkPos[i*3] = point.x; sparkPos[i*3+1] = point.y; sparkPos[i*3+2] = point.z;
    const sp = spikes[i % 8];
    const speed = 30 + Math.random()*70;
    // jitter around spike direction
    const jx = sp.x + (Math.random()-0.5)*0.4;
    const jy = sp.y + (Math.random()-0.5)*0.4;
    const jz = sp.z + (Math.random()-0.5)*0.4;
    const inv = 1/Math.hypot(jx,jy,jz);
    sparkVel[i*3]   = jx*inv*speed;
    sparkVel[i*3+1] = jy*inv*speed;
    sparkVel[i*3+2] = jz*inv*speed;
  }
  const sparkGeom = new THREE.BufferGeometry();
  sparkGeom.setAttribute('position', new THREE.BufferAttribute(sparkPos, 3));
  const sparkMat = new THREE.PointsMaterial({
    color: 0xffee88, size: 2.5, transparent:true, opacity:1,
    blending:THREE.AdditiveBlending, depthWrite:false
  });
  const sparks = new THREE.Points(sparkGeom, sparkMat);
  sparks.userData = { type:'sparks', velocities: sparkVel, lifetime: 1.5 };
  scene.add(sparks); rec.parts.push(sparks);

  explosions.push(rec);
}
```

### 11.3 Update function — fully defined, no missing helpers

```js
function updateExplosions(dt) {
  for (let i = explosions.length - 1; i >= 0; i--) {
    const rec = explosions[i];
    rec.t += dt;
    let allDead = true;
    for (const p of rec.parts) {
      const ud = p.userData;
      if (!ud || ud.dead) continue;
      ud.lifetime -= dt;
      if (ud.lifetime <= 0) {
        p.visible = false; ud.dead = true; continue;
      }
      allDead = false;
      switch (ud.type) {
        case 'flash': {
          const a = 1 - (1 - ud.lifetime/0.5);
          const s = ud.maxSize * (1 - ud.lifetime/0.5);
          p.scale.set(s,s,s);
          p.material.opacity = clamp(ud.lifetime/0.5, 0, 1);
          break;
        }
        case 'ring': {
          const prog = 1 - ud.lifetime/1.5;
          const s = ud.maxSize * prog;
          p.scale.set(s,s,s);
          p.lookAt(camera.position);
          p.material.opacity = (1 - prog) * 0.9;
          const c = _c1.setRGB(1, 1-prog*0.6, 0.3 - prog*0.3);
          p.material.color.copy(c);
          break;
        }
        case 'sphere': {
          const prog = 1 - ud.lifetime/1.2;
          const s = ud.maxSize * prog;
          p.scale.set(s,s,s);
          p.material.opacity = (1 - prog) * 0.5;
          break;
        }
        case 'bigFrag':
        case 'medFrag': {
          p.position.addScaledVector(ud.vel, dt);
          p.rotation.x += ud.angVel.x * dt;
          p.rotation.y += ud.angVel.y * dt;
          p.rotation.z += ud.angVel.z * dt;
          break;
        }
        case 'sparks': {
          const pos = p.geometry.attributes.position.array;
          const vel = ud.velocities;
          for (let k = 0; k < pos.length; k++) pos[k] += vel[k] * dt;
          p.geometry.attributes.position.needsUpdate = true;
          p.material.opacity = clamp(ud.lifetime/1.5, 0, 1);
          break;
        }
      }
    }
    if (allDead || rec.t > rec.duration + 8) {
      cleanupExplosion(rec);
      explosions.splice(i, 1);
    }
  }
}

function cleanupExplosion(rec) {
  for (const p of rec.parts) {
    scene.remove(p);
    if (p.geometry && p.userData && p.userData.type === 'sparks') p.geometry.dispose();
    if (p.material && p.userData && p.userData.type !== 'bigFrag') p.material.dispose();
  }
}

function removeDestructible(obj) {
  const idx = destructibles.indexOf(obj);
  if (idx !== -1) destructibles.splice(idx, 1);
  if (obj.parent) obj.parent.remove(obj);
}
```

**Every identifier above is defined.** No `easyFade`, no `disposeMesh`, no `smoothStep`. If you want to add anything, write the function body.

---

## 12. POST-PROCESSING — RESTRAINED

```js
const composer = new EffectComposer(renderer);
composer.addPass(new RenderPass(scene, camera));
const bloom = new UnrealBloomPass(new THREE.Vector2(innerWidth, innerHeight), 0.5, 0.5, 0.95);
composer.addPass(bloom);
```

Strength 0.5, threshold 0.95 — only the brightest pixels bloom (stars, laser, explosion cores). Combined with `toneMappingExposure = 0.55` this gives moody contrast instead of overall haze.

---

## 13. RENDER LOOP

```js
const clock = new THREE.Clock();
let frame = 0;
function animate() {
  const dt = Math.min(clock.getDelta(), 0.05);
  frame++;
  updateShip(dt);
  if (frame % 10 === 0) updateNearStars();
  updateActiveSystems(performance.now() * 0.001);
  updateLaser(dt);
  updateExplosions(dt);
  updateHUD();
  composer.render();
  requestAnimationFrame(animate);
}
animate();
```

---

## 14. ANTI-BUG CHECKLIST — VERIFY BEFORE OUTPUT

- [ ] No `easyFade`, `lerp` (use the one defined in §0.D), `randRange`, or any other helper that isn't in §0.D or defined in your file
- [ ] **Every function call** has a matching definition in the file
- [ ] Asteroid geometry uses `posNoise(x,y,z)` for displacement (deterministic per position)
- [ ] `computeVertexNormals()` called after every vertex displacement
- [ ] `flatShading: true` on asteroid material
- [ ] `toneMapping = ACESFilmicToneMapping` and `toneMappingExposure = 0.55`
- [ ] `AmbientLight` intensity ≤ 0.1
- [ ] `DirectionalLight` from active star with intensity 3.5
- [ ] Bloom strength 0.5, threshold 0.95
- [ ] Laser is `BoxGeometry`, child of camera (`camera.add(laser)`)
- [ ] `scene.add(camera)` is called
- [ ] All explosion size variables are capped via `clamp()` against the constants in §11.1
- [ ] Explosion fragments use `asteroidGeoms` (real closed meshes), not loose `BufferGeometry`
- [ ] Sparks use `THREE.Points` with `additive` blending and `depthWrite:false`
- [ ] No `new Vector3` / `new Color` inside `animate()` or anything it calls per-frame
- [ ] Pointer-lock requested on user click only
- [ ] Camera placed near a G/K star after star generation
- [ ] 8–12 nebulae spawned at startup, first 3 within 16000 of home star
- [ ] HUD shows speed in `c` units, updates each frame
- [ ] No `console.error` on load

---

## 15. OUTPUT

Output the entire `galaxy.html` content, starting with `<!DOCTYPE html>`, ending with `</html>`. No code fences, no commentary, no explanation.