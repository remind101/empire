package saml

import (
	"bytes"
	"compress/flate"
	"encoding/base64"
	"encoding/pem"
	"encoding/xml"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"regexp"
	"time"

	"github.com/crewjam/go-xmlsec"
	"github.com/remind101/empire/pkg/jwt"
)

// ServiceProvider implements SAML Service provider.
//
// In SAML, service providers delegate responsibility for identifying
// clients to an identity provider. If you are writing an application
// that uses passwords (or whatever) stored somewhere else, then you
// are service provider.
//
// See the example directory for an example of a web application using
// the service provider interface.
type ServiceProvider struct {
	// Key is the RSA private key we use to sign requests.
	Key string

	// Certificate is the RSA public part of Key.
	Certificate string

	// MetadataURL is the full URL to the metadata endpoint on this host,
	// i.e. https://example.com/saml/metadata
	MetadataURL string

	// AcsURL is the full URL to the SAML Assertion Customer Service endpoint
	// on this host, i.e. https://example.com/saml/acs
	AcsURL string

	// IDPMetadata is the metadata from the identity provider.
	IDPMetadata *Metadata
}

// MaxIssueDelay is the longest allowed time between when a SAML assertion is
// issued by the IDP and the time it is received by ParseResponse. (In practice
// this is the maximum allowed clock drift between the SP and the IDP).
const MaxIssueDelay = time.Second * 90

// DefaultValidDuration is how long we assert that the SP metadata is valid.
const DefaultValidDuration = time.Hour * 24 * 2

// DefaultCacheDuration is how long we ask the IDP to cache the SP metadata.
const DefaultCacheDuration = time.Hour * 24 * 1

// Metadata returns the service provider metadata
func (sp *ServiceProvider) Metadata() *Metadata {
	if cert, _ := pem.Decode([]byte(sp.Certificate)); cert != nil {
		sp.Certificate = base64.StdEncoding.EncodeToString(cert.Bytes)
	}

	return &Metadata{
		EntityID:   sp.MetadataURL,
		ValidUntil: TimeNow().Add(DefaultValidDuration),
		SPSSODescriptor: &SPSSODescriptor{
			AuthnRequestsSigned:        false,
			WantAssertionsSigned:       true,
			ProtocolSupportEnumeration: "urn:oasis:names:tc:SAML:2.0:protocol",
			KeyDescriptor: []KeyDescriptor{
				{
					Use: "signing",
					KeyInfo: KeyInfo{
						Certificate: sp.Certificate,
					},
				},
				{
					Use: "encryption",
					KeyInfo: KeyInfo{
						Certificate: sp.Certificate,
					},
					EncryptionMethods: []EncryptionMethod{
						{Algorithm: "http://www.w3.org/2001/04/xmlenc#aes128-cbc"},
						{Algorithm: "http://www.w3.org/2001/04/xmlenc#aes192-cbc"},
						{Algorithm: "http://www.w3.org/2001/04/xmlenc#aes256-cbc"},
						{Algorithm: "http://www.w3.org/2001/04/xmlenc#rsa-oaep-mgf1p"},
					},
				},
			},
			AssertionConsumerService: []IndexedEndpoint{{
				Binding:  HTTPPostBinding,
				Location: sp.AcsURL,
				Index:    1,
			}},
		},
	}
}

// MakeRedirectAuthenticationRequest creates a SAML authentication request using
// the HTTP-Redirect binding. It returns a URL that we will redirect the user to
// in order to start the auth process.
func (sp *ServiceProvider) MakeRedirectAuthenticationRequest(relayState string) (*url.URL, error) {
	req, err := sp.MakeAuthenticationRequest(sp.GetSSOBindingLocation(HTTPRedirectBinding))
	if err != nil {
		return nil, err
	}
	return req.Redirect(relayState), nil
}

