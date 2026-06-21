# ContainerScope — UI & UX Specification (3D / Layered Edition)

> The UI **is** the product. This is a multi-tenant, real-time, **3D** visualization of
> infrastructure topology, live network packet flow, and security posture. It must read as a
> premium, living, spatial system — not a flat node-graph. Think "command deck for your
> infrastructure," rendered in WebGL.

This spec is research-backed and implementation-ready. It pins exact libraries, performance
techniques, and a layered visual model so an LLM/engineer can build it without guessing.

---

## 0. Design north star

A dark, cinematic 3D space where:
- **Containers/pods are 3D objects** floating in space, grouped into spatial clusters.
- **Networks are translucent layered planes/zones** stacked in depth (the "layering" you want).
- **Packets are real moving particles** flowing along 3D links — speed/density = throughput,
  color = protocol, direction = arrow/flow.
- **Resource pressure is physical**: hot containers glow, pulse, and grow; idle ones dim and shrink.
- **Security is spatial**: vulnerable nodes carry glowing severity halos; blocked network paths
  render as severed/red links.

The whole thing should feel like you can *fly through your cluster*.

---

## 1. The 3D rendering stack (decided, not optional)

| Concern | Choice | Why |
|---|---|---|
| 3D engine | **Three.js** | The standard for production WebGL on the web |
| Graph layout in 3D | **`3d-force-graph`** (Vasco Asturiano) | Purpose-built: force-directed 3D graph on Three.js, supports d3-force-3d / ngraph physics, custom node objects, curved bezier links, particle-on-link animation out of the box |
| React integration | **react-three-fiber (R3F)** + **Drei** | Declarative Three.js in React; Drei gives `<Detailed>` (LOD), `<Instances>`, controls, postprocessing helpers |
| Packet particles | **`THREE.InstancedMesh`** (via R3F `<Instances>`) | One draw call for thousands of particles. **Non-negotiable** — one mesh per packet kills the GPU (10k separate meshes ≈ 5fps; instanced ≈ smooth) |
| Heavy particle motion | **GPU shaders** (custom vertex shader / GPGPU) when particle counts get large | Updating JS attribute arrays every frame is the expensive path; move motion to the GPU |
| Post-processing | **postprocessing** lib (bloom, depth-of-field, vignette) | The "glow" and cinematic feel; bloom on hot/critical nodes |
| Charts/sparklines | **D3 / visx** (2D overlay, HTML/SVG) | Charts stay 2D in the side panel + as billboard sprites |
| 2D fallback graph | **Cytoscape.js** | Accessibility / low-power / "flatten to 2D" toggle |

> `3d-force-graph` already supports **directional particles on links** (`linkDirectionalParticles`,
> `linkDirectionalParticleSpeed`, `linkDirectionalParticleWidth`) — use this for the MVP packet
> flow, then graduate to a custom instanced/shader particle system for scale and richer protocol
> coloring.

### Architecture decision
- **MVP (fast):** Use `3d-force-graph`'s built-in node objects + directional particles. Wrap it in
  React. This gets a real animated 3D topology live quickly.
- **Scale (later):** Drop to raw R3F + Three.js for full control: instanced packet particles,
  custom shaders, layered planes, LOD. Keep the same data contract so the swap is invisible.

---

## 2. The layered spatial model (your "UI layering")

Render infrastructure as **stacked translucent depth-layers** (Z-axis), like floors in a building.
Camera can orbit, fly, and "explode" the layers apart.

```
   Z+  ┌─────────────────────────────────────┐   Layer 3: INGRESS / EXTERNAL
       │  internet · LB · ingress · DNS      │   (where traffic enters/leaves)
       └─────────────────────────────────────┘
       ┌─────────────────────────────────────┐   Layer 2: SERVICE MESH
       │  services · virtual IPs · policies  │   (k8s services, edges/policies)
       └─────────────────────────────────────┘
       ┌─────────────────────────────────────┐   Layer 1: WORKLOADS
       │  pods / containers (the 3D objects) │   (the main floating cluster)
       └─────────────────────────────────────┘
   Z-  ┌─────────────────────────────────────┐   Layer 0: INFRASTRUCTURE
       │  hosts / nodes (platforms/slabs)    │   (physical/VM hosts as ground)
       └─────────────────────────────────────┘
```

- **Namespaces / Docker networks** = translucent **zone volumes** (rounded boxes / convex hulls)
  that contain their nodes; color-tinted, glassmorphic, with a soft glowing boundary.
