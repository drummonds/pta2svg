# Roadmap

## Done

- [x] Core parser (PTA, Goluca, Beancount via tree-sitter)
- [x] LR column layout engine
- [x] SVG renderer with text/template
- [x] CSS flow animations (optional)
- [x] CLI with -o, -animate, -title, -layout, -format flags
- [x] Multi-commodity support
- [x] Balance display on account nodes
- [x] Syntax-highlighted PTA source in docs site

## Phase 1 — Better automatic layout

- [ ] Integrate `nulab/autog` as Sugiyama/layered layout (`-layout sugiyama`)
- [ ] Proper crossing minimisation and edge routing
- [ ] Support autog edge points (polyline/ortho/spline) in renderer
- [ ] Keep LR as default, Sugiyama as alternative

## Phase 2 — Clustering / grouping

- [ ] Group nodes by account hierarchy prefix (e.g. all `Asset:Bank:*`)
- [ ] Post-layout bounding-box computation with padding
- [ ] Render cluster backgrounds as SVG `<g>` + `<rect>` (rounded, dashed)
- [ ] Irregular shapes via SVG `<path>` for non-rectangular groups

## Phase 3 — Animation

- [ ] SMIL `<animateMotion>` for money flowing along edge paths
- [ ] Sequenced animation via existing delay indices → SMIL `begin` attributes
- [ ] Docs samples: static vs animated diagrams side-by-side

## Phase 4 — Visual styles & docs samples

- [ ] Varied node shapes (ellipse, rounded-rect, diamond)
- [ ] Sankey-style edge width proportional to amount
- [ ] Gradients and shadows via SVG `<defs>`
- [ ] Docs site sections: basic, animated, clustered, styled
- [ ] Account declarations with explicit types/colors
