package main

import (
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net"
	"net/http/httputil"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/docker/docker/pkg/term"
	"github.com/remind101/empire/pkg/heroku"
)

var (
	detachedRun bool
	dynoSize    string
)

var cmdRun = &Command{
	Run:             maybeMessage(runRun),
	Usage:           "run [-s <size>] [-d] <command> [<argument>...]",
	NeedsApp:        true,
	OptionalMessage: true,
	Category:        "dyno",
	Short:           "run a process in a dyno",
	Long: `
Run a process on Heroku. Flags such as` + " `-a` " + `may be parsed out of
the command unless the command is quoted or provided after a
double-dash (--).

Options:

    -s <size>  set the size for this dyno (e.g. 2X)
    -d         run in detached mode instead of attached to terminal

Examples:

    $ emp run echo "hello"
    Running ` + "`echo \"hello\"`" + ` on myapp as run.1234:
    "hello"

    $ emp run console
    Running ` + "`console`" + ` on myapp as run.5678:
    Loading production environment (Rails 3.2.14)
    irb(main):001:0> ...

    $ emp run -d -s 2X bin/my_worker
    Ran ` + "`bin/my_worker`" + ` on myapp as run.4321, detached.

    $ emp run -a myapp -- ls -a /
    Running ` + "`ls -a bin /`" + ` on myapp as run.8650:
    /:
    .  ..  app  bin  dev  etc  home  lib  lib64  lost+found  proc  sbin  tmp  usr  var
`,
}

func init() {
	cmdRun.Flag.BoolVarP(&detachedRun, "detached", "d", false, "detached")
	cmdRun.Flag.StringVarP(&dynoSize, "size", "s", "", "dyno size")
}

func runRun(cmd *Command, args []string) {
	if len(args) == 0 {
		cmd.PrintUsage()
		os.Exit(2)
	}
	appname := mustApp()
	message := getMessage()

	w, err := term.GetWinsize(inFd)
	if err != nil {
		// If syscall.TIOCGWINSZ is not supported by the device, we're
		// probably trying to run tests. Set w to some sensible default.
		if err.Error() == "operation not supported by device" || err.Error() == "inappropriate ioctl for device" {
			w = &term.Winsize{
				Height: 20,
				Width:  80,
			}
		} else {
			printFatal(err.Error())
		}
	}

	attached := !detachedRun
	opts := heroku.DynoCreateOpts{Attach: &attached, Message: message}
	if attached {
		env := map[string]string{
			"COLUMNS": strconv.Itoa(int(w.Width)),
			"LINES":   strconv.Itoa(int(w.Height)),
			"TERM":    os.Getenv("TERM"),
		}
		opts.Env = &env
	}
	if dynoSize != "" {
		opts.Size = &dynoSize
	}

	command := strings.Join(args, " ")
	if detachedRun {
		dyno, err := client.DynoCreate(appname, command, &opts)
		must(err)

		log.Printf("Ran `%s` on %s as %s, detached.", dyno.Command, appname, dyno.Name)
		return
	}

	params := struct {
		Command string             `json:"command"`
		Attach  *bool              `json:"attach,omitempty"`
		Env     *map[string]string `json:"env,omitempty"`
		Size    *string            `json:"size,omitempty"`
	}{
		Command: command,
		Attach:  opts.Attach,
		Env:     opts.Env,
		Size:    opts.Size,
	}

	rh := heroku.RequestHeaders{CommitMessage: message}
	req, err := client.NewRequest("POST", "/apps/"+appname+"/dynos", params, rh.Headers())
	must(err)

	u, err := url.Parse(apiURL)
	must(err)

	proto, address := dialParams(u)

	var dial net.Conn
	if proto == "tls" {
		dial, err = tlsDial("tcp", address, &tls.Config{})
		if err != nil {
			printFatal(err.Error())
		}
	} else {
		dial, err = net.Dial(proto, address)
		if err != nil {
			printFatal(err.Error())
		}
	}

	clientconn := httputil.NewClientConn(dial, nil)
	defer clientconn.Close()
	res, err := clientconn.Do(req)
	defer res.Body.Close()
	if err = heroku.CheckResp(res); err != nil {
		printFatal(err.Error())
	}
	if err != nil && err != httputil.ErrPersistEOF {
		printFatal(err.Error())
	}
	rwc, br := clientconn.Hijack()
	defer rwc.Close()

	if isTerminalIn && isTerminalOut {
		state, err := term.SetRawTerminal(inFd)
		if err != nil {
			printFatal(err.Error())
		}
		defer term.RestoreTerminal(inFd, state)
	}

	errChanOut := make(chan error, 1)
	errChanIn := make(chan error, 1)
	exit := make(chan bool)
	go func() {
		defer close(exit)
		defer close(errChanOut)
		var err error
		_, err = io.Copy(os.Stdout, br)
		errChanOut <- err
	}()
	go func() {
		_, err := io.Copy(rwc, os.Stdin)
		errChanIn <- err
		rwc.(interface {
			CloseWrite() error
		}).CloseWrite()
	}()
	<-exit
	select {
	case err = <-errChanIn:
		must(err)
	case err = <-errChanOut:
		must(err)
	}
}

func dialParams(u *url.URL) (proto, address string) {
	// u.Host can be either host or host:port
	host, port := splitHost(u.Host)

	switch u.Scheme {
	case "https":
		proto = "tls"
		if port == "" {
			port = "443"
		}
	case "http":
		proto = "tcp"
		if port == "" {
			port = "80"
		}
	default:
		panic(fmt.Sprintf("I don't know what to do for the %v protocol", u.Scheme))
	}

	address = fmt.Sprintf("%s:%s", host, port)

	return
}

func splitHost(hostPort string) (host, port string) {
	parts := strings.SplitN(hostPort, ":", 2)
	host = parts[0]
	if len(parts) == 2 {
		port = parts[1]
	}
	return
}
