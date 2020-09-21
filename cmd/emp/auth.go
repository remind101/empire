package main

import (
	"context"
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
	"net"
	"net/http"
	"runtime"
	"sync"

	"net/url"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/bgentry/speakeasy"
	"github.com/remind101/empire/cmd/emp/hkclient"
	"github.com/remind101/empire/pkg/heroku"
)

var cmdAuthorize = &Command{
	Run:      runAuthorize,
	Usage:    "authorize",
	Category: "emp",
	NumArgs:  0,
	Short:    "procure a temporary privileged token" + extra,
	Long: `
Have heroku-agent procure and store a temporary privileged token
that will bypass any requirement for a second authentication factor.

Example:

    $ emp authorize
    Enter email: user@test.com
	Enter two-factor auth code: 
    Authorization successful.
`,
}

func runAuthorize(cmd *Command, args []string) {
	cmd.AssertNumArgsCorrect(args)

	if os.Getenv("HEROKU_AGENT_SOCK") == "" {
		printFatal("Authorize must be used with heroku-agent; please set " +
			"HEROKU_AGENT_SOCK")
	}

	var twoFactorCode string
	fmt.Printf("Enter two-factor auth code: ")
	if _, err := fmt.Scanln(&twoFactorCode); err != nil {
		printFatal("reading two-factor auth code: " + err.Error())
	}

	client.AdditionalHeaders.Set("Heroku-Two-Factor-Code", twoFactorCode)

	// No-op: GET /apps with max=0. heroku-agent will detect that a two-factor
	// code was included and attempt to procure a temporary token. This token
	// will then be re-used automatically on subsequent requests.
	_, err := client.AppList(&heroku.ListRange{Field: "name", Max: 0})
	must(err)

	fmt.Println("Authorization successful.")
}

var cmdCreds = &Command{
	Run:      runCreds,
	Usage:    "creds",
	Category: "emp",
	Short:    "show credentials" + extra,
	Long:     `Creds shows credentials that will be used for API calls.`,
}

func runCreds(cmd *Command, args []string) {
	var err error

	nrc, err = hkclient.LoadNetRc()
	if err != nil {
		printFatal(err.Error())
	}

	u, err := url.Parse(apiURL)
	if err != nil {
		printFatal("could not parse API url: " + err.Error())
	}

	user, pass, err := nrc.GetCreds(u)
	if err != nil {
		printFatal("could not get credentials: " + err.Error())
	}

	fmt.Println(user, pass)
}

var cmdWebLogin = &Command{
	Run:      runWebLogin,
	Usage:    "weblogin",
	Category: "emp",
	NumArgs:  0,
	Short:    "Trigger a web-authentication workflow",
	Long: `
Ask the empire server to provide a URL that will trigger the web authentication 
workflow, then open that URL in a browser and wait for authentication to 
complete by providing token via a callback URL
`,
}

type OAuthWebFlow struct {
	wg *sync.WaitGroup
	srv *http.Server
	port int
	email string // The email associated with the GitHub token
	githubToken string // A GitHub bearer token for API access
	empireToken string // A signed JTW containing details about the Empire login
	empireAddress string // The server address for the empire token
	startUrl string
}

func (wf*OAuthWebFlow) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/oauth/token":
		wf.handleOAuthToken(w, r)
		return

	case "/oauth/failure":
		wf.handleOAuthError(w, r)
		return

	case "/started":
		fmt.Fprintf(w, "OK!")
		return
	}
}

func (wf*OAuthWebFlow) handleOAuthToken(w http.ResponseWriter, r *http.Request) {
	defer wf.wg.Done() // let main know we are done cleaning up
	wf.githubToken = r.FormValue("token")
	fmt.Fprintf(w, "You may close this browser window and return to your empire client\n")
}

func (wf*OAuthWebFlow) handleOAuthError(w http.ResponseWriter, r *http.Request) {
	defer wf.wg.Done() // let main know we are done cleaning up
	fmt.Fprintf(w, r.FormValue("err"))
}

func (wf*OAuthWebFlow) waitForGithubToken() {
	// Wait for success or failure
	wf.wg.Wait()
	// Shut down the http server
	wf.srv.Shutdown(context.Background())

	if wf.githubToken == "" {
		printFatal("No token was received, check your browser for an error")
	}
}

