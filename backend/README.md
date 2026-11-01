# Checkbox detection backend

Go + GoCV/OpenCV implementation of `POST /detect`: upload a document image and
receive checkbox locations and checked/unchecked classifications. Detection runs
locally without a trained model, external API, or database.

All commands below run from `backend/` unless stated otherwise.

## Run with Docker

Requirements: Docker Engine and Docker Compose. The image installs Go 1.27.3,
GoCV 0.43.0, and OpenCV 4.13.0, so native dependencies are not needed on the host.

```sh
docker compose up -d --build --wait
curl --fail-with-body -F 'image=@testdata/im3.png' http://localhost:8080/detect
```

The first build compiles OpenCV and takes several minutes. Compose mounts
`config.example.json` read-only, runs the API as a non-root user, and publishes
port 8080 on localhost. Use `API_PORT=8081 docker compose up -d --build --wait`
if port 8080 is occupied.

```sh
docker compose logs --tail=100 api
docker compose down
```

The Docker image retains source and build tools so tests can run in the same
OpenCV environment as the server.

## Test the required endpoint

`POST /detect` accepts `multipart/form-data` with an `image` file field:

```sh
curl --fail-with-body \
  -F 'image=@testdata/im0.png' \
  http://localhost:8080/detect
```

Illustrative response:

```json
{
  "boxes": [
    { "bbox": [10, 20, 40, 60], "is_checked": true },
    { "bbox": [80, 20, 110, 60], "is_checked": false }
  ]
}
```

- `bbox` is `[left, top, right, bottom]` in original-image pixels. Right and
  bottom are exclusive; the first illustrative box is 30 pixels wide and 40 high.
- `is_checked` indicates whether the detected box's interior contains enough ink
  to classify it as filled.
- Results are ordered by position. No detections returns `{"boxes":[]}`.
- The server computes results per image. It does not return the sample JSON or
  save result files. Multipart parsing may use temporary files, which are removed
  before image processing.

The compatibility multipart field `file` and raw `image/png` or `image/jpeg`
bodies are also accepted:

```sh
curl --fail-with-body -H 'Content-Type: image/png' \
  --data-binary @testdata/im3.png http://localhost:8080/detect
```

For browser uploads, let FormData set the multipart Content-Type and boundary.
The endpoint is `/detect`; `/api/detect` is not registered.

### Input limits and errors

Uploads are limited to 20 MiB for the entire request, including multipart
metadata. Decoded images are limited to 40 megapixels by default. Only valid PNG
and JPEG images are processed.

Errors have the shape `{"error":"message"}`:

| Status | Meaning                                                                                                     |
| ------ | ----------------------------------------------------------------------------------------------------------- |
| `400`  | Invalid image, malformed upload, or invalid detection settings                                              |
| `405`  | Method not supported by the endpoint                                                                        |
| `408`  | Processing canceled or its deadline exceeded                                                                |
| `413`  | Request byte limit or decoded image pixel limit exceeded                                                    |
| `415`  | Unsupported request content type                                                                            |
| `500`  | Unexpected processing failure; details are logged server-side                                               |
| `503`  | Capacity exhausted, queue wait expired, or server draining; includes `Retry-After: 1` on detection requests |

An invalid-image example:

```sh
curl -i -H 'Content-Type: image/png' --data-binary 'not an image' \
  http://localhost:8080/detect
```

## Verification and known limitations

Run the tests using the already-built Compose image:

```sh
docker compose run --rm --no-deps --entrypoint go api \
  test -mod=readonly -tags matprofile ./...
```

Alternatively, `make docker-test` builds a test image and runs the same Go tests.
With the native toolchain installed, `make check` checks formatting, verifies
modules, runs vet and tests, and builds the server.

The four assignment images are `testdata/im0.png`, `im1.jpg`, `im2.png`, and
`im3.png`. With the default settings, observed detection counts are:

| Image     | Detected boxes |
| --------- | -------------: |
| `im0.png` |            118 |
| `im1.jpg` |             42 |
| `im2.png` |             48 |
| `im3.png` |             79 |

These counts are observations, not accuracy scores or assertions in the test suite.
There are no stored expected-output files or complete ground-truth annotations.

Tests cover detector switches, diagnostic consistency, marked-box recovery,
selected false-positive cases, classification, upload validation and cleanup,
settings isolation, admission limits, cancellation, shutdown, and native matrix
ownership using GoCV's `matprofile` tag.

Known weaknesses include a text false positive in `im1`, a missed damaged
No Zoning checkbox, and a miss Neighborhood Boundaries checkbox.
The heuristics are sensitive to image scale, skew, broken borders, adjacent text,
and documents with few reliable reference boxes. There is no OCR, deskew,
perspective correction, or missing-border reconstruction.

The next accuracy step would be to label the sample boxes and marks, measure
localization precision/recall separately from checked-state classification, and
use those measurements to evaluate threshold changes. Larger or more varied
inputs would also need latency and memory measurements; no throughput benchmark
is claimed here.

## Approach and design decisions

### Candidate generation: two complementary detectors

Preprocessing converts to grayscale, applies a small Gaussian blur, and uses
adaptive inverted thresholding. Foreground ink becomes white in the binary
matrix. Local thresholding accommodates uneven illumination better than one
page-wide cutoff, but can lose evidence inside uniformly dark regions.

The **outline detector** finds contours, approximates them as convex
quadrilaterals, and checks dimensions, rectangularity, and visible border ink.
Its minimum/maximum size applies to both width and height. By default it requires
rectangularity of 0.85 and enough ink on at least three sides.

