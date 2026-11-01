# Checkbox Detection

**[Try the live demo](https://checkbox-detection.jpmiceli.com.ar)** ·
[Watch the 1-minute walkthrough](#walkthrough) ·
API: `POST https://api.jpmiceli.com.ar/detect`

Try the deployed API from the repository root:

```sh
curl --fail-with-body -F 'image=@backend/testdata/im1.jpg' https://api.jpmiceli.com.ar/detect
```

A Go HTTP API that detects checkboxes in document images and classifies each as
checked or unchecked. It uses GoCV/OpenCV with contour and line-based detection;
no trained model or external inference service is required.

The backend is the main deliverable. The Angular frontend is an optional visual
inspector for exploring the API's results and detection settings.

## Build, run, and test the API

Requirements: Docker Engine and Docker Compose. No local Go, OpenCV, or frontend
installation is needed for this path.

From the repository root:

```sh
cd backend
docker compose up -d --build --wait
curl --fail-with-body -F 'image=@testdata/im3.png' http://localhost:8080/detect
```

The first build compiles OpenCV and can take several minutes. The API listens on
`http://localhost:8080`. If that port is occupied, start Compose with
`API_PORT=8081 docker compose up -d --build --wait` and use port 8081 in requests.

`POST /detect` takes an image file and returns:

```json
{
  "boxes": [
    { "bbox": [10, 20, 40, 60], "is_checked": true },
    { "bbox": [80, 20, 110, 60], "is_checked": false }
  ]
}
```

The coordinates above illustrate the response format. `bbox` contains the
original image's `[x1, y1, x2, y2]` pixel coordinates, with an exclusive right and
bottom boundary. An image with no detected checkboxes returns `{"boxes":[]}`.
The response is computed from the uploaded image, not a hardcoded sample.

Try the four assignment examples, then run the backend tests in the same image:

```sh
for image in testdata/im0.png testdata/im1.jpg testdata/im2.png testdata/im3.png; do
  echo "$image"
  curl --fail-with-body -F "image=@$image" http://localhost:8080/detect
  echo
done

docker compose run --rm --no-deps --entrypoint go api \
  test -mod=readonly -tags matprofile ./...
```

Tests cover detection behavior, classification, invalid requests, concurrency,
cancellation, and native memory cleanup. See [verification and known limitations](backend/README.md#verification-and-known-limitations).

To inspect logs or stop the service:

```sh
docker compose logs --tail=100 api
docker compose down
```

For native Go development, configuration, and the full API contract, see the
[backend README](backend/README.md).

## Approach and tradeoffs

1. Convert the image to grayscale, blur lightly, and apply adaptive thresholding.
2. Generate candidates with two complementary detectors: rectangular contours
   and horizontal/vertical border lines.
3. Learn repeated checkbox dimensions from strong contour references and filter
   candidates against the document's geometry.
4. Inspect the original image's background and border contrast to reject shapes
   produced by text in dark bars.
5. Remove overlapping duplicates and classify each remaining box by interior ink
   density, excluding its border.

This approach makes the detection rules inspectable and avoids collecting a
training dataset or deploying a model. The tradeoff is sensitivity to scan scale,
skew, damaged borders, nearby text, and faint marks. It can produce false
positives and miss checkboxes; perfect accuracy is not assumed. The
[backend writeup](backend/README.md#approach-and-design-decisions) explains the
algorithm, design choices, and remaining limitations.

## Code map

| Location                     | Responsibility                                                    |
| ---------------------------- | ----------------------------------------------------------------- |
| `backend/main.go`            | Server startup, flags, and graceful shutdown                      |
| `backend/internal/httpapi/`  | Upload parsing, API responses, and bounded admission              |
| `backend/internal/checkbox/` | Candidate detection, filtering, deduplication, and classification |
| `backend/internal/imaging/`  | Image decoding and preprocessing                                  |
| `backend/internal/config/`   | Server defaults and per-request detection settings                |
| `backend/testdata/`          | Four assignment sample images                                     |
| `frontend/`                  | Optional Angular inspector                                        |

## Deployment

The Go API runs in Docker on AWS EC2 behind Caddy for HTTPS. The optional frontend
is served from S3 through CloudFront. GitHub Actions deploys each on relevant
pushes to `main`; backend images are published to ECR and deployed through SSM.

## Optional frontend

### Walkthrough

https://github.com/user-attachments/assets/b28e0484-b5e3-4d9f-a039-5f228dd8a16b

### Run locally

The API can be reviewed entirely with curl and backend tests. For visual
inspection, keep the backend running and open a second terminal at the repository
root. With Node.js 24.15+ in the 24.x line and Bun 1.4.4 installed:

```sh
cd frontend
bun install --frozen-lockfile
bun run start
```

Open `http://localhost:4200`, upload an image from `backend/testdata/`, and click
**Process**. The interface supports image zoom, selectable checkbox annotations,
a JSON response view, and editable detection settings. The settings panel keeps
annotations visible for comparing reprocessing results. Documents and results
live in browser memory.

**The frontend is designed for a full-screen desktop browser window. It is not
responsive:** smaller windows, tablets, and phones may have clipped or difficult-to-use
layouts.

The frontend sends same-origin `/detect` requests. `ng serve` proxies `/detect`
and nested paths to `http://localhost:8080` using `frontend/proxy.conf.json`.
Update that target if you run the backend on another port. The frontend is not
required to build, run, or test the backend.

Frontend checks, from `frontend/`:

```sh
bun run lint
bun run build
bun run test --watch=false
```
