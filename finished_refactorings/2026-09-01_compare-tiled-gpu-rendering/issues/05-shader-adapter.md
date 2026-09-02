# 05 - Implement the tiled shader adapter

Status: resolved

## Contract

Render one overview and up to seven detail tiles with stable pane-specific
shader names. Update scalar uniforms for movement; update texture identities only
when source/tile bindings change. Supply equivalent desktop GLSL 110 and GLES
GLSL 100 programs, bilinear sampling, transparent bounds, and RGB unpremultiply.

## Acceptance

Structural tests lock declarations, sampler count, lookup/body equivalence,
stable identity, slot clearing, finest-match selection, and scene-to-uniform
mapping without requiring a GPU.

## Comments

The shader tests failed first on absent GLSL and adapter APIs. Both GLSL
variants now share one byte-identical body after their preambles, declare eight
samplers, select the finest matching detail over the overview, and unpremultiply
alpha. Adapter tests pin stable names/objects, fixed slots, geometry/uniform
mapping, texture reuse, and bounded clear behavior. Independent audit tests
also caught and fixed vertical texture inversion, sampler-slot shuffling,
excess GLES uniforms, raw-coordinate mediump hazards, and extreme-aspect
selection/division guards.
