# Checkbox Detection

An Angular workspace for uploading document images and inspecting checkbox
locations. Includes multiple uploads, drag and drop, concurrent processing,
retry/removal, image zoom, selectable annotations, and a highlighted JSON view.

## Run locally

Prerequisites: Node.js 22.22.3+ within the 22.x release line, 24.15.0+ within
24.x, or 26+, and Bun 1.4.4.

From the repository root:

```bash
cd frontend
bun install --frozen-lockfile
bun run start
```

Open http://localhost:4200.

## Browser support

The interface is intended for use in a computer (desktop or laptop) browser.
Responsive layouts for phones and tablets are out of scope; on smaller screens,
parts of the interface may be difficult to use or clipped.

**Detection requires a separate API running at http://localhost:8080.** The backend
currently lives outside this repository; `backend/` is reserved for its source. The frontend sends `POST /api/detect`; configure
its URL in `frontend/src/app/services/detection.service.ts` if needed. The API must allow
requests from `http://localhost:4200` through CORS. Upload and preview work without
it; processing reports a connection error until it is available.

## Demo walkthrough

1. Choose or drag in `frontend/public/samples/document.png`.
2. Click **Process**, or upload multiple images and use **Process all**.
3. Select a checkbox in the image or results list; use Previous/Next to navigate.
4. Use the zoom controls, Fit, or Ctrl+mouse wheel to inspect the image.
5. Switch results between Checkboxes and JSON. Hide annotations hides the results
   panel too, giving the preview the full available width.
6. Reprocess or remove documents as needed. Failed requests show a Retry action.

## API contract and limitations

`POST /api/detect` accepts `multipart/form-data` with an `image` file field:

```bash
curl -F 'image=@frontend/public/samples/document.png' http://localhost:8080/api/detect
```

Expected response shape (coordinates are `[x1, y1, x2, y2]` in original image pixels):

```json
{
  "boxes": [
    { "bbox": [330, 508, 386, 552], "is_checked": true }
  ]
}
```

`frontend/public/samples/detections.json` contains the complete fixture for the sample image.
The original development mock returned that fixed response for every uploaded image;
**its coordinates match the sample document, not arbitrary images**. The frontend
renders whatever detections the connected API returns.

The frontend accepts image files and shows a preview error if decoding fails.
The development mock imposed a 10 MiB request limit; limits depend on the connected
API. Requests that fail can be retried. Documents and their results live in memory
and are lost on refresh; object URLs and requests are cleaned up on removal.

## Structure

```text
checkbox-detection/
├── frontend/       # Angular app, dependencies, assets and build configuration
├── backend/        # Reserved for the API source; currently contains setup notes
└── README.md
```

Frontend responsibilities:

- `DocumentWorkspace`: document state, selection, requests and resource cleanup.
- `DetectionService`: multipart HTTP adapter.
- `UploadArea` and `UploadedDocumentList`: file input and document actions.
- `DocumentViewer`: selection and annotation visibility coordination.
- `ImageViewport`: image geometry, zoom and overlays.
- `DetectionResults` and `JsonResponse`: list navigation and formatted response.
- `frontend/src/styles`: shared color tokens and button styles.

## Build and checks

From `frontend/`:

```bash
bun run build
bun run test --watch=false
```

The production output is in `frontend/dist/checkbox-detection`. To run its SSR server:

```bash
bun run serve:ssr:checkbox-detection
```

Existing frontend tests are basic smoke tests; full browser interaction coverage
is not included.

Once the backend source has been copied into `backend/`, start it in a separate
terminal from that directory (for the Go mock: `go run .`). See
[backend setup notes](backend/README.md) for the expected API contract.
