package httpapi

import (
	"errors"
	"io"
	"mime"
	"net/http"
)

const maxUploadBytes = 20 << 20

type requestError struct {
	status  int
	message string
}

func (e *requestError) Error() string { return e.message }

// readUpload returns encoded bytes and closes the request body and temporary
// multipart files before computation. The byte cap includes multipart overhead.
func readUpload(w http.ResponseWriter, r *http.Request) ([]byte, error) {
	defer r.Body.Close()
	if r.ContentLength > maxUploadBytes {
		return nil, uploadTooLarge()
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadBytes)
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil {
		return nil, unsupportedUpload()
	}
	switch mediaType {
	case "multipart/form-data":
		err := r.ParseMultipartForm(4 << 20)
		if r.MultipartForm != nil {
			defer r.MultipartForm.RemoveAll()
		}
		if err != nil {
			return nil, uploadReadError(err)
		}
		file, _, err := r.FormFile("image")
		if errors.Is(err, http.ErrMissingFile) {
			file, _, err = r.FormFile("file")
		}
		if err != nil {
			return nil, &requestError{status: http.StatusBadRequest, message: "upload an image using the 'image' or 'file' field"}
		}
		defer file.Close()
		data, err := io.ReadAll(file)
		if err != nil {
			return nil, uploadReadError(err)
		}
		return data, nil
	case "image/png", "image/jpeg":
		data, err := io.ReadAll(r.Body)
		if err != nil {
			return nil, uploadReadError(err)
		}
		return data, nil
	default:
		return nil, unsupportedUpload()
	}
}

func uploadTooLarge() error {
	return &requestError{status: http.StatusRequestEntityTooLarge, message: "request exceeds the 20 MiB upload limit"}
}

func unsupportedUpload() error {
	return &requestError{status: http.StatusUnsupportedMediaType, message: "send multipart/form-data, image/png, or image/jpeg"}
}

func uploadReadError(err error) error {
	var limitError *http.MaxBytesError
	if errors.As(err, &limitError) {
		return uploadTooLarge()
	}
	return &requestError{status: http.StatusBadRequest, message: "could not read image upload"}
}
