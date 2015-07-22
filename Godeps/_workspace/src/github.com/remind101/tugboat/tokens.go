package tugboat

import (
	"errors"

	"github.com/dgrijalva/jwt-go"
)

// ErrInvalidToken is returned when the token provided is not valid.
var ErrInvalidToken = errors.New("invalid token")

// Token represents an authentication token for external providers.
type Token struct {
	Provider string
	Token    string
}

// tokensService represents a service for creating and finding provider tokens.
type tokensService interface {
	TokensCreate(*Token) error
	TokensFind(id string) (*Token, error)
}

func newTokensService(secret []byte) tokensService {
	return &jwtTokensService{
		secret: secret,
	}
}

// jwtTokensService is a tokensService implementation backed by jwt.
type jwtTokensService struct {
	secret []byte
}

func (s *jwtTokensService) TokensCreate(token *Token) error {
	signed, err := signToken(s.secret, token)
	if err != nil {
		return err
	}

	token.Token = signed
	return nil
}

func (s *jwtTokensService) TokensFind(token string) (*Token, error) {
	t, err := parseToken(s.secret, token)
	if err != nil {
		switch err.(type) {
		case *jwt.ValidationError:
			return t, ErrInvalidToken
		default:
			return t, err
		}
	}

	if t != nil {
		t.Token = token
	}

	return t, nil
}

// signToken jwt signs the token and adds the signature to the Token field.
func signToken(secret []byte, token *Token) (string, error) {
	t := tokenToJWT(token)
	return t.SignedString(secret)
}

// parseToken parses a string token, verifies it, and returns an Token
// instance.
func parseToken(secret []byte, token string) (*Token, error) {
	t, err := jwtParse(secret, token)

	if err != nil {
		return nil, err
	}

	if !t.Valid {
		return nil, nil
	}

	return jwtToToken(t)
}

func tokenToJWT(token *Token) *jwt.Token {
	t := jwt.New(jwt.SigningMethodHS256)
	t.Claims["Provider"] = token.Provider
	return t
}

// jwtToToken maps a jwt.Token to an AccessToken.
func jwtToToken(t *jwt.Token) (*Token, error) {
	var token Token

	if p, ok := t.Claims["Provider"].(string); ok {
		token.Provider = p
	} else {
		return &token, errors.New("missing provider")
	}

	return &token, nil
}

func jwtParse(secret []byte, token string) (*jwt.Token, error) {
	return jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		return secret, nil
	})
}
