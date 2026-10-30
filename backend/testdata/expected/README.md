# Reviewed characterization fixtures

These files freeze the detector's default behavior before structural cleanup.
They are independent of `results/`, which the CLI is allowed to overwrite.

The original assignment examples produce 118, 42, 48, and 79 boxes for im0–im3.
The additional user-supplied `im5.png` produces 26. All coordinates, output
ordering, and checked flags are preserved, not just the counts. `manifest.json`
records source-image hashes and the native versions used to create the snapshot.
Tests fail if an input image changes without a corresponding review.

These are characterization outputs, not complete ground-truth labels. Known
limitations such as the remaining im1 text false positive and faint-mark
misclassification are intentionally recorded. The focused recall tests retain
the six positive marked-box annotations and the negative Att. text-cell case.

There is deliberately no automatic "update goldens" command. For an intentional
algorithm change, inspect every changed box, document its effect, then update
the selected expected file and image hash if the source changed. A structural
refactor must not update these files to hide a regression.
