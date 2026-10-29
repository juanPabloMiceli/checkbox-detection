# Backend

Reserved for the detection API, which currently lives in a separate directory.
Copy its source and Go module files here when ready. No server is included yet.

The frontend expects `POST http://localhost:8080/api/detect`, accepting a multipart
file field named `image` and returning `{ "boxes": [{ "bbox": [x1, y1, x2, y2],
"is_checked": true }] }`. Allow CORS from `http://localhost:4200`.

After the Go module and server are added, run them from this directory with
`go run .` and `go test ./...`.