// Redirect returns a URL suitable for using the redirect binding with the request
func (req *AuthnRequest) Redirect(relayState string) *url.URL {
	w := &bytes.Buffer{}
	w1 := base64.NewEncoder(base64.StdEncoding, w)
	w2, _ := flate.NewWriter(w1, 9)
	if err := xml.NewEncoder(w2).Encode(req); err != nil {
		panic(err)
	}
	w2.Close()
	w1.Close()

	rv, _ := url.Parse(req.Destination)

	query := rv.Query()
	query.Set("SAMLRequest", string(w.Bytes()))
	if relayState != "" {
		query.Set("RelayState", relayState)
	}
	rv.RawQuery = query.Encode()

	return rv
}

// GetSSOBindingLocation returns URL for the IDP's Single Sign On Service binding
// of the specified type (HTTPRedirectBinding or HTTPPostBinding)
func (sp *ServiceProvider) GetSSOBindingLocation(binding string) string {
	for _, singleSignOnService := range sp.IDPMetadata.IDPSSODescriptor.SingleSignOnService {
		if singleSignOnService.Binding == binding {
			return singleSignOnService.Location
		}
	}
	return ""
}

// getIDPSigningCert returns the certificate which we can use to verify things
// signed by the IDP in PEM format, or nil if no such certificate is found.
func (sp *ServiceProvider) getIDPSigningCert() []byte {
	cert := ""

	for _, keyDescriptor := range sp.IDPMetadata.IDPSSODescriptor.KeyDescriptor {
		if keyDescriptor.Use == "signing" {
			cert = keyDescriptor.KeyInfo.Certificate
			break
		}
	}

	// If there are no explicitly signing certs, just return the first
	// non-empty cert we find.
	if cert == "" {
		for _, keyDescriptor := range sp.IDPMetadata.IDPSSODescriptor.KeyDescriptor {
			if keyDescriptor.Use == "" && keyDescriptor.KeyInfo.Certificate != "" {
				cert = keyDescriptor.KeyInfo.Certificate
				break
			}
		}
	}

	if cert == "" {
		return nil
	}

	// cleanup whitespace and re-encode a PEM
	cert = regexp.MustCompile("\\s+").ReplaceAllString(cert, "")
	certBytes, _ := base64.StdEncoding.DecodeString(cert)
	certBytes = pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certBytes})
	return certBytes
}

// MakeAuthenticationRequest produces a new AuthnRequest object for idpURL.
func (sp *ServiceProvider) MakeAuthenticationRequest(idpURL string) (*AuthnRequest, error) {
	req := AuthnRequest{
		AssertionConsumerServiceURL: sp.AcsURL,
		Destination:                 idpURL,
		ProtocolBinding:             HTTPPostBinding, // default binding for the response
		ID:                          fmt.Sprintf("id-%x", randomBytes(20)),
		IssueInstant:                TimeNow(),
		Version:                     "2.0",
		Issuer: Issuer{
			Format: "urn:oasis:names:tc:SAML:2.0:nameid-format:entity",
			Value:  sp.MetadataURL,
		},
		NameIDPolicy: NameIDPolicy{
			AllowCreate: true,
			// TODO(ross): figure out exactly policy we need
			// urn:mace:shibboleth:1.0:nameIdentifier
			// urn:oasis:names:tc:SAML:2.0:nameid-format:transient
			Format: "urn:oasis:names:tc:SAML:2.0:nameid-format:transient",
		},
	}
	return &req, nil
}

// MakePostAuthenticationRequest creates a SAML authentication request using
// the HTTP-POST binding. It returns HTML text representing an HTML form that
// can be sent presented to a browser to initiate the login process.
func (sp *ServiceProvider) MakePostAuthenticationRequest(relayState string) ([]byte, error) {
	req, err := sp.MakeAuthenticationRequest(sp.GetSSOBindingLocation(HTTPPostBinding))
	if err != nil {
		return nil, err
	}
	return req.Post(relayState), nil
}

