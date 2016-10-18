package heroku

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/dgrijalva/jwt-go"
	"github.com/remind101/empire"
	"github.com/remind101/empire/server/auth"
	"github.com/remind101/pkg/logger"
	"github.com/remind101/pkg/reporter"
	"golang.org/x/net/context"
)

// AccessToken represents a token that allow access to the api.
type AccessToken struct {
	// The encoded token.
	Token string

	// The user that this AccessToken belongs to.
	User *empire.User
}

// IsValid returns nil if the AccessToken is valid.
func (t *AccessToken) IsValid() error {
	if err := t.User.IsValid(); err != nil {
		return err
	}

	return nil
}

// ServeHTTPContext implements the httpx.Handler interface. It will ensure that
// there is a Bearer token present and that it is valid.
func (s *Server) Authenticate(ctx context.Context, r *http.Request) (context.Context, error) {
	username, password, ok := r.BasicAuth()
	if !ok {
		return nil, ErrUnauthorized
	}

	// Add an auth strategy for authenticating with an access token.
	auther := s.Auth.AddAuthenticator(&accessTokenAuthenticator{
		findAccessToken: s.AccessTokensFind,
	})

	ctx, err := auther.Authenticate(ctx, username, password, r.Header.Get(HeaderTwoFactor))
	if err != nil {
		switch err {
		case auth.ErrTwoFactor:
			return nil, ErrTwoFactor
		case auth.ErrForbidden:
			return nil, ErrUnauthorized
		}

		if err, ok := err.(*auth.UnauthorizedError); ok {
			return nil, errUnauthorized(err)
		}

		return nil, &ErrorResource{
			Status:  http.StatusForbidden,
			ID:      "forbidden",
			Message: err.Error(),
		}
	}

	user := auth.UserFromContext(ctx)

	logger.Info(ctx,
		"authenticated",
		"user", user.Name,
	)

	reporter.AddContext(ctx, "user", user.Name)

	return ctx, nil
}

// accessTokenAuthenticator is an Authenticator that uses empire JWT access tokens to
// authenticate.
type accessTokenAuthenticator struct {
	findAccessToken func(string) (*AccessToken, error)
}

// Authenticate authenticates the access token, which should be provided as the
// password parameter. Username and otp are ignored.
func (a *accessTokenAuthenticator) Authenticate(_ string, token string, _ string) (*empire.User, error) {
	at, err := a.findAccessToken(token)
	if err != nil {
		return nil, err
	}

	if at == nil {
		return nil, auth.ErrForbidden
	}

	return at.User, nil
}

// AccessTokensCreate "creates" the token by jwt signing it and setting the
// Token value.
func (s *Server) AccessTokensCreate(token *AccessToken) (*AccessToken, error) {
	signed, err := signToken(s.Secret, token)
	if err != nil {
		return token, err
	}

	token.Token = signed

	return token, token.IsValid()
}

func (s *Server) AccessTokensFind(token string) (*AccessToken, error) {
	at, err := parseToken(s.Secret, token)
	if err != nil {
		if err == errTokenIncompatible {
			return nil, nil
		}

		switch err.(type) {
		case *jwt.ValidationError:
			return nil, nil
		default:
			return at, fmt.Errorf("error parsing token: %v", err)
		}
	}

	if at != nil {
		at.Token = token
	}

	return at, at.IsValid()
}

// signToken jwt signs the token and adds the signature to the Token field.
func signToken(secret []byte, token *AccessToken) (string, error) {
	t := accessTokenToJwt(token)
	return t.SignedString(secret)
}

// parseToken parses a string token, verifies it, and returns an AccessToken
// instance.
func parseToken(secret []byte, token string) (*AccessToken, error) {
	t, err := jwtParse(secret, token)

	if err != nil {
		return nil, err
	}

	if !t.Valid {
		return nil, nil
	}

	return jwtToAccessToken(t)
}

// When changes to the token format are not backwards compatible, this should
// incremented. The user will be asked to re-authenticate.
var tokenCompatibilityVersion = "v1"
var errTokenIncompatible = errors.New("token incompatible")

func accessTokenToJwt(token *AccessToken) *jwt.Token {
	t := jwt.New(jwt.SigningMethodHS256)
	t.Claims["Version"] = tokenCompatibilityVersion
	t.Claims["User"] = struct {
		ID          string
		Name        string
		GitHubToken string
	}{
		ID:          token.User.ID,
		Name:        token.User.Name,
		GitHubToken: token.User.GitHubToken,
	}

	return t
}

// jwtToAccessTokens maps a jwt.Token to an AccessToken.
func jwtToAccessToken(t *jwt.Token) (*AccessToken, error) {
	var token AccessToken

	v, ok := t.Claims["Version"]
	if !ok || v != tokenCompatibilityVersion {
		return nil, errTokenIncompatible
	}

	if u, ok := t.Claims["User"].(map[string]interface{}); ok {
		var user empire.User

		if id, ok := u["ID"].(string); ok {
			user.ID = id
		} else {
			return &token, errors.New("missing id")
		}

		if n, ok := u["Name"].(string); ok {
			user.Name = n
		} else {
			return &token, errors.New("missing name")
		}

		if gt, ok := u["GitHubToken"].(string); ok {
			user.GitHubToken = gt
		} else {
			return &token, errors.New("missing github token")
		}

		token.User = &user
	} else {
		return &token, errors.New("missing user")
	}

	return &token, nil
}

func jwtParse(secret []byte, token string) (*jwt.Token, error) {
	return jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		return secret, nil
	})
}
