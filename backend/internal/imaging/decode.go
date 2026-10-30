package imaging

import (
	"errors"
	"fmt"

	"gocv.io/x/gocv"
)

// Decode returns an owned, nonempty BGR matrix. The caller must close it on
// success. On failure any native allocation is released and the returned
// matrix is closed, including when OpenCV returns both a matrix and an error.
func Decode(data []byte) (gocv.Mat, error) {
	img, err := gocv.IMDecode(data, gocv.IMReadColor)
	if err := validateDecodedImage(&img, err); err != nil {
		return gocv.Mat{}, err
	}
	return img, nil
}

// validateDecodedImage takes ownership on failure; success leaves ownership
// with the caller. Closed is checked before Empty because Empty calls native code.
func validateDecodedImage(img *gocv.Mat, decodeErr error) error {
	if decodeErr == nil && !img.Closed() && !img.Empty() {
		return nil
	}
	if !img.Closed() {
		img.Close()
	}
	if decodeErr != nil {
		return fmt.Errorf("decode image: %w", decodeErr)
	}
	return errors.New("decoded image is empty")
}