// Post returns an HTML form suitable for using the HTTP-POST binding with the request
func (req *AuthnRequest) Post(relayState string) []byte {
	reqBuf, err := xml.Marshal(req)
	if err != nil {
		panic(err)
	}
	encodedReqBuf := base64.StdEncoding.EncodeToString(reqBuf)

	tmpl := template.Must(template.New("saml-post-form").Parse(`` +
		`<form method="post" action="{{.URL}}" id="SAMLRequestForm">` +
		`<input type="hidden" name="SAMLRequest" value="{{.SAMLRequest}}" />` +
		`<input type="hidden" name="RelayState" value="{{.RelayState}}" />` +
		`<input type="submit" value="Submit" />` +
		`</form>` +
		`<script>document.getElementById('SAMLRequestForm').submit();</script>`))
	data := struct {
		URL         string
		SAMLRequest string
		RelayState  string
	}{
		URL:         req.Destination,
		SAMLRequest: encodedReqBuf,
		RelayState:  relayState,
	}

	rv := bytes.Buffer{}
	if err := tmpl.Execute(&rv, data); err != nil {
		panic(err)
	}

	return rv.Bytes()
}

// AssertionAttributes is a list of AssertionAttribute
type AssertionAttributes []AssertionAttribute

// Get returns the assertion attribute whose Name or FriendlyName
// matches name, or nil if no matching attribute is found.
func (aa AssertionAttributes) Get(name string) *AssertionAttribute {
	for _, attr := range aa {
		if attr.Name == name {
			return &attr
		}
		if attr.FriendlyName == name {
			return &attr
		}
	}
	return nil
}

// AssertionAttribute represents an attribute of the user extracted from
// a SAML Assertion.
type AssertionAttribute struct {
	FriendlyName string
	Name         string
	Value        string
}

// InvalidResponseError is the error produced by ParseResponse when it fails.
// The underlying error is in PrivateErr. Response is the response as it was
// known at the time validation failed. Now is the time that was used to validate
// time-dependent parts of the assertion.
type InvalidResponseError struct {
	PrivateErr error
	Response   string
	Now        time.Time
}

func (ivr *InvalidResponseError) Error() string {
	return fmt.Sprintf("Authentication failed")
}

func (sp *ServiceProvider) ParseResponse(req *http.Request, possibleRequestIDs []string) (*Assertion, error) {
	samlResponse := req.PostForm.Get("SAMLResponse")
	return sp.ParseSAMLResponse(samlResponse, possibleRequestIDs)
}

