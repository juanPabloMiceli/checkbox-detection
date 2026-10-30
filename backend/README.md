# Checkbox detection backend

Go + GoCV service that finds checkbox bounds and checked/unchecked state in
PNG/JPEG document images, without a trained model. Run commands from `backend/`.
The server starts in [main.go](main.go); there are no other executables.

## Run with Docker

Requirements: Docker Engine and Docker Compose. Docker installs Go 1.27.3,
GoCV 0.43.0, and OpenCV 4.13.0; no native setup is needed on your machine.

```sh
docker compose up -d --build --wait
curl --fail -F 'image=@testdata/im3.png' http://localhost:8080/detect
docker compose logs --tail=100 api
docker compose down
```

Use `API_PORT=8081 docker compose up -d --build --wait` if port 8080 is occupied.
The first build compiles OpenCV and takes several minutes. The Dockerfile uses
one image: install OpenCV, copy the source, build the server, then run as a
non-root user. Source and build tools remain available for running tests.
Docker checks `/healthz` using curl.

## Native development

Requires Go 1.27.3+, Make, a C++ compiler, pkg-config, and OpenCV 4.13.0 already
installed. Docker is the primary setup path; the project does not install native
dependencies on the host.

```sh
OPENCV_PREFIX=/path/to/opencv make serve
OPENCV_PREFIX=/path/to/opencv make check
```

The small build wrapper also accepts pkg-config's default search path or
`/opt/opencv-4.13`. It embeds the library path in executables and tests.
`make` defaults to starting the server at `127.0.0.1:8080`.

## API

`POST /detect` accepts multipart uploads with an `image` file field. The `file`
field and raw bodies with `Content-Type: image/png` or `image/jpeg` also work.
Let the browser set the multipart boundary when using FormData.

```json
{ "boxes": [{ "bbox": [10, 20, 40, 60], "is_checked": true }] }
```

Bounds are `[left, top, right, bottom]` in original-image pixels; right and bottom
are exclusive. Boxes are sorted by position. Empty results are `{"boxes":[]}`.
Images are processed in memory without saving output files.

Requests are limited to 20 MiB, including multipart overhead, and decoded images
to 40 megapixels by default. Errors use `{"error":"message"}` with status `400`
for invalid images, `413` for size limits, `415` for unsupported content types,
`405` for unsupported methods, `408` for canceled processing, or `503` with
`Retry-After: 1` when capacity is full. Browser CORS allows origins without
credentials. `/api/detect` is not registered.

`GET /healthz` reports liveness; `GET /readyz` returns `503` during shutdown.
SIGINT/SIGTERM rejects queued/new requests and drains active work for up to
120 seconds. Compose allows 130 seconds before forced termination.

## Configuration

`-config path/to/config.json` applies partial overrides to the defaults in
[internal/config/config.go](internal/config/config.go).
[config.example.json](config.example.json) is mounted read-only by Compose.
Unknown fields, nulls, invalid values, and inconsistent ranges fail at startup.
Each detector holds an immutable copy of its configuration.

For native development: `make serve CONFIG=config.example.json`.
For operational settings, copy `.env.example` to `.env`. Compose defaults to two
active requests, four queued requests, a 15-second queue wait, and a 60-second
processing deadline. After changing settings, recreate the container:

```sh
docker compose up -d --no-build --force-recreate --wait api
```

The native server defaults to one active request and a two-second queue wait;
startup flags override these. A capacity slot covers upload reading through
response writing. Cancellation is checked between stages and inside processing
loops; native OpenCV calls finish before releasing their slot.

## Per-request detection settings

`GET /detect/settings` returns the active detector's twelve numeric defaults plus
`outlineEnabled` and `lineEnabled` switches
and their numeric limits as `{ "defaults": { ... }, "limits": { ... } }`. These
reflect the loaded server configuration, not a second set of frontend constants.
The response is not cached. Preprocessing and operational settings are not exposed.

For a multipart detection request, optionally include a `settings` JSON field:

```sh
curl -F 'image=@testdata/im3.png' \
  -F 'settings={"checkedInkRatio":0.2,"matchToleranceRatio":0.15}' \
  http://localhost:8080/detect
```

Omitted fields retain server defaults. Ratios are fractions from 0 to 1, not UI
percentages. Requests without `settings` keep the previous behavior. Overrides are
validated and applied to an isolated configuration copy; they never change other
requests or the defaults endpoint. Invalid ranges, nulls, unknown fields (including
preprocessing fields), non-integer pixel/side counts, and malformed JSON return
400 before image processing. Settings JSON is limited to 8192 bytes.

