package main

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/chzyer/readline"
	"github.com/mgutz/ansi"
	"github.com/remind101/empire/cmd/emp/hkclient"
	"github.com/remind101/empire/pkg/heroku"
)

var nrc *hkclient.NetRc

func hkHome() string {
	return filepath.Join(hkclient.HomePath(), ".hk")
}

func netrcPath() string {
	if s := os.Getenv("NETRC_PATH"); s != "" {
		return s
	}

	return filepath.Join(hkclient.HomePath(), netrcFilename)
}

func loadNetrc() {
	var err error

	if nrc == nil {
		if nrc, err = hkclient.LoadNetRc(); err != nil {
			if os.IsNotExist(err) {
				nrc = &hkclient.NetRc{}
				return
			}
			printFatal("loading netrc: " + err.Error())
		}
	}
}

func getCreds(u string) (user, pass string) {
	loadNetrc()
	if nrc == nil {
		return "", ""
	}

	apiURL, err := url.Parse(u)
	if err != nil {
		printFatal("invalid API URL: %s", err)
	}

	user, pass, err = nrc.GetCreds(apiURL)
	if err != nil {
		printError(err.Error())
	}

	return user, pass
}

func saveCreds(host, user, pass string) error {
	loadNetrc()
	m := nrc.FindMachine(host)
	if m == nil || m.IsDefault() {
		m = nrc.NewMachine(host, user, pass, "")
	}
	m.UpdateLogin(user)
	m.UpdatePassword(pass)

	body, err := nrc.MarshalText()
	if err != nil {
		return err
	}
	return ioutil.WriteFile(netrcPath(), body, 0600)
}

func removeCreds(host string) error {
	loadNetrc()
	nrc.RemoveMachine(host)

	body, err := nrc.MarshalText()
	if err != nil {
		return err
	}
	return ioutil.WriteFile(netrcPath(), body, 0600)
}

// exists returns whether the given file or directory exists or not
func fileExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func must(err error) {
	if err != nil {
		if herror, ok := err.(heroku.Error); ok {
			switch herror.Id {
			case "two_factor":
				printError(err.Error() + " Authorize with `emp authorize`.")
				os.Exit(79)
			case "unauthorized":
				printFatal(err.Error() + " Log in with `emp login`.")
			}
		}
		printFatal(err.Error())
	}
}

func printError(message string, args ...interface{}) {
	log.Println(colorizeMessage("red", "error:", message, args...))
}

func printFatal(message string, args ...interface{}) {
	log.Fatal(colorizeMessage("red", "error:", message, args...))
}

func printWarning(message string, args ...interface{}) {
	log.Println(colorizeMessage("yellow", "warning:", message, args...))
}

func mustConfirm(warning, desired string) {
	if isTerminalIn {
		printWarning(warning)
		fmt.Printf("> ")
	}
	var confirm string
	if _, err := fmt.Scanln(&confirm); err != nil {
		printFatal(err.Error())
	}

	if confirm != desired {
		printFatal("Confirmation did not match %q.", desired)
	}
}

func askForMessage() (string, error) {
	if !isTerminalIn {
		return "", errors.New("Can't ask for message")
	}

	fmt.Println("A commit message is required, enter one below:")
	reader, err := readline.New("> ")
	if err != nil {
		return "", err
	}
	message, err := reader.Readline()
	return strings.Trim(message, " \n"), err
}

func colorizeMessage(color, prefix, message string, args ...interface{}) string {
	prefResult := ""
	if prefix != "" {
		prefResult = ansi.Color(prefix, color+"+b") + " " + ansi.ColorCode("reset")
	}
	return prefResult + ansi.Color(fmt.Sprintf(message, args...), color) + ansi.ColorCode("reset")
}

func listRec(w io.Writer, a ...interface{}) {
	for i, x := range a {
		fmt.Fprint(w, x)
		if i+1 < len(a) {
			w.Write([]byte{'\t'})
		} else {
			w.Write([]byte{'\n'})
		}
	}
}

type prettyTime struct {
	time.Time
}

func (s prettyTime) String() string {
	if time.Now().Sub(s.Time) < 12*30*24*time.Hour {
		return s.Local().Format("Jan _2 15:04")
	}
	return s.Local().Format("Jan _2  2006")
}

type prettyDuration struct {
	time.Duration
}

func (a prettyDuration) String() string {
	switch d := a.Duration; {
	case d > 2*24*time.Hour:
		return a.Unit(24*time.Hour, "d")
	case d > 2*time.Hour:
		return a.Unit(time.Hour, "h")
	case d > 2*time.Minute:
		return a.Unit(time.Minute, "m")
	}
	return a.Unit(time.Second, "s")
}

func (a prettyDuration) Unit(u time.Duration, s string) string {
	return fmt.Sprintf("%2d", roundDur(a.Duration, u)) + s
}

func roundDur(d, k time.Duration) int {
	return int((d + k/2 - 1) / k)
}

func abbrev(s string, n int) string {
	if len(s) > n {
		return s[:n-1] + "…"
	}
	return s
}

func ensurePrefix(val, prefix string) string {
	if !strings.HasPrefix(val, prefix) {
		return prefix + val
	}
	return val
}

func ensureSuffix(val, suffix string) string {
	if !strings.HasSuffix(val, suffix) {
		return val + suffix
	}
	return val
}

func openURL(url string) error {
	var command string
	var args []string
	switch runtime.GOOS {
	case "darwin":
		command = "open"
		args = []string{command, url}
	case "windows":
		command = "cmd"
		args = []string{"/c", "start " + strings.Replace(url, "&", "^&", -1)}
	default:
		if _, err := exec.LookPath("xdg-open"); err != nil {
			log.Println("xdg-open is required to open web pages on " + runtime.GOOS)
			os.Exit(2)
		}
		command = "xdg-open"
		args = []string{command, url}
	}
	return runCommand(command, args, os.Environ())
}

func runCommand(command string, args, env []string) error {
	if runtime.GOOS != "windows" {
		p, err := exec.LookPath(command)
		if err != nil {
			log.Printf("Error finding path to %q: %s\n", command, err)
			os.Exit(2)
		}
		command = p
	}
	return sysExec(command, args, env)
}

func stringsIndex(s []string, item string) int {
	for i := range s {
		if s[i] == item {
			return i
		}
	}
	return -1
}

func maybeMessage(action func(cmd *Command, args []string)) func(cmd *Command, args []string) {
	return func(cmd *Command, args []string) {
		defer func() {
			if r := recover(); r != nil {
				e := r.(heroku.Error)
				if e.Id == "message_required" {
					message, err := askForMessage()
					if message == "" || err != nil {
						printFatal("A message is required for this action, please run again with '-m'.")
					}
					flagMessage = message
					action(cmd, args)
				}
			}
		}()

		action(cmd, args)
	}
}