// ParseResponse extracts the SAML IDP response received in req, validates
// it, and returns the verified attributes of the request.
//
// This function handles decrypting the message, verifying the digital
// signature on the assertion, and verifying that the specified conditions
// and properties are met.
//
// If the function fails it will return an InvalidResponseError whose
// properties are useful in describing which part of the parsing process
// failed. However, to discourage inadvertent disclosure the diagnostic
// information, the Error() method returns a static string.
func (sp *ServiceProvider) ParseSAMLResponse(samlResponse string, possibleRequestIDs []string) (*Assertion, error) {
	now := TimeNow()
	retErr := &InvalidResponseError{
		Now:      now,
		Response: samlResponse,
	}

	rawResponseBuf, err := base64.StdEncoding.DecodeString(samlResponse)
	if err != nil {
		retErr.PrivateErr = fmt.Errorf("cannot parse base64: %s", err)
		return nil, retErr
	}
	retErr.Response = string(rawResponseBuf)

	// do some validation first before we decrypt
	resp := Response{}
	if err := xml.Unmarshal(rawResponseBuf, &resp); err != nil {
		retErr.PrivateErr = fmt.Errorf("cannot unmarshal response: %s", err)
		return nil, retErr
	}
	if resp.Destination != sp.AcsURL {
		retErr.PrivateErr = fmt.Errorf("`Destination` does not match AcsURL (expected %q)", sp.AcsURL)
		return nil, retErr
	}

	requestIDvalid := false
	for _, possibleRequestID := range possibleRequestIDs {
		if resp.InResponseTo == possibleRequestID {
			requestIDvalid = true
		}
	}
	if !requestIDvalid {
		retErr.PrivateErr = fmt.Errorf("`InResponseTo` does not match any of the possible request IDs (expected %v)", possibleRequestIDs)
		return nil, retErr
	}

	if resp.IssueInstant.Add(MaxIssueDelay).Before(now) {
		retErr.PrivateErr = fmt.Errorf("IssueInstant expired at %s", resp.IssueInstant.Add(MaxIssueDelay))
		return nil, retErr
	}
	if resp.Issuer.Value != sp.IDPMetadata.EntityID {
		retErr.PrivateErr = fmt.Errorf("Issuer does not match the IDP metadata (expected %q)", sp.IDPMetadata.EntityID)
		return nil, retErr
	}
	if resp.Status.StatusCode.Value != StatusSuccess {
		retErr.PrivateErr = fmt.Errorf("Status code was not %s", StatusSuccess)
		return nil, retErr
	}

	var assertion *Assertion
	if resp.EncryptedAssertion == nil {
		if err := xmlsec.Verify(sp.getIDPSigningCert(), rawResponseBuf,
			xmlsec.SignatureOptions{
				XMLID: []xmlsec.XMLIDOption{{
					ElementName:      "Response",
					ElementNamespace: "urn:oasis:names:tc:SAML:2.0:protocol",
					AttributeName:    "ID",
				}},
			}); err != nil {
			retErr.PrivateErr = fmt.Errorf("failed to verify signature on response: %s", err)
			return nil, retErr
		}
		assertion = resp.Assertion
	}

	// decrypt the response
	if resp.EncryptedAssertion != nil {
		plaintextAssertion, err := xmlsec.Decrypt([]byte(sp.Key), resp.EncryptedAssertion.EncryptedData)
		if err != nil {
			retErr.PrivateErr = fmt.Errorf("failed to decrypt response: %s", err)
			return nil, retErr
		}
		retErr.Response = string(plaintextAssertion)

		if err := xmlsec.Verify(sp.getIDPSigningCert(), plaintextAssertion,
			xmlsec.SignatureOptions{
				XMLID: []xmlsec.XMLIDOption{{
					ElementName:      "Assertion",
					ElementNamespace: "urn:oasis:names:tc:SAML:2.0:assertion",
					AttributeName:    "ID",
				}},
			}); err != nil {
			retErr.PrivateErr = fmt.Errorf("failed to verify signature on response: %s", err)
			return nil, retErr
		}

		assertion = &Assertion{}
		xml.Unmarshal(plaintextAssertion, assertion)
	}

	if err := sp.validateAssertion(assertion, possibleRequestIDs, now); err != nil {
		retErr.PrivateErr = fmt.Errorf("assertion invalid: %s", err)
		return nil, retErr
	}

	return assertion, nil
}

// validateAssertion checks that the conditions specified in assertion match
// the requirements to accept. If validation fails, it returns an error describing
// the failure. (The digital signature on the assertion is not checked -- this
// should be done before calling this function).
func (sp *ServiceProvider) validateAssertion(assertion *Assertion, possibleRequestIDs []string, now time.Time) error {
	if assertion.IssueInstant.Add(MaxIssueDelay).Before(now) {
		return fmt.Errorf("expired on %s", assertion.IssueInstant.Add(MaxIssueDelay))
	}
	if assertion.Issuer.Value != sp.IDPMetadata.EntityID {
		return fmt.Errorf("issuer is not %q", sp.IDPMetadata.EntityID)
	}
	requestIDvalid := false
	for _, possibleRequestID := range possibleRequestIDs {
		if assertion.Subject.SubjectConfirmation.SubjectConfirmationData.InResponseTo == possibleRequestID {
			requestIDvalid = true
			break
		}
	}
	if !requestIDvalid {
		return fmt.Errorf("SubjectConfirmation one of the possible request IDs (%v)", possibleRequestIDs)
	}
	if assertion.Subject.SubjectConfirmation.SubjectConfirmationData.Recipient != sp.AcsURL {
		return fmt.Errorf("SubjectConfirmation Recipient is not %s", sp.AcsURL)
	}
	if assertion.Subject.SubjectConfirmation.SubjectConfirmationData.NotOnOrAfter.Before(now) {
		return fmt.Errorf("SubjectConfirmationData is expired")
	}
	if assertion.Conditions.NotBefore.After(now) {
		return fmt.Errorf("Conditions is not yet valid")
	}
	if assertion.Conditions.NotOnOrAfter.Before(now) {
		return fmt.Errorf("Conditions is expired")
	}
	if assertion.Conditions.AudienceRestriction.Audience.Value != sp.MetadataURL {
		return fmt.Errorf("Conditions AudienceRestriction is not %q", sp.MetadataURL)
	}
	return nil
}