The **border-line detector** extracts horizontal and vertical structures with
morphological opening, pairs possible top/bottom lines, and verifies all four
sides and an open interior. It can recover boxes that the outline path misses,
but also generates proposals from nearby text and form rules. Its width and
height ranges remain separate in the API; the optional frontend applies one
minimum and one maximum to both dimensions.

Neither detector assumes checkboxes must be square. Both propose geometry;
checked-state classification happens later.

### Document context and false-positive rejection

Strong contour references with clear borders and open interiors teach repeated
width/height clusters. References are deduplicated before learning so inner and
outer contours cannot count as independent boxes. Line candidates do not teach
the model because their numbers could overwhelm stronger evidence.

References require rectangularity of 0.90. Weaker contours need a close 12% size
match to the learned model; established contours and lines allow 20%, with a
two-pixel tolerance floor. Without enough references, established proposals pass
through while weak contours that require model support are withheld.

Background filtering uses the original image to distinguish real outlines from
rectangular shapes formed by lettering inside dark bars. It examines exterior
brightness and border contrast without using the checkbox's interior mark. This
keeps localization and checked-state classification separate.

Geometry and background filtering precede final overlap deduplication: otherwise
a large false proposal could suppress a smaller real checkbox before being
rejected. Larger overlapping boxes are preferred, with deterministic tie-breaking.

### Classification and algorithm tradeoffs

After detection, the classifier excludes the border and measures foreground
pixels in the center. Interior ink density strictly above 12% means checked with
the default settings. This is simple to inspect and tune, but faint ticks can be
missed and stray ink can be mistaken for a mark.

Classical image processing was chosen to make the rules explicit and avoid a
training-data requirement or model-serving dependency. The cost is hand-tuned
heuristics with limited tolerance for unfamiliar document styles. A learned
approach could be evaluated once labeled, representative data is available.

### Request lifecycle and resource ownership

The HTTP layer handles methods, uploads, errors, and capacity independently of
the image pipeline. A request must acquire a processing slot before its body is
read. A bounded waiting queue avoids starting native work for every incoming
request at once; exhausted capacity returns 503 so callers can retry later.

GoCV matrices and contour allocations are explicitly closed. The detector keeps
an immutable configuration value; each request owns its matrices. Cancellation
is checked between stages and inside Go loops, but cannot interrupt an OpenCV
call already running. Its active slot stays held until processing returns.

`GET /healthz` reports liveness. `GET /readyz` returns 503 during shutdown.
SIGINT/SIGTERM stops admission and drains active requests for up to 120 seconds;
Compose allows 130 seconds before forced termination.

## Configuration and optional API extensions

The required upload endpoint works without any settings field. Defaults live in
[internal/config/config.go](internal/config/config.go); `-config path/to/config.json`
applies partial server overrides. Compose mounts [config.example.json](config.example.json).
Each detector holds a value copy of its configuration. Public detection settings
are validated; preprocessing and internal heuristic values are trusted server
configuration. The configuration loader rejects unknown fields, nulls, and
malformed JSON.

Operational overrides are listed in [.env.example](.env.example). Compose defaults
to two active requests, four waiting requests, a 15-second queue wait, and a
60-second processing deadline. The native server defaults to one active request
and a two-second queue wait; startup flags can override these. A capacity slot
covers reading the upload through writing its response.

After editing the server config or operational environment:

```sh
docker compose up -d --no-build --force-recreate --wait api
```

### Per-request settings

As an optional extension, `GET /detect/settings` returns current defaults and
numeric bounds as `{ "defaults": { ... }, "limits": { ... } }`, with caching
disabled. These are the server's loaded values, not frontend constants.

Supply a partial JSON object in the multipart `settings` field to override one
request:

```sh
curl --fail-with-body -F 'image=@testdata/im3.png' \
  -F 'settings={"checkedInkRatio":0.2,"matchToleranceRatio":0.15}' \
  http://localhost:8080/detect
```

Omitted values retain server defaults. Ratios are fractions, not percentages.
Overrides never change another request or the defaults endpoint. Unknown fields,
nulls, invalid ranges, non-integer pixel/side counts, and malformed settings JSON
return 400. Settings JSON is limited to 8192 bytes; preprocessing and operational
settings cannot be changed through this API.

| Controls                          | JSON keys                                            |
| --------------------------------- | ---------------------------------------------------- |
| Detector switches                 | `outlineEnabled`, `lineEnabled`                      |
| Checked-mark threshold            | `checkedInkRatio`                                    |
| Outline dimensions                | `minSidePx`, `maxSidePx`                             |
| Outline shape and border evidence | `minRectangularity`, `minStrongSides`, `minCoverage` |
| Line width                        | `minWidthPx`, `maxWidthPx`                           |
| Line height                       | `minEdgeSeparationPx`, `maxEdgeSeparationPx`         |
| Document-size tolerance           | `matchToleranceRatio`                                |
| Duplicate overlap                 | `finalIoUThreshold`                                  |

At least one detector must remain enabled, and minimum sizes cannot exceed their
maximums. In line-only mode there are no outline references to learn document
sizes; background filtering, deduplication, and classification still run.

## Native development and formatting

Requires Go 1.27.3+, Make, a C++ compiler, pkg-config, and OpenCV 4.13.0. Docker is
the primary review path; the repository does not install native host dependencies.

```sh
OPENCV_PREFIX=/path/to/opencv make serve
OPENCV_PREFIX=/path/to/opencv make check
make fmt
```

The build wrapper also accepts pkg-config's default search path or
`/opt/opencv-4.13`. It embeds the native library path in executables and tests.
`make` starts the server at `127.0.0.1:8080`; use
`make serve CONFIG=config.example.json` to load server overrides.

For the optional visual inspector, see the [root README](../README.md#optional-frontend).
