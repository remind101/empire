// Package image contains methods and helpers for parsing the docker image
// format. Unfortunately, Docker's codebase does not provide a simple library
// for encoding/decoding the string representation of an image into a consistent
// structure.
//
// The general format is:
//
//	NAME[:TAG|@DIGEST]
//
// Example:
//
//	ubuntu:14.04
//	localhost.localdomain:5000/samalba/hipache:latest
//	localhost:5000/foo/bar@sha256:bc8813ea7b3603864987522f02a76101c17ad122e1c46d790efc0fca78ca7bfb
package image

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
var ErrInvalidImage = errors.New("image: invalid")

// Image represents all the information about an image.
type Image struct {
	// Registry is the registry that the image belongs to.
	Registry string

	// Repository is the repository part of the image.
	Repository string

	// If provided a tag.
	Tag string

	// If provided, a digest for the image.
	Digest string
}

func (i Image) String() string {
	return Encode(i)
}

// Scan implements the sql.Scanner interface.
func (i *Image) Scan(src interface{}) error {
	if src, ok := src.([]byte); ok {
		image, err := Decode(string(src))
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

	image, err := Decode(s)
	if err != nil {
		return err
	}

	*i = image

	return nil
}

// Decode decodes the string representation of an image into an Image structure.
func Decode(in string) (image Image, err error) {
	repo, tag := parseRepositoryTag(in)
	image.Registry, image.Repository = splitRepository(repo)

	if strings.Contains(tag, ":") {
		image.Digest = tag
	} else {
		image.Tag = tag
	}

	if image.Repository == "" {
		err = ErrInvalidImage
		return
	}

	return
}

// Encode encodes an Image to it's string representation.
func Encode(image Image) string {
	repo := image.Repository
	if image.Registry != "" {
		repo = fmt.Sprintf("%s/%s", image.Registry, repo)
	}

	if image.Digest != "" {
		return fmt.Sprintf("%s@%s", repo, image.Digest)
	} else if image.Tag != "" {
		return fmt.Sprintf("%s:%s", repo, image.Tag)
	}

	return repo
}

// splitRepository splits a full docker repo into registry and path segments.
func splitRepository(fullRepo string) (registry string, path string) {
	parts := strings.Split(fullRepo, "/")

	if len(parts) < 2 {
		return "", parts[0]
	}

	if len(parts) == 2 {
		return "", strings.Join(parts, "/")
	}

	return parts[0], strings.Join(parts[1:], "/")
}

// Copied from https://github.com/docker/docker/blob/50a1d0f0ef83a9ed55ea2caaa79539ec835877a3/pkg/parsers/parsers.go#L71-L89
func parseRepositoryTag(repos string) (string, string) {
	n := strings.Index(repos, "@")
	if n >= 0 {
		parts := strings.Split(repos, "@")
		return parts[0], parts[1]
	}
	n = strings.LastIndex(repos, ":")
	if n < 0 {
		return repos, ""
	}
	if tag := repos[n+1:]; !strings.Contains(tag, "/") {
		return repos[:n], tag
	}
	return repos, ""
}
