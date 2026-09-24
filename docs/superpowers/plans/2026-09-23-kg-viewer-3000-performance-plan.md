# 3D Knowledge Graph 3000+ Performance Optimization Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Reduce Three.js object count, per-frame CPU/GPU uploads, draw calls, and raycast work while preserving graph interaction and adding amount-proportional ingredient visuals.

**Architecture:** Keep the viewer as a single embedded HTML module. Replace per-link Lines and per-node hitboxes with shared `LineSegments` and a second `InstancedMesh`; make node/link updates dirty-driven; add pure inline helpers for amount normalization and label candidate selection so behavior can be regression-checked through the embedded page tests.

**Tech Stack:** Go 1.26, Gin, embedded HTML, Three.js 0.160.0, WebGL2-compatible shader attributes, Node syntax checking, Go tests.

**Spec:** `docs/superpowers/specs/2026-09-23-kg-viewer-3000-performance-design.md`

## Global Constraints

- Preserve the existing `/kg-viewer`, `/v1/graph/kg`, `/v1/node/detail`, offline snapshot, search, click, hover, focus, Esc, and reset flows.
- Keep Three.js pinned to `0.160.0` and do not add runtime dependencies.
- Work only in `/Users/mac/.codex/worktrees/kg-viewer-3000-optimize/3D知识图谱kg-viewer` on `optimize/kg-viewer-3000`.
- Do not modify the user's existing uncommitted changes in the original checkout.
- Every production behavior change must have a regression assertion before implementation and a failing test observed first.

### Task 1: Establish optimization artifacts and static regression seams

**Files:**
- Create: `docs/superpowers/specs/2026-09-23-kg-viewer-3000-performance-design.md`
- Create: `docs/superpowers/plans/2026-09-23-kg-viewer-3000-performance-plan.md`
- Modify: `internal/viewer/viewer_test.go`

**Interfaces:**
- Produces static assertions for the shared node, hitbox, and relationship render paths that later tasks must satisfy.

- [x] **Step 1: Write the design and plan documents.** Include node counts, object-count bottlenecks, amount encoding, dirty updates, label LOD, shader line batching, testing, and non-goals.
- [ ] **Step 2: Add failing static assertions.** Extend `TestHandler_ServesEmbeddedHTML` to require `hitboxInstances`, `THREE.LineSegments`, `lineOpacity`, `lineBirth`, `parseAmountMg`, `applyRecipeAmountVisuals`, `MAX_VISIBLE_LABELS`, and dirty flags, and to reject `new THREE.Line(`, per-node hitbox creation, and unconditional per-frame `setColorAt`/`setMatrixAt` calls.
- [ ] **Step 3: Run the targeted test and verify RED.** Run `go test ./internal/viewer -run TestHandler_ServesEmbeddedHTML -count=1`; it must fail because the new structures are absent.
- [ ] **Step 4: Commit the approved design artifacts and RED test.** Run `git add docs/superpowers internal/viewer/viewer_test.go && git commit -m "docs(viewer): plan 3000 node performance optimization"`.

### Task 2: Add amount parsing and recipe visual-state tests

**Files:**
- Modify: `internal/viewer/viewer_test.go`
- Modify: `internal/viewer/index.html`

**Interfaces:**
- `parseAmountMg(value) -> number|null`: normalize common mass units and average numeric ranges.
- `amountVisualFactor(share, maxShare) -> number`: return a bounded scale factor.
- `applyRecipeAmountVisuals(productId)`: update ingredient `renderScale` and `renderColor` from direct Product→Ingredient amounts.

- [ ] **Step 1: Add failing assertions for amount helper behavior.** Assert the page contains unit cases (`kg`, `g`, `mg`, `mcg`, `μg`), range handling, null fallback, `amountShare`, and the bounded visual factor constants.
- [ ] **Step 2: Run the targeted test and verify RED.** Run `go test ./internal/viewer -run TestHandler_ServesEmbeddedHTML -count=1`; it must fail on the missing helper strings.
- [ ] **Step 3: Implement pure amount helpers.** Parse the first numeric token or a two-number range, normalize units to mg, return null for missing/non-numeric values, and keep the original display string untouched.
- [ ] **Step 4: Implement recipe visual state.** Build a direct-child ingredient list from `linkRecords`, compute valid shares, set `n.renderScale` and `n.renderColor`, and restore `n.baseColor`/`n.baseSize` when no recipe is active.
- [ ] **Step 5: Run the targeted test and verify GREEN.** Re-run the targeted viewer test and confirm it passes.
- [ ] **Step 6: Commit the amount-visual behavior.** Run `git add internal/viewer/index.html internal/viewer/viewer_test.go && git commit -m "feat(viewer): encode recipe ingredient amounts visually"`.