- **Hosts/nodes** = ground **slabs** at the bottom layer; containers visually "sit on" their host
  (a faint tether line to its host slab).
- **Vertical links** between layers show ingress→service→pod→host paths.
- **Layer toggles**: show/hide each layer; **"explode view"** animates layers apart along Z so the
  user can see depth; **"flatten"** collapses to a single plane (or to the 2D Cytoscape fallback).

---

## 3. Node design (containers / pods as 3D objects)

Each node is a custom Three.js object, not a flat circle:

- **Shape encodes kind**: container = rounded cube; pod = grouped cluster of small cubes inside a
  translucent shell; service = ring/torus; external endpoint = distant glowing orb; host = slab.
- **Size encodes weight**: scale by resource footprint (e.g., memory) so big workloads read as big.
- **Color/material encodes state**: running = brand-cool tint; stopped = desaturated/grey;
  unhealthy/restarting = amber pulse; crashloop = red strobe.
- **Heat encodes load**: emissive intensity ∝ CPU; high-CPU nodes **glow and gently pulse**
  (drive bloom). This is the "CPU heatmap" rendered physically.
- **Halos encode security**: a ring/aura around the node colored by worst CVE severity
  (Critical = red glow, High = orange, Medium = yellow), with a small floating badge count.
- **Sparkline billboards**: a small always-facing-camera sprite above each node showing a live
  CPU/mem micro-chart (toggleable; auto-hidden at distance via LOD).
- **Selection**: selected node lifts slightly, brightens, spawns an orbiting highlight ring, and
  dims the rest of the scene (focus mode).

### Level of Detail (LOD) — required for performance
Use Drei `<Detailed distances={[0, 50, 100]}>`:
- **Near:** full mesh + label + sparkline + halo.
- **Mid:** mesh + halo, no sparkline, simplified label.
- **Far:** low-poly billboard sprite, no text. (LOD can recover 30–40% FPS in big scenes.)

---

## 4. Link & packet-flow design (the headline visual)

