package empire

import (
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/ejholmes/cloudwatch"

	"code.google.com/p/go-uuid/uuid"

	"golang.org/x/net/context"
)

// RunRecorder is a function that returns an io.Writer that will be written to
// to record Stdout and Stdin of interactive runs.
type RunRecorder func() (io.Writer, error)

// RecordToCloudWatch returns a RunRecorder that writes the log record to
// CloudWatch Logs.
func RecordToCloudWatch(group string, config client.ConfigProvider) RunRecorder {
	c := cloudwatchlogs.New(config)
	g := cloudwatch.NewGroup(group, c)
	return func() (io.Writer, error) {
		stream := uuid.New()
		w, err := g.Create(stream)
		if err != nil {
			return nil, err
		}

		url := fmt.Sprintf("https://console.aws.amazon.com/cloudwatch/home?region=%s#logEvent:group=%s;stream=%s", *c.Config.Region, group, stream)
		return &writerWithURL{w, url}, nil
	}
}

// writerWithURL is an io.Writer that has a URL() method.
type writerWithURL struct {
	io.Writer
	url string
}

// URL returns the underyling url.
func (w *writerWithURL) URL() string {
	return w.url
}

// RecordTo returns a RunRecorder that writes the log record to the io.Writer
func RecordTo(w io.Writer) RunRecorder {
	return func() (io.Writer, error) {
		return w, nil
	}
}

type runnerService struct {
	*Empire
}

func (r *runnerService) Run(ctx context.Context, opts RunOpts) error {
	release, err := releasesFind(r.db, ReleasesQuery{App: opts.App})
	if err != nil {
		return err
	}

	procName := opts.Command[0]
	proc := Process{
		Quantity: 1,
	}

	if cmd, ok := release.Formation[procName]; ok {
		proc.Command = append(cmd.Command, opts.Command[1:])
	} else {
		if r.RequireWhitelistedRuns {
			return &CommandWhitelistError{Command: opts.Command}
		}

		// This is an unnamed command, fallback to a generic proc name.
		procName = "run"
		proc.Command = opts.Command
	}

	// Set the size of the process.
	constraints := DefaultConstraints
	if opts.Constraints != nil {
		constraints = *opts.Constraints
	}
	proc.SetConstraints(constraints)

	a := newSchedulerApp(release)
	p := newSchedulerProcess(release, procName, proc)
	p.Labels["empire.user"] = opts.User.Name

	// Add additional environment variables to the process.
	for k, v := range opts.Env {
		p.Env[k] = v
	}

	return r.Scheduler.Run(ctx, a, p, opts.Input, opts.Output)
}
