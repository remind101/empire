package slash_test

import (
	"net/http"
	"regexp"

	"github.com/ejholmes/slash"
	"golang.org/x/net/context"
)

func Example() {
	r := slash.NewMux()
	r.Command("/weather", "secrettoken", slash.HandlerFunc(Weather))

	s := slash.NewServer(r)
	http.ListenAndServe(":8080", s)
}

// Weather is the primary slash handler for the /weather command.
func Weather(ctx context.Context, r slash.Responder, command slash.Command) error {
	h := slash.NewMux()

	var zipcodeRegex = regexp.MustCompile(`(?P<zip>[0-9])`)
	h.MatchText(zipcodeRegex, slash.HandlerFunc(Zipcode))

	return h.ServeCommand(ctx, r, command)
}

// Zipcode is a slash handler that returns the weather for a zip code.
func Zipcode(ctx context.Context, r slash.Responder, command slash.Command) error {
	params := slash.Params(ctx)
	zip := params["zip"]
	return r.Respond(slash.Reply(zip))
}