### Task 3: Replace per-node hitboxes with one instanced picker

**Files:**
- Modify: `internal/viewer/index.html`
- Modify: `internal/viewer/viewer_test.go`

**Interfaces:**
- `hitboxInstances`: shared transparent `InstancedMesh` with `userData.instanceIds`.
- `nodeIdFromIntersection(intersection)`: resolve either主体 or hitbox `instanceId`, reject invisible nodes.

- [ ] **Step 1: Add failing assertions for one hitbox mesh.** Require `new THREE.InstancedMesh(hitboxGeometry`, `hitboxInstances.userData.instanceIds`, and `raycaster.intersectObject(hitboxInstances, false)`; reject `new THREE.SphereGeometry(1, 12, 8)` inside the node loop.
- [ ] **Step 2: Run the targeted test and verify RED.** Run `go test ./internal/viewer -run TestHandler_ServesEmbeddedHTML -count=1` and confirm the expected missing-batch failure.
- [ ] **Step 3: Implement the shared hitbox mesh.** Create one low-resolution sphere and transparent material, populate one instance matrix per node, and map hitbox instance indices to node IDs.
- [ ] **Step 4: Update picking and visibility.** Filter both instance meshes through `instanceVisible`; toggle hitbox instance scale/visibility state with the same dirty node sync used by主体 instances.
- [ ] **Step 5: Run viewer tests and syntax checks.** Run `go test ./internal/viewer -count=1` and the inline-module extraction check.
- [ ] **Step 6: Commit.** Run `git add internal/viewer/index.html internal/viewer/viewer_test.go && git commit -m "perf(viewer): batch node hitboxes"`.

### Task 4: Replace per-link objects with shader-backed LineSegments

**Files:**
- Modify: `internal/viewer/index.html`
- Modify: `internal/viewer/viewer_test.go`

**Interfaces:**
- `linkRecords`: lightweight relation records `{a,b,birth,index}` for data and info-card lookup.
- `linkSegments`: one `THREE.LineSegments` with `position`, `color`, `lineOpacity`, and `lineBirth` attributes.
- `syncLinkVisibility(focus, reachable)`: update opacity attributes only when focus/hidden-type state changes.

- [ ] **Step 1: Add failing assertions for LineSegments and shader attributes.** Require the shared attributes and reject `new THREE.Line(`, `linkObjs.push`, and per-link `mat.opacity` writes.
- [ ] **Step 2: Run the targeted test and verify RED.** Run `go test ./internal/viewer -run TestHandler_ServesEmbeddedHTML -count=1`.
- [ ] **Step 3: Implement shared geometry.** Allocate typed arrays once from valid links, write paired endpoints/colors/birth values, and create one `LineSegments` object.
- [ ] **Step 4: Implement the shader.** Pass vertex color, opacity, and birth to a fragment shader; use `uTime` for fade-in and `lineOpacity` for true hiding.
- [ ] **Step 5: Replace focus logic.** Compute desired edge visibility only when focus or hidden types change; update the opacity attribute and `needsUpdate` once per change.
- [ ] **Step 6: Run tests and commit.** Run `go test ./internal/viewer -count=1`, syntax checks, then commit `perf(viewer): batch relationship rendering`.

### Task 5: Make node updates dirty-driven and add amount styling to instance buffers

**Files:**
- Modify: `internal/viewer/index.html`
- Modify: `internal/viewer/viewer_test.go`

**Interfaces:**
- `syncNodeInstance(n, scale, visible, focused) -> boolean`: writes matrix/color/hitbox only if the desired state changed.
- `nodeVisualsDirty` and `linkVisualsDirty`: explicit upload flags.
- `applyRecipeAmountVisuals(productId)`: calls `syncNodeInstance` for affected ingredient nodes.

