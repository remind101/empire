package empire

import (
	"errors"
	"fmt"

	"golang.org/x/net/context"

	"github.com/remind101/empire/pkg/image"
	"github.com/remind101/empire/pkg/jsonmessage"
	"github.com/remind101/empire/procfile"
)

// Example instance: Procfile doesn't exist
type ProcfileError struct {
	Err error
}

func (e *ProcfileError) Error() string {
	return fmt.Sprintf("Procfile not found: %s", e.Err)
}

// Procfile is the name of the Procfile file that Empire will use to
// determine the process formation.
const Procfile = "Procfile"

// ProcfileExtractor represents something that can extract a Procfile from an image.
type ProcfileExtractor interface {
	// ExtractProcfile should return extracted a Procfile from an image, returning
	// it's YAML representation.
	ExtractProcfile(context.Context, image.Image, *jsonmessage.Stream) ([]byte, error)
}

// ProcfileExtractorFunc implements the ProcfileExtractor interface.
type ProcfileExtractorFunc func(context.Context, image.Image, *jsonmessage.Stream) ([]byte, error)

func (fn ProcfileExtractorFunc) ExtractProcfile(ctx context.Context, image image.Image, w *jsonmessage.Stream) ([]byte, error) {
	return fn(ctx, image, w)
}

// ImageRegistry represents something that can interact with container images.
type ImageRegistry interface {
	ProcfileExtractor

	// Resolve should resolve an image to an "immutable" reference of the
	// image.
	Resolve(context.Context, image.Image, *jsonmessage.Stream) (image.Image, error)
}

func formationFromProcfile(p procfile.Procfile) (Formation, error) {
	switch p := p.(type) {
	case procfile.StandardProcfile:
		return formationFromStandardProcfile(p)
	case procfile.ExtendedProcfile:
		return formationFromExtendedProcfile(p)
	default:
		return nil, &ProcfileError{
			Err: errors.New("unknown Procfile format"),
		}
	}
}

func formationFromStandardProcfile(p procfile.StandardProcfile) (Formation, error) {
	f := make(Formation)

	for name, command := range p {
		cmd, err := ParseCommand(command)
		if err != nil {
			return nil, err
		}

		f[name] = Process{
			Command: cmd,
		}
	}

	return f, nil
}

func formationFromExtendedProcfile(p procfile.ExtendedProcfile) (Formation, error) {
	f := make(Formation)

	for name, process := range p {
		var cmd Command
		var err error

		switch command := process.Command.(type) {
		case string:
			cmd, err = ParseCommand(command)
			if err != nil {
				return nil, err
			}
		case []interface{}:
			for _, v := range command {
				cmd = append(cmd, v.(string))
			}
		default:
			return nil, errors.New("unknown command format")
		}

		var ports []Port

		for _, port := range process.Ports {
			protocol := port.Protocol
			if protocol == "" {
				protocol = protocolFromPort(port.Host)
			}

			ports = append(ports, Port{
				Host:      port.Host,
				Container: port.Container,
				Protocol:  protocol,
			})
		}

		f[name] = Process{
			Command:     cmd,
			Cron:        process.Cron,
			NoService:   process.NoService,
			Ports:       ports,
			Environment: process.Environment,
		}
	}

	return f, nil
}

// protocolFromPort attempts to automatically determine what protocol a port
// should use. For example, port 80 is well known to be http, so we can assume
// that http should be used. Defaults to "tcp" if unknown.
func protocolFromPort(port int) string {
	switch port {
	case 80, 8080:
		return "http"
	case 443:
		return "https"
	default:
		return "tcp"
	}
}
