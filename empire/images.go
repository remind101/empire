package empire

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// DefaultTag is used when json decoding an image. If there is no tag part
// present, this will be used as the tag.
const DefaultTag = "latest"

// ErrInvalidImage is returned when the image does not specify a repo.
var ErrInvalidImage = errors.New("invalid image")

// Image represents a container image, which is tied to a repository.
type Image struct {
	ID   string
	Repo string
}

func (i Image) String() string {
	return encodeImage(i)
}

// Scan implements the sql.Scanner interface.
func (i *Image) Scan(src interface{}) error {
	if src, ok := src.([]byte); ok {
		image, err := decodeImage(string(src))
		if err != nil {
			return err
		}

		*i = image
	}

	return nil
}

// Value implements the driver.Value interface.
func (i Image) Value() (driver.Value, error) {
	return driver.Value(i.String()), nil
}

func (i *Image) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}

	image, err := decodeImage(s)
	if err != nil {
		return err
	}

	*i = image

	return nil
}

func encodeImage(i Image) string {
	return fmt.Sprintf("%s:%s", i.Repo, i.ID)
}

func decodeImage(s string) (image Image, err error) {
	p := strings.Split(s, ":")

	if len(p) == 0 {
		err = ErrInvalidImage
		return
	}

	image.Repo = p[0]

	if image.Repo == "" {
		err = ErrInvalidImage
		return
	}

	if len(p) == 1 {
		image.ID = DefaultTag
		return
	}

	image.ID = p[1]
	return
}