Both detectors default to enabled. Set `outlineEnabled` or `lineEnabled` to false
to skip that algorithm; at least one must remain enabled. Numeric controls for
disabled detectors are inactive in the frontend. In line-only mode, document-size
matching is inactive because no outline references are generated; background
filtering, duplicate removal and checked-mark classification still apply.

The supported numeric keys are `checkedInkRatio`, `minSidePx`, `maxSidePx`,
`matchToleranceRatio`, `minRectangularity`, `finalIoUThreshold`, `minWidthPx`,
`maxWidthPx`, `minEdgeSeparationPx`, `maxEdgeSeparationPx`, `minStrongSides`, and
`minCoverage`. Minimum sizes cannot exceed their maximums. Shape strictness cannot
exceed the server's reference-shape threshold; the defaults endpoint supplies that
upper bound. Positive-only ratio limits include `exclusiveMin: true`.

## Detection approach

1. Convert to grayscale, blur lightly, and apply adaptive inverted thresholding.
2. Generate proposals from convex quadrilateral contours and morphology-extracted
   horizontal/vertical lines.
3. Learn width/height clusters from strong contour references with open interiors,
   then filter proposals against those dimensions.
4. Inspect original-image exterior brightness and outline contrast to reject
   rectangular text counters in dark bars, without sampling the interior mark.
5. Deduplicate proposals and classify central foreground density.

Contours need rectangularity of at least 0.85; learning references need 0.90.
Weaker contours need a close 12% match to learned dimensions. Established contours
and lines allow 20%, with a two-pixel tolerance floor. Without enough references,
established proposals pass and recovery-only contours are withheld.
Geometry and background filtering precede final deduplication. Equal-area ties
have a deterministic order. Central ink density above 12% means checked.

The pipeline can optionally return stage diagnostics for inspection and tests.
These reuse stage results; normal HTTP processing returns only detections.

## Verification and limitations

```sh
make check        # Formatting, module verification, vet, tests, native cleanup, build.
make docker-test  # Build the image and run tests, including native cleanup checks.
```

Tests freeze every box, order, and checked flag for five supplied images:
118 / 42 / 48 / 79 / 26 boxes. The [fixtures](testdata/expected/README.md) are
behavior snapshots rather than complete accuracy labels. Tests also cover input
validation, temporary upload cleanup, CORS, configuration, concurrent admission,
cancellation, shutdown, and native matrix ownership via GoCV's `matprofile` tag.

The detector is sensitive to scale, skew, broken borders, adjacent text, sparse
references, and faint marks. There is no OCR, deskew, perspective correction, or
missing-border reconstruction. Known sample issues include a text false positive
in `im1`, a missed damaged No Zoning box, and an Electricity/Public faint mark
classified unchecked. Further work should measure precision/recall against
labeled boxes and classification separately before adjusting the heuristics.

## Deployment

[Deploy backend](../.github/workflows/deploy-backend.yml) runs on pushes to `main`
that change any file under `backend/`, or can be started manually. It builds an
AMD64 image, pushes a unique release tag to ECR, and sends
[scripts/deploy.sh](scripts/deploy.sh) to the EC2 server using SSM Run Command.
The workflow waits for that command to finish and reports its output. It uses
no SSH keys and has no test or post-deployment verification steps.

The workflow uses the `checkbox-backend-github-deploy` AWS role, which trusts this
repository's `main` branch through GitHub OIDC. Its trust uses the immutable
owner/repository IDs in the GitHub subject, rather than the older name-only
format. Its permissions cover ECR pushes
to `checkbox-detection/backend`, `ecr:GetAuthorizationToken`, `ssm:SendCommand`
for `AWS-RunShellScript` on instance `i-054db4493fc491324`, and
`ssm:GetCommandInvocation` to report completion. No GitHub AWS secrets or role
variables are needed.

The server needs SSM Agent running, AWS CLI, and an EC2 instance role with
`AmazonSSMManagedInstanceCore` and ECR pull permissions for this repository.
Docker, `.env`, `compose.ec2.yaml`, and Caddy are already configured.
Deployments pull the image and extract its Compose file and detector config,
preserving the server's operational settings and Caddy. Cleanup retains the
current and previous backend images locally; ECR retains its three newest images.