- [ ] **Step 1: Add failing assertions for dirty updates.** Require `nodeVisualsDirty`, guarded `instanceMatrix.needsUpdate`, guarded `instanceColor.needsUpdate`, `hitboxInstances.instanceMatrix.needsUpdate`, and a comparison against cached scale/color/visibility.
- [ ] **Step 2: Run the targeted test and verify RED.** Run the targeted viewer test.
- [ ] **Step 3: Implement cached state.** Store last scale/visible/focus/color on each node and only call `setMatrixAt`/`setColorAt` on differences; set `needsUpdate` once after the node loop.
- [ ] **Step 4: Separate animation from steady-state.** Continue syncing nodes whose birth animation is incomplete; skip settled nodes and avoid recreating arrays or colors.
- [ ] **Step 5: Wire product selection and reset.** Call amount styling on Product selection, clear it on non-Product/escape/load, and force one visual sync after state transitions.
- [ ] **Step 6: Run tests and commit.** Run viewer tests, full Go tests, syntax checks, and commit `perf(viewer): make instance updates dirty-driven`.

### Task 6: Add label LOD, raycast throttling, geometry/DPR tuning, and diagnostics

**Files:**
- Modify: `internal/viewer/index.html`
- Modify: `internal/viewer/viewer_test.go`

**Interfaces:**
- `MAX_VISIBLE_LABELS = 96`: high-scale label budget.
- `syncLabelLOD(force)`: selects labels using focus/hover/search priority and camera distance.
- `maybeUpdateHover(now)`: runs raycast only when dirty and no more than 30Hz.
- `configureRenderQuality(nodeCount)`: selects pixel ratio cap.

- [ ] **Step 1: Add failing assertions for label budget and diagnostics.** Require the constants/functions and `renderer.info.render.calls`; reject unconditional label scale updates in the main loop.
- [ ] **Step 2: Run the targeted test and verify RED.** Run the targeted viewer test.
- [ ] **Step 3: Implement lazy labels.** Keep text data on node records, create/dispose Sprite labels only for selected candidates, and preserve full labels for small graphs.
- [ ] **Step 4: Implement hover throttling.** Mark picking dirty on pointer/camera changes, keep click immediate, and execute hover raycast at most every 33ms.
- [ ] **Step 5: Tune geometry/DPR and expose diagnostics.** Use 16×12 node spheres, cap DPR to 1.5 for large graphs, and update stats with draw calls/triangles/lines without changing the main user-facing count text semantics.
- [ ] **Step 6: Run full static tests and commit.** Run `go test -count=1 -race ./...`, `make build`, syntax checks, and commit `perf(viewer): add scale-aware label and render quality controls`.

### Task 7: Run browser/live validation and final confirmation

**Files:**
- Modify: `internal/viewer/viewer_test.go` only if a discovered regression needs a permanent assertion.

**Interfaces:**
- Validation evidence: API response source/code, page load, console errors, node/relationship counts, amount visual state, reset behavior, and renderer diagnostics.

- [ ] **Step 1: Start the isolated service on an unused port.** Provision ignored runtime configuration only if needed, use `KG_VIEWER_PORT=8092` or another confirmed-free port, and do not print secrets.
- [ ] **Step 2: Verify API/page source.** Check `/healthz`, `/v1/graph/kg`, and `/kg-viewer`; confirm the served page contains the optimization markers and API returns `code=0`.
- [ ] **Step 3: Run browser checks.** Load the offline snapshot and live graph, inspect console errors, click a Product, confirm connected Ingredient sizes/colors change, click a non-Product/Esc, confirm defaults restore, and verify search/hover still work.
- [ ] **Step 4: Capture performance evidence.** Record `renderer.info` calls/triangles/lines and load/render timing for the available graph; if the live graph is not the stated 2871-node dataset, label the result as a structural/functional validation and do not claim an unmeasured FPS gain.
- [ ] **Step 5: Run final checks.** Run `git diff --check`, `go test -count=1 -race ./...`, `make build`, and inspect `git status`/`git log`.
- [ ] **Step 6: Report the worktree path, branch, commits, test results, browser evidence, and any remaining limitation.**