func (wf*OAuthWebFlow) fetchEmpireToken() {
	var err error
	// Use the token directly as part of the basic-auth header, with the special $token$ username
	// This tells the server to treat the password as like the token it would have received from
	// the original username/password authentication endpoint
	wf.empireAddress, wf.empireToken, err = attemptLogin("$token$", wf.githubToken, "")
	if err != nil {
		printFatal(err.Error())
	}
}

func (wf*OAuthWebFlow) extractEmail() {
	// We don't have the email associated with the token, but we can get it from the JWT
	token := parseUnsignedToken(wf.empireToken)

	userEntry := token.Claims["User"]
	if userEntry == nil {
		printError("Unable to parse token from server: no User details")
	}

	user := userEntry.(map[string]interface{})
	emailEntry := user["Email"]

	if emailEntry != nil {
		wf.email = emailEntry.(string)
	} else {
		ctx := context.Background()
		ts := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: wf.githubToken},
		)
		tc := oauth2.NewClient(ctx, ts)
		client := github.NewClient(tc)
		user, _, err := client.Users.Get("")
		if err != nil {
			printFatal(err.Error())
		}
		wf.email = *user.Email
	}

	if wf.email == "" {
		printFatal("Unable to parse email address from token")
	}
}

func (wf*OAuthWebFlow) persist() {
	if wf.empireAddress == "" || wf.email == "" || wf.empireToken == "" {
		printFatal("Unable to persist credentials, login failed")
		return
	}
	// Save the credentials to disk using the shared logic with the old `login` command
	persistCredentials(wf.empireAddress, wf.email, wf.empireToken)
}

func newWebFlow() *OAuthWebFlow {
	// Create the base object
	wf := &OAuthWebFlow{
		wg: &sync.WaitGroup{},
	}
	wf.wg.Add(1)

	{
		// Bind to a free local port
		listener, err := net.Listen("tcp", ":0")
		if err != nil {
			printFatal(err.Error())
		}

		wf.port = listener.Addr().(*net.TCPAddr).Port

		wf.srv = &http.Server{
			Handler: wf,
		}

		// Start the server, record the port on which we're listening
		go func() {
			err = wf.srv.Serve(listener)
			if err != nil && err != http.ErrServerClosed {
				panic(err)
			}
		}()
	}


	// Wait for the server to start up
	{
		client := http.Client{ Timeout: 5 * time.Millisecond, }
		startedUrl := fmt.Sprintf("http://localhost:%d/started", wf.port)
		for {
			_, err := client.Get(startedUrl)
			if err == nil {
				break;
			}
			time.Sleep(200 * time.Millisecond)
		}
	}

	// Generate the starting URL to initiate the web flow in the browser
	{
		u, err := url.Parse(client.URL)
		if err != nil {
			panic(err)
		}
		u.Path = "/oauth/start"
		q, err := url.ParseQuery(u.RawQuery)
		if err != nil {
			panic(err)
		}
		q.Add("port", fmt.Sprintf("%d", wf.port))
		u.RawQuery = q.Encode()
		wf.startUrl = u.String()
	}

	return wf
}


func openBrowser(url string) {
	var err error
	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	default:
		err = fmt.Errorf("unsupported platform")
	}
	if err != nil {
		printFatal(err.Error())
	}
}

func parseUnsignedToken(tokenString string) *jwt.Token {
	// We're passing a nonsense keyFunc because we're not actually going to attempt to
	// validate the signature
	token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
		return "", nil
	})

	// We can't validate the signature as the client, so ignore ONLY that specific error
	if err != nil && err.(*jwt.ValidationError).Errors != jwt.ValidationErrorSignatureInvalid {
		printFatal(err.Error())
	}

	return token
}

func runWebLogin(cmd *Command, args []string) {
	// Creating a web flow automatically starts the
	wf := newWebFlow()

	// Currently the start URL is the Empire API server. However, a better use experience would probably involve
	// Opening the browser to a localhost address and having the webFlow handler write out some HTML that will
	// create a oop-up window for the web flow, which can then be reliably closed by HTML/JS written by
	// handleOAuthToken
	go openBrowser(wf.startUrl)

	wf.waitForGithubToken()

	wf.fetchEmpireToken()

	wf.extractEmail()

	wf.persist()

	fmt.Println("Logged in.")
}

