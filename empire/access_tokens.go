package empire

import (
	"errors"

	"github.com/dgrijalva/jwt-go"
)

// AccessToken represents a token that allow access to the api.
type AccessToken struct {
	Token string
	User  *User
}

type accessTokensService struct {
	Secret []byte // Secret used to sign jwt tokens.
}

// AccessTokensCreate "creates" the token by jwt signing it and setting the
// Token value.
func (s *accessTokensService) AccessTokensCreate(token *AccessToken) (*AccessToken, error) {
	signed, err := SignToken(s.Secret, token)
	if err != nil {
		return token, err
	}

	token.Token = signed

	return token, nil
}

func (s *accessTokensService) AccessTokensFind(token string) (*AccessToken, error) {
	at, err := ParseToken(s.Secret, token)

	if at != nil {
		at.Token = token
	}

	return at, err
}

// SignToken jwt signs the token and adds the signature to the Token field.
func SignToken(secret []byte, token *AccessToken) (string, error) {
	t := accessTokenToJwt(token)
	return t.SignedString(secret)
}

// ParseToken parses a string token, verifies it, and returns an AccessToken
// instance.
func ParseToken(secret []byte, token string) (*AccessToken, error) {
	t, err := jwtParse(secret, token)

	if err != nil {
		return nil, err
	}

	if !t.Valid {
		return nil, nil
	}

	return jwtToAccessToken(t)
}

func accessTokenToJwt(token *AccessToken) *jwt.Token {
	t := jwt.New(jwt.SigningMethodHS256)
	t.Claims["User"] = struct {
		Name        string
		GitHubToken string
	}{
		Name:        token.User.Name,
		GitHubToken: token.User.GitHubToken,
	}

	return t
}

// jwtToAccessTokens maps a jwt.Token to an AccessToken.
func jwtToAccessToken(t *jwt.Token) (*AccessToken, error) {
	var token AccessToken

	// TODO Should probably return an error here if a user isn't present.
	if u, ok := t.Claims["User"].(map[string]interface{}); ok {
		var user User

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