// Login performs the necessary actions to start an SP initiated login.
func (sp *ServiceProvider) InitiateLogin(w http.ResponseWriter) error {
	acsURL, _ := url.Parse(sp.AcsURL)

	binding := HTTPRedirectBinding
	bindingLocation := sp.GetSSOBindingLocation(binding)
	if bindingLocation == "" {
		binding = HTTPPostBinding
		bindingLocation = sp.GetSSOBindingLocation(binding)
	}

	req, err := sp.MakeAuthenticationRequest(bindingLocation)
	if err != nil {
		return err
	}

	relayState := base64.URLEncoding.EncodeToString(randomBytes(42))
	state := jwt.New(jwt.GetSigningMethod("HS256"))
	claims := state.Claims.(jwt.MapClaims)
	claims["id"] = req.ID
	signedState, err := state.SignedString(sp.cookieSecret())
	if err != nil {
		return err
	}

	http.SetCookie(w, &http.Cookie{
		Name:     fmt.Sprintf("saml_%s", relayState),
		Value:    signedState,
		MaxAge:   int(MaxIssueDelay.Seconds()),
		HttpOnly: false,
		Path:     acsURL.Path,
	})

	if binding == HTTPRedirectBinding {
		redirectURL := req.Redirect(relayState)
		w.Header().Add("Location", redirectURL.String())
		w.WriteHeader(http.StatusFound)
		return nil
	}
	if binding == HTTPPostBinding {
		w.Header().Set("Content-Security-Policy", ""+
			"default-src; "+
			"script-src 'sha256-D8xB+y+rJ90RmLdP72xBqEEc0NUatn7yuCND0orkrgk='; "+
			"reflected-xss block; "+
			"referrer no-referrer;")
		w.Header().Add("Content-type", "text/html")
		w.Write([]byte(`<!DOCTYPE html><html><body>`))
		w.Write(req.Post(relayState))
		w.Write([]byte(`</body></html>`))
		return nil
	}
	panic("not reached")
}

// Parse parses the SAMLResponse
func (sp *ServiceProvider) Parse(w http.ResponseWriter, r *http.Request) (*Assertion, error) {
	allowIdPInitiated := ""
	possibleRequestIDs := []string{allowIdPInitiated}

	// Find the request id that relates to this RelayState.
	if relayState := r.PostFormValue("RelayState"); relayState != "" {
		cookieName := fmt.Sprintf("saml_%s", relayState)
		cookie, err := r.Cookie(cookieName)
		if err != nil {
			return nil, fmt.Errorf("cannot find %s cookie", cookieName)
		}

		// Verify the integrity of the cookie.
		state, err := jwt.Parse(cookie.Value, func(t *jwt.Token) (interface{}, error) {
			return sp.cookieSecret(), nil
		})
		if err != nil || !state.Valid {
			return nil, fmt.Errorf("could not decode state JWT: %v", err)
		}

		claims := state.Claims.(jwt.MapClaims)
		id := claims["id"].(string)

		possibleRequestIDs = append(possibleRequestIDs, id)

		// delete the cookie
		cookie.Value = ""
		cookie.Expires = time.Time{}
		http.SetCookie(w, cookie)
	}

	samlResponse := r.PostFormValue("SAMLResponse")
	return sp.ParseSAMLResponse(samlResponse, possibleRequestIDs)
}

func (sp *ServiceProvider) cookieSecret() []byte {
	secretBlock, _ := pem.Decode([]byte(sp.Key))
	return secretBlock.Bytes
}