var cmdLogin = &Command{
	Run:      runLogin,
	Usage:    "login",
	Category: "emp",
	NumArgs:  0,
	Short:    "log in to your Heroku account" + extra,
	Long: `
Log in with your Heroku credentials. Input is accepted by typing
on the terminal. On unix machines, you can also pipe a password
on standard input.

Example:

    $ emp login
    Enter email: user@test.com
    Enter password: 
    Login successful.
`,
}

func runLogin(cmd *Command, args []string) {
	cmd.AssertNumArgsCorrect(args)
	fmt.Println("The login command is deprecated and will stop working in Nov 2020.  Please use weblogin.")

	oldEmail := client.Username
	var email string
	if oldEmail == "" {
		fmt.Printf("Enter email: ")
	} else {
		fmt.Printf("Enter email [%s]: ", oldEmail)
	}
	_, err := fmt.Scanln(&email)
	switch {
	case err != nil && err.Error() != "unexpected newline":
		printFatal(err.Error())
	case email == "" && oldEmail == "":
		printFatal("email is required.")
	case email == "":
		email = oldEmail
	}

	// NOTE: gopass doesn't support multi-byte chars on Windows
	password, err := readPassword("Enter password: ")
	switch {
	case err == nil:
	case err.Error() == "unexpected newline":
		printFatal("password is required.")
	default:
		printFatal(err.Error())
	}

	address, token, err := attemptLogin(email, password, "")
	if err != nil {
		if herror, ok := err.(heroku.Error); ok && herror.Id == "two_factor" {
			// 2FA requested, attempt 2FA login
			var twoFactorCode string
			fmt.Printf("Enter two-factor auth code: ")
			if _, err := fmt.Scanln(&twoFactorCode); err != nil {
				printFatal("reading two-factor auth code: " + err.Error())
			}
			address, token, err = attemptLogin(email, password, twoFactorCode)
			must(err)
		} else {
			must(err)
		}
	}


	persistCredentials(address, email, token)
	fmt.Println("Logged in.")
}

func persistCredentials(address string, email string, token string) {
	nrc, err := hkclient.LoadNetRc()
	if err != nil {
		printFatal("loading netrc: " + err.Error())
	}

	err = nrc.SaveCreds(address, email, token)
	if err != nil {
		printFatal("saving new token: " + err.Error())
	}
}

func readPassword(prompt string) (password string, err error) {
	if acceptPasswordFromStdin && !isTerminalIn {
		_, err = fmt.Scanln(&password)
		return
	}
	// NOTE: speakeasy may not support multi-byte chars on Windows
	return speakeasy.Ask("Enter password: ")
}

func attemptLogin(username, password, twoFactorCode string) (hostname, token string, err error) {
	description := "emp login from " + time.Now().UTC().Format(time.RFC3339)
	expires := 2592000 // 30 days
	opts := heroku.OAuthAuthorizationCreateOpts{
		Description: &description,
		ExpiresIn:   &expires,
	}

	req, err := client.NewRequest("POST", "/oauth/authorizations", &opts, nil)
	if err != nil {
		return "", "", fmt.Errorf("unknown error when creating login request: %s", err.Error())
	}
	req.SetBasicAuth(username, password)

	if twoFactorCode != "" {
		req.Header.Set("Heroku-Two-Factor-Code", twoFactorCode)
	}

	var auth heroku.OAuthAuthorization
	if err = client.DoReq(req, &auth); err != nil {
		return
	}
	if auth.AccessToken == nil {
		return "", "", fmt.Errorf("access token missing from Heroku API login response")
	}
	return req.Host, auth.AccessToken.Token, nil
}

var cmdLogout = &Command{
	Run:      runLogout,
	Usage:    "logout",
	Category: "emp",
	NumArgs:  0,
	Short:    "log out of your Heroku account" + extra,
	Long: `
Log out of your Heroku account and remove credentials from
this machine.

Example:

    $ emp logout
    Logged out.
`,
}

func runLogout(cmd *Command, args []string) {
	cmd.AssertNumArgsCorrect(args)

	u, err := url.Parse(client.URL)
	if err != nil {
		printFatal("couldn't parse client URL: " + err.Error())
	}

	nrc, err = hkclient.LoadNetRc()
	if err != nil {
		printError(err.Error())
	}

	err = removeCreds(strings.Split(u.Host, ":")[0])
	if err != nil {
		printFatal("saving new netrc: " + err.Error())
	}
	fmt.Println("Logged out.")
}
