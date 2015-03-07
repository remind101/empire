package empire

import "github.com/dgrijalva/jwt-go"

// AccessToken represents a token that allow access to the api.
type AccessToken struct {
	Token       string `json:"token"`
	GitHubToken string `json:"-"`
}

type AccessTokensService interface {
	AccessTokensCreate(*AccessToken) (*AccessToken, error)
}

// an implementation of the accessTokensService backed by JWT signed tokens.
type accessTokensService struct {
	Secret []byte // Secret used to sign jwt tokens.
}

func (s *accessTokensService) AccessTokensCreate(token *AccessToken) (*AccessToken, error) {
	return token, signToken(s.Secret, token)
}

// signToken jwt signs the token and adds the signature to the Token field.
func signToken(secret []byte, token *AccessToken) error {
	t := jwt.New(jwt.SigningMethodHS256)
	t.Claims["github_token"] = token.GitHubToken

	signed, err := t.SignedString(secret)
	if err != nil {
		return err
	}

	token.Token = signed

	return err
}