Links are 3D bezier curves (so parallel links don't overlap). Packets flow along them.

- **Direction:** particles travel src→dst (bidirectional = two streams).
- **Throughput:** particle **density + speed** ∝ bytes/sec on that flow. A busy edge is a bright
  river; an idle edge is a faint thread.
- **Protocol = color:** TCP / UDP / HTTP / gRPC / DNS / SQL each get a distinct hue (legend in UI).
- **Link thickness:** ∝ sustained bandwidth; **latency:** longer/slower particles or a subtle
  color shift for high-latency edges.
- **Events:** a dropped/blocked connection flashes red and the particle stream "shatters" at the
  blockage point (used by the NetworkPolicy simulator).

### Implementation path
1. **MVP:** `3d-force-graph` directional particles (`linkDirectionalParticles*`) keyed to flow data.
2. **Scale:** Replace with an **InstancedMesh particle pool** — preallocate N particles, recycle
   them along active edges; advance positions in a **GPU shader** (animated offset along the
   curve) to keep 60fps with thousands of in-flight packets. Color via per-instance attribute =
   protocol. Reference techniques: instanced rendering (single draw call), dashed-line animated
   offset, and shader-driven position updates.
3. **Budget:** target smooth 60fps at **500–1000 nodes** and **5–10k concurrent packets**.

---

## 5. Camera, navigation & interaction

- **Orbit / pan / zoom** (default), plus a **"fly" mode** (WASD + pointer) to move through the space.
- **Click node** → focus + slide-in side panel.
- **Click link** → flow detail (proto breakdown, bytes, latency, endpoints).
- **Double-click** → zoom-to-fit on that node and its neighbors; dim everything else.
- **Hover** → tooltip + highlight the node's links and neighbors (rest fade).
- **Right-click** → context menu (logs, restart/stop where allowed, scan now, pin).
- **Search/command palette (⌘K)** → jump-to-node, run actions, switch views.
- **Minimap** (corner) showing camera position within the full topology.
- **Time scrubber** (bottom) for time-travel replay of topology + flows (drives the whole scene).

---

## 6. App shell & layout

```
┌───────────────────────────────────────────────────────────────────────────┐
│  Top bar:  Org switcher ▸ Cluster/Connection picker   |  ⌘K  |  alerts  | me │
├───────┬───────────────────────────────────────────────────────┬───────────┤
│ Left  │                                                       │  Right    │
│ nav   │              3D TOPOLOGY VIEWPORT                     │  side     │
│       │      (the WebGL canvas — the hero)                    │  panel    │
│ •Topo │                                                       │ (node/    │
│ •Dash │   [layer toggles]            [legend / protocol key]  │  link     │
│ •Sec  │   [explode] [flatten] [2D]   [heatmap: CPU|Mem|Sec]   │  detail)  │
│ •Flows│                                                       │           │
│ •Set  │   ◐ minimap                       time scrubber ⏱     │           │
└───────┴───────────────────────────────────────────────────────┴───────────┘
```

- **Left nav:** Topology · Dashboards · Security · Flow Explorer · Settings.
- **Hero viewport:** the 3D canvas, full-bleed, with floating glass control clusters (bottom-left
  = view controls, bottom-right = legend, bottom-center = time scrubber).
- **Right side panel:** slides in on selection; tabs for Overview · Metrics · Security · Logs.
- **Flow Explorer:** a dockable bottom drawer — filterable data grid of live flows that
  cross-highlights the 3D graph (select a row → its edge lights up in 3D).

---

## 7. Visual language (premium aesthetic)

- **Mode:** dark-first, cinematic. Deep near-black background with subtle depth fog so distant
  nodes recede.
- **Palette:** one cool brand accent (e.g., electric cyan/violet) for healthy/active, warm
  spectrum (amber→red) reserved for heat & severity so "hot = important" reads instantly. Keep the
  base neutral; let glow do the talking. Avoid flat default web colors.
- **Material:** glassmorphic panels (blurred translucent surfaces), thin luminous borders, soft
  inner shadows. Zone volumes are frosted glass.
- **Glow:** selective **bloom** post-processing on emissive elements (hot nodes, active packets,
  critical halos). Don't bloom everything — reserve it as a signal.
- **Typography:** **Inter** (UI) or **Outfit** (display headers); tabular numerals for metrics.
  Never browser defaults.
- **Motion:** everything eases (no hard cuts). Nodes spring in/out; camera moves are damped;
  panels slide; hovers lift. Micro-animations on every state change so the UI feels alive.
- **Empty/loading states:** a calm animated particle field + skeleton panels, never a blank screen.

---

## 8. Real-time behavior

- **WebSocket-driven:** topology deltas and flow updates push live.
- **Node lifecycle:** container start → node springs into the scene (scale 0→1 + glow flash) within
  1–2s; stop → node fades, drifts down, and dissolves.
- **Live metrics:** node emissive heat + sparkline update on each metric tick (throttle to ~1–2 Hz
  for the heat, keep particle animation at 60fps independently).
- **Reconnect:** on WS drop, show a subtle "reconnecting" pill; on resync, reconcile without ghost
  nodes (diff against authoritative snapshot).
- **Decouple sim from render:** physics/layout tick and packet animation run on the render loop
  (`requestAnimationFrame` / R3F `useFrame`), independent of data arrival, so the scene never stutters.

---

## 9. Heatmap & overlay modes (toggleable)

- **CPU heatmap:** recolor all nodes on a cool→hot ramp by CPU%; hottest pulse + bloom.
- **Memory heatmap:** same ramp by memory pressure (vs limit for k8s).
- **Security heatmap:** recolor by worst CVE severity; safe = dim, critical = red beacon.
- **Network heatmap:** thicken/brighten edges by throughput; quiet edges fade.
- **Blast-radius mode:** select a node → highlight everything it can reach (and what reaches it).

---

## 10. Security visualization

- **Severity halos + badges** on nodes (Critical/High/Medium/Low), count floating beside the node.
- **Click a vulnerable node → Security tab:** CVE list (id, severity, package, installed→fixed,
  CVSS, fixable), SBOM download, "scan now."
- **NetworkPolicy simulator:** pick a policy → the 3D graph **paints allowed links green and blocked
  links red with a severed/shattered particle effect** at the cut point. Toggle "what-if" edits.
- **Misconfig markers:** privileged/root/exposed nodes get a warning glyph orbiting the node.
- **Org/connection security score:** a glanceable gauge in the top bar; trend sparkline on hover.

---

## 11. Performance budget & techniques (hard requirements)

- **Target:** 60fps at 500–1000 nodes; usable (≥30fps) at ~2000.
- **Draw calls:** keep low. **InstancedMesh/BatchedMesh** for all repeated geometry (nodes of the
  same kind, all packets). Share materials. Audit with `renderer.info.render.calls`.
- **LOD:** Drei `<Detailed>` for nodes; cull labels/sparklines at distance.
- **Particles:** instanced + shader-driven; cap concurrent particles and recycle from a pool.
- **GPU memory hygiene:** explicitly dispose geometries/materials/textures on unmount (Three.js
  leaks otherwise → tab crash after long sessions).
- **Adaptive quality:** drop pixel ratio / disable bloom / reduce particles automatically under
  load (R3F `regress()` / performance scaling).
- **Tooling:** ship with `stats-gl` (FPS/CPU/GPU), keep a dev `lil-gui` for tuning, profile with
  Spector.js. Use `three-mesh-bvh` for fast raycasting/picking in dense scenes.
- **Mobile/low-power:** auto-detect and offer the **2D Cytoscape** fallback; touch-friendly orbit.

---

## 12. Component structure (frontend)

```
/frontend/src
  /scene
    Viewport3D.tsx            # R3F <Canvas>, camera, controls, postprocessing
    GraphLayout.ts            # 3d-force-graph OR d3-force-3d driver (data → positions)
    nodes/
      ContainerNode.tsx       # instanced container/pod meshes + LOD
      ServiceNode.tsx
      HostSlab.tsx
      ZoneVolume.tsx          # translucent namespace/network volumes
      NodeHalo.tsx            # severity aura
      SparklineBillboard.tsx  # camera-facing micro-chart sprite
    flow/
      PacketSystem.tsx        # InstancedMesh particle pool + shader motion
      LinkCurves.tsx          # bezier link geometry
    overlays/
      Legend.tsx  Minimap.tsx  ViewControls.tsx  TimeScrubber.tsx
    effects/
      Bloom.tsx  Fog.ts  Materials.ts
  /panels
    SidePanel.tsx  (Overview | Metrics | Security | Logs tabs)
    FlowExplorer.tsx          # filterable flow grid, cross-highlights 3D
  /shell
    AppShell.tsx  LeftNav.tsx  OrgSwitcher.tsx  CommandPalette.tsx
  /charts
    Sparkline.tsx  MetricChart.tsx (D3/visx)
  /data
    useTopologyStream.ts  useMetrics.ts  useFlows.ts  (WS + REST hooks)
  /fallback
    Graph2D.tsx               # Cytoscape.js flat view
```

Keep a **single data contract** (`nodes[], links[], flows[], findings[]`) so the MVP
`3d-force-graph` driver and the later raw-R3F driver are interchangeable.

---

## 13. Accessibility & engineering hygiene

- **Routing:** client-side routes per view + deep-linkable selected node (`/topology?node=…`).
- **2D + reduced-motion:** the Cytoscape fallback and a `prefers-reduced-motion` mode (freeze
  particles, disable camera auto-motion) are **required**, not optional — WebGL-only is exclusionary.
- **Keyboard:** full keyboard nav for panels, command palette, and node cycling; focus rings.
- **Semantic HTML** for all 2D chrome (nav, panels, tables, dialogs); ARIA on interactive overlays.
  The canvas gets an accessible text summary / data-table equivalent of the current view.
- **Testing hooks:** every interactive element has a stable `data-testid`; expose a
  `window.__scene` debug handle (dev only) so e2e tests can assert node/edge state without pixels.
- **Color independence:** never rely on color alone — pair severity/protocol color with
  shape/glyph/label (color-blind safe).

---

## 14. Build order for the UI (maps onto the main spec's chunks)

1. **App shell + 2D Cytoscape graph** (Chunk 6) — prove data + interaction in 2D first.
2. **Swap in `3d-force-graph`** with real nodes + built-in directional particles (Chunk 6→7) —
   first 3D + live-update milestone.
3. **Layered zones + host slabs + node materials/heat** (Chunk 9, with metrics) — the spatial model.
4. **Custom InstancedMesh/shader packet system + LOD + bloom** (Chunk 13) — the "wow," at scale.
5. **Security halos + NetworkPolicy simulator** (Chunks 16–17).
6. **Time-travel scrubber** (Chunk 23).

Each step must hold the performance budget (Section 11) before moving on.

---

## 15. Definition of "WOW" (acceptance for the hero view)

- You open an org, the camera eases into a dark 3D space, namespaces glow as frosted layers, and
  containers float in clusters above their host slabs.
- Real traffic streams as colored particles between nodes; a busy service is visibly a bright river.
- A container spins up → it springs into the scene live; you `stress` it → it heats up red and pulses.
- A node with a critical CVE wears a red halo; clicking it slides in its CVE list.
- You hit "explode" → the layers separate in depth; "flatten" → it collapses to a clean 2D map.
- It stays at 60fps the whole time on a 500+ node cluster.
