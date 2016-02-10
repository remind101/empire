package conveyor

import (
	"fmt"
	"io"
)

func (s *Service) LogsStream(w io.Writer, buildIdentity string) error {
	return s.Get(w, fmt.Sprintf("/logs/%s", buildIdentity), nil, nil)
}
