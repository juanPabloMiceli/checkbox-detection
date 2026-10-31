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
currently lives outside this repository; `backend/` is reserved for its source. The frontend sends same-origin `POST /detect` requests. `ng serve` forwards
them to `http://localhost:8080` using `frontend/proxy.conf.json`; update that target
if your local API uses a different address. Browser CORS configuration is not
required for this proxied setup. Upload and preview work without
it; processing reports a connection error until it is available.

## Demo walkthrough

1. Choose or drag in `frontend/public/samples/document.png`.
2. Click **Process**, or upload multiple images and use **Process all**.
3. Select a checkbox in the image or results list; use Previous/Next to navigate.
4. Use the zoom controls, Fit, or Ctrl+mouse wheel to inspect the image.
5. Switch results between Checkboxes and JSON. Hide annotations hides the results
   panel too, giving the preview the full available width.
6. Reprocess or remove documents as needed. Failed requests show a Retry action.

## Detection settings

The viewer's **Settings** button opens a collapsible right-hand panel, sharing its
space with the annotation results list. Image annotations remain visible while
settings are open, so reprocessing updates the boxes alongside the controls. Hover over or focus the question marks for
help. Ten numeric controls and individual detector switches cover checked-mark classification, outline and line
sizes, shape/border requirements, size tolerance and duplicate removal. There are
no preprocessing controls. Both detectors start enabled; each can be switched off
independently, but at least one must stay enabled. Controls for disabled detectors
are inactive, and Reset defaults restores the server’s detector choices.

Defaults and bounds load from `GET /detect/settings`. Percentages are displayed as
percentages and sent as fractions in the multipart `settings` field on `POST /detect`.
Settings apply to all documents on their next processing run; existing results stay
unchanged until reprocessed. **Reset defaults** restores the server values.
Invalid or missing values and inconsistent size ranges are highlighted and block
processing. If defaults cannot load, open Settings and retry. The backend validates
these values independently and isolates overrides to their request.

## API contract and limitations

`POST /detect` accepts `multipart/form-data` with an `image` file field:

```bash
curl -F 'image=@frontend/public/samples/document.png' http://localhost:8080/detect
```

Expected response shape (coordinates are `[x1, y1, x2, y2]` in original image pixels):

```json
{
  "boxes": [{ "bbox": [330, 508, 386, 552], "is_checked": true }]
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

## Formatting and linting

From `frontend/`:

```bash
bun run format        # Format supported files across the repository
bun run format:check  # Check formatting without changing files
bun run lint          # Check TypeScript and Angular templates; warnings fail
bun run lint:fix      # Apply automatic lint fixes
```

Prettier uses the shared root `.prettierrc` and `.prettierignore`. Generated
output, caches, dependencies, the lockfile and sample fixtures are excluded.
ESLint uses `frontend/eslint.config.mjs` with the recommended TypeScript and Angular
rules, template accessibility checks, and Angular selector conventions. Prettier
handles formatting; ESLint checks code quality. Editor settings in `.vscode/` stay
local and are not tracked.

`ng serve` and `ng lint` are Angular CLI commands. There is no built-in `ng format`
command: use `bun run format` or `npm run format` to invoke Prettier.

## Frontend Docker image

From the repository root (Docker must be installed and running):

```bash
docker build -t checkbox-detection-frontend ./frontend

docker run --rm -p 4200:8080 \
  -e API_UPSTREAM=http://host.docker.internal:8080 \
  checkbox-detection-frontend
```

Open http://localhost:4200. On Linux, add
`--add-host=host.docker.internal:host-gateway` to the run command when connecting
to an API running on your host. The backend must listen on an address reachable
from Docker, rather than only its loopback interface.

The multi-stage image builds with Node 24 and Bun 1.4.4, then serves only the
prerendered browser output using unprivileged Nginx. Node, Bun, source code and
build dependencies are not needed in the runtime image. The existing Node SSR
server remains available for local use; this image serves the currently
prerendered app as static files. If routes later require runtime SSR, the image
will need a Node runtime instead.

Runtime configuration:

| Variable       | Value                                                                                                                 |
| -------------- | --------------------------------------------------------------------------------------------------------------------- |
| `API_UPSTREAM` | Required backend origin, such as `http://backend:8080` or `https://api.example.com`, without a path or trailing slash |
| `PORT`         | Nginx listening port; defaults to `8080`. Match your container port mapping or cloud service port.                    |

Nginx forwards `/detect` to the same path on `API_UPSTREAM`, so backend
addresses stay server-side and the browser uses the frontend's origin. Restart the
container to change configuration; rebuilding the image is unnecessary. The
backend hostname must resolve when Nginx starts.

The container exposes `/healthz` for cloud health probes. Upload requests are capped
at 11 MiB by Nginx, while the API may enforce a lower limit. Terminate public HTTPS
at your cloud ingress/load balancer. To connect a future backend container, put
both containers on the same Docker network and set `API_UPSTREAM` to the backend's
service name and port. This step does not package or deploy the backend.
