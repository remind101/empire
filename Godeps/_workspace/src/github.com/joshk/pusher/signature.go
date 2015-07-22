package pusher

import (
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/url"
	"sort"
	"strings"
)

type Signature struct {
	key, secret                          string
	method, path, timestamp, authVersion string
	content                              []byte
	queryParameters                      map[string]string
}

type AuthPart struct {
	key, value string
}

type OrderedAuthParts []*AuthPart

func (s OrderedAuthParts) Len() int           { return len(s) }
func (s OrderedAuthParts) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s OrderedAuthParts) Less(i, j int) bool { return s[i].key < s[j].key }

func (s *Signature) Sign() string {
	authParts := []*AuthPart{
		{"auth_key", s.key},
		{"auth_timestamp", s.timestamp},
		{"auth_version", s.authVersion},
		{"body_md5", s.md5Content()},
	}

	for k := range s.queryParameters {
		authParts = append(authParts, &AuthPart{k, s.queryParameters[k]})
	}

	sort.Sort(OrderedAuthParts(authParts))

	sortedAuthParts := []string{}
	for index := range authParts {
		newPart := fmt.Sprintf("%s=%s", authParts[index].key, authParts[index].value)
		sortedAuthParts = append(sortedAuthParts, newPart)
	}

	authPartsQueryString := strings.Join(sortedAuthParts, "&")
	completeAuthParts := fmt.Sprintf("%s\n%s\n%s", s.method, s.path, authPartsQueryString)

	return s.hmacSha256(completeAuthParts)
}

func (s *Signature) EncodedQuery() string {
	query := url.Values{
		"auth_key":       {s.key},
		"auth_timestamp": {s.timestamp},
		"auth_version":   {s.authVersion},
		"body_md5":       {s.md5Content()},
		"auth_signature": {s.Sign()},
	}
	for k := range s.queryParameters {
		query.Add(k, s.queryParameters[k])
	}
	return query.Encode()
}

func (s *Signature) auth_key() string {
	return "auth_key=" + s.key
}

func (s *Signature) auth_timestamp() string {
	return "auth_timestamp=" + s.timestamp
}

func (s *Signature) auth_version() string {
	return "auth_version=" + s.authVersion
}

func (s *Signature) body_md5() string {
	return "body_md5=" + s.md5Content()
}

func (s *Signature) md5Content() string {
	return s.md5(s.content)
}

func (s *Signature) md5(content []byte) string {
	hash := md5.New()
	hash.Write(content)
	return hex.EncodeToString(hash.Sum(nil))
}

func (s *Signature) hmacSha256(content string) string {
	hash := hmac.New(sha256.New, []byte(s.secret))
	io.WriteString(hash, content)
	return hex.EncodeToString(hash.Sum(nil))
}
