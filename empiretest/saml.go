package empiretest

import (
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/remind101/empire/pkg/saml"
)

// IdentityProvider wraps a saml.IdentityProvider to serve it via an
// httptest.Server.
type IdentityProvider struct {
	*saml.IdentityProvider
	svr *httptest.Server
}

// NewIdentityProvider returns a new saml.IdentityProvider that can be used for testing
// the SAML integration.
func NewIdentityProvider() *IdentityProvider {
	idp := &saml.IdentityProvider{
		Key:              idpKey,
		Certificate:      idpCert,
		SessionProvider:  new(sessionProvider),
		ServiceProviders: map[string]*saml.Metadata{},
	}

	s := httptest.NewServer(idp)
	idp.MetadataURL = fmt.Sprintf("%s/metadata", s.URL)
	idp.SSOURL = fmt.Sprintf("%s/sso", s.URL)

	return &IdentityProvider{
		IdentityProvider: idp,
		svr:              s,
	}
}

// Close closes the underlying httptest.Server serving this IdP.
func (idp *IdentityProvider) Close() {
	idp.svr.Close()
}

// AddServiceProvider adds a service provider to this IDP.
func (idp *IdentityProvider) AddServiceProvider(url string) *saml.ServiceProvider {
	metadataURL := fmt.Sprintf("%s/saml/metadata", url)
	acsURL := fmt.Sprintf("%s/saml/acs", url)

	encryptionCert, _ := pem.Decode([]byte(spCert))
	signingCert, _ := pem.Decode([]byte(idpCert))

	metadata := &saml.Metadata{
		EntityID: metadataURL,
		SPSSODescriptor: &saml.SPSSODescriptor{
			AssertionConsumerService: []saml.IndexedEndpoint{
				{
					Binding:  saml.HTTPRedirectBinding,
					Location: acsURL,
					Index:    1,
				},
			},
			KeyDescriptor: []saml.KeyDescriptor{
				{
					Use: "encryption",
					KeyInfo: saml.KeyInfo{
						Certificate: base64.StdEncoding.EncodeToString(encryptionCert.Bytes),
					},
					EncryptionMethods: []saml.EncryptionMethod{
						{Algorithm: "http://www.w3.org/2001/04/xmlenc#aes128-cbc"},
						{Algorithm: "http://www.w3.org/2001/04/xmlenc#aes192-cbc"},
						{Algorithm: "http://www.w3.org/2001/04/xmlenc#aes256-cbc"},
						{Algorithm: "http://www.w3.org/2001/04/xmlenc#rsa-oaep-mgf1p"},
					},
				},
			},
		},
		IDPSSODescriptor: &saml.IDPSSODescriptor{
			ProtocolSupportEnumeration: "urn:oasis:names:tc:SAML:2.0:protocol",
			KeyDescriptor: []saml.KeyDescriptor{
				{
					Use: "signing",
					KeyInfo: saml.KeyInfo{
						Certificate: base64.StdEncoding.EncodeToString(signingCert.Bytes),
					},
				},
			},
			NameIDFormat: []string{
				"urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress",
			},
			SingleSignOnService: []saml.Endpoint{
				{
					Binding:  saml.HTTPRedirectBinding,
					Location: idp.SSOURL,
				},
				{
					Binding:  saml.HTTPPostBinding,
					Location: idp.SSOURL,
				},
			},
		},
	}
	sp := &saml.ServiceProvider{
		IDPMetadata: metadata,
		Key:         spKey,
		Certificate: spCert,
		MetadataURL: metadataURL,
		AcsURL:      acsURL,
	}
	idp.ServiceProviders[metadataURL] = metadata
	return sp
}

// sessionProvider is a null implementation of the saml.SessionProvider
// interface.
type sessionProvider struct{}

func (s *sessionProvider) GetSession(w http.ResponseWriter, r *http.Request, req *saml.IdpAuthnRequest) *saml.Session {
	return &saml.Session{
		ID:         "session",
		CreateTime: time.Now(),
		ExpireTime: time.Now().Add(24 * time.Hour),
		Index:      "0",
		NameID:     "dummy",
	}
}

// Dummy RSA key/certs generated with:
//
//	openssl req -x509 -newkey rsa:2048 -keyout sp.key -out sp.cert -days 365 -nodes -subj "/CN=myservice.example.com"
const (
	idpKey = `-----BEGIN RSA PRIVATE KEY-----
MIIEowIBAAKCAQEAzauzW4quwlSrS1DC6zbG7jWNusz1ICJFvHXfytBFsTxMcWdH
ltXbVGpRhMwfryJcNjHOlVS32PNpTeUu0h1Ctg0guNRpZYMWmVpDSepC6U3s5fhi
woZf7/tgRJRHGe9jiDlltR8XqamUZA+CUNUUHxaHktY7qbGhqiWxCgIJhTQJgZkx
s7oGvgjxv8PbL0zoI1hJGN8j8GCjzjhhD4GuNo87/0PjYWVT6+CkNlLpPinLF7vt
K81kk7EpIzGozjcfbJzKsX7tHKZHgxqJ8+nk89LzWifAchtwi5AbZZh7wdUC9xFX
kVbmTyt7v5kjTILxCvLWMn8GnKDUupzx1doQ9QIDAQABAoIBAEFlBMRoliSIE2cB
KNjtM9duA8CPhqlO/Glt+VhdXKt8BrpQIn0dRn3SyFS3KqPfAv7gW1Uz+LjsvGDe
LEL2ts134x2hvFlgkwDzqE2KRPg1pMuCbLR5UWvWu8dSdkES0szvox0x4644k0w1
ejQFbD5uRXE02QedyU92aJJMD4bgcJJQbHSXXCA0NiGk9sXrq+ITEhlHgJhkOCax
aDIXRn+jf5H0gID3t5G3VCLahBEmA7RquGScvRYZj7X41fJKeA4MjuDEPBCwxrca
zJECVF/ZJGUqYuV0KnfVA2n97SXACGrmDiIywsVmyurR/TIsqOXdSPBxqbXnX+LN
n5GbRAkCgYEA9nxzCNMom/0tgK4rCmtxwzbmNtIg/ZuI55vc3jQmXGU6YoXD9UyE
eVje//4ze2LtnMyTzgPiWVoBzMOW1NDpghTeCPxsrDxL0l4WmxxInDOm9f3QeR6A
PDuxRoQqesddnNd458kF7Hk4QjrMLbny0kA05W2qk0xRKmP7EdZNV4sCgYEA1Zvz
VqO712jRJHv5Ri01d8D+RL1mCp4A7uYlLMScZkxbH+zImpj3c7qm9B7IVwjWQH65
l/ottRIko10Ep3p8GQY2c+adx3AIAZ2WuJ4Zd09AdDRUn86RjJG5ydxTv5/1cYmJ
oe2BS088RNNwjKeMRiwJ86WeMmdem1klAM8vSX8CgYB0/YiUDbVepIJuazxei7TJ
VUtbhczG0oXeeGoSxWnXvOxDSv5BdXoDJp1hn8PLsp7ZJ3iX9dv/UOs9xy/V/vp2
FXV1imoCLfRG+wV7xabpDNMYOsoyUrnG3QY9VAndkLbr9JGcYht/q+F5/fJfWbzY
8kSpCK5Hj5eOqTnHs5GuFQKBgF1SfMVlUzOQ/45I+2bFaY6gKnYtqN8KmK3GrocY
fpvS0BzqfdnM6o8NBNOyfyRHIBOdScgz7LQm8QrOILJquLzWEgQgxN3U/Cp4htix
eb6+SRJ7ql0HCl+3asveDlixsbGgvRiZgts8CsCm/4zzxj0CEHb57Fto/dQw5hGs
cqRRAoGBAL8UtWhtGB/A7LvD3AjHZaJyPWhWKsZRvoAxIXDOzB+ZUtCQ486e9p0s
nEZOAAdx1wAFJomdvUMs92C3LrW4VX/SVQ6R+HiTLlDBH8R0GPsq7DabKlwG/vxw
9vpQe0RLm/k0ggFodYr91KWazddysR/20HOQxOOkT7UyCa9f/dD5
-----END RSA PRIVATE KEY-----`
	idpCert = `-----BEGIN CERTIFICATE-----
MIIDRTCCAi2gAwIBAgIJAP+pPLMtFjY8MA0GCSqGSIb3DQEBBQUAMCAxHjAcBgNV
BAMTFW15c2VydmljZS5leGFtcGxlLmNvbTAeFw0xNjEwMjYwMTA2MzFaFw0xNzEw
MjYwMTA2MzFaMCAxHjAcBgNVBAMTFW15c2VydmljZS5leGFtcGxlLmNvbTCCASIw
DQYJKoZIhvcNAQEBBQADggEPADCCAQoCggEBAM2rs1uKrsJUq0tQwus2xu41jbrM
9SAiRbx138rQRbE8THFnR5bV21RqUYTMH68iXDYxzpVUt9jzaU3lLtIdQrYNILjU
aWWDFplaQ0nqQulN7OX4YsKGX+/7YESURxnvY4g5ZbUfF6mplGQPglDVFB8Wh5LW
O6mxoaolsQoCCYU0CYGZMbO6Br4I8b/D2y9M6CNYSRjfI/Bgo844YQ+BrjaPO/9D
42FlU+vgpDZS6T4pyxe77SvNZJOxKSMxqM43H2ycyrF+7RymR4MaifPp5PPS81on
wHIbcIuQG2WYe8HVAvcRV5FW5k8re7+ZI0yC8Qry1jJ/Bpyg1Lqc8dXaEPUCAwEA
AaOBgTB/MB0GA1UdDgQWBBQB+BvlI974Se0xDrggPlKAgi8K4TBQBgNVHSMESTBH
gBQB+BvlI974Se0xDrggPlKAgi8K4aEkpCIwIDEeMBwGA1UEAxMVbXlzZXJ2aWNl
LmV4YW1wbGUuY29tggkA/6k8sy0WNjwwDAYDVR0TBAUwAwEB/zANBgkqhkiG9w0B
AQUFAAOCAQEApUqtb/UzNjevQpUl7mfWGolD7tQuRoGFm6Fb0rdsi9/f1fjcrzHC
E96tkvqxVteLaygeJqde2r/jhMrag930Vc+57Y80Ln2S4atfhu5Ee49JBGxKNdb0
gUGlsKKRThyP6LsW3GpSb0o5cEXPwPcWPw5nD+NJsXT6/cSAJliCA7Fju/TgKVS3
5xI/Kah2s9cFYVx4wDqtIT9hO5b2n8zQ44+iXxzwXKHFqMWqp+kH3X7X+YqELpTl
igrEuZixs/cGukVUhBgDtp5gOGaNEUTjZfxlnUrqc4fojxGTa6OQmhd8M9co577G
2TMVWn4vovJMG4neATs8IFGk0HFq0qu50g==
-----END CERTIFICATE-----`

	spKey = `-----BEGIN RSA PRIVATE KEY-----
MIIEowIBAAKCAQEA798XSRz57iU4YlMfxBe14SOJXa1WY59NzXfiw99NYuD7LGbA
fNVAgI+/Av56uVq1AjHmduDpyeoRrCHqB80pfIHTsa3lxcEaqUZ4Wr8h65iPbn9U
xri1IQsotEH+lq2BmCzOUgDe1WTNEn3sP9IndXL/3lkw1RDvZZ+13O0q6z+9Rd+y
wkEUaVUgn5VANVwHgzB+tTNE/bqBUKPpu0KC/EmAhDppk+u+6SQfMpeUbBUjk4VC
nSwkJAbMZK/SVCxFEG9PROUTcKqVZJS1v2tJOgTh0gxCxf9aFkFNZJ0kkMA8EsVL
XL4x9tqfLSuH+4W7PtiFZsEdoP70MCcG6AsitwIDAQABAoIBAFcjtFHbNPBOlS0j
BWc3NduUDVL6pWeLd7gs1TRS6soA8T4wFb1Duyr5DWsJB4xOZ3NkrVSCYGv5nHHr
4Bj1bxzMDRt/EPnGLOlRsGdHCAuOYIzDtQh8EVDvvNm/H72dSnb9z/X6WvkqpVUS
acDFl5ATNuCPhi3g+Rbx7h9UCUNsMEWWO9qj6VXl3Qm0hijqwL2Q53D7CHKpoSby
8F9DthPXaJt2GI/3T66eaDph4OR7vmyk/uTjvxPbsXQOMQ1eSQjLddLN4/McJhXn
yOir4wjsvo/6PVeRw+SiPjSJf5gVsXhEVqTqiC1oXtqgpogd+nH3SKylHWRuRDrS
aYwreAECgYEA+3fUET+UGAMdHEs75sxxNtow08RBRKO6Y+OrPERP5jPxjZ1t/Pmq
XkmALJUKmfcwNEIval1vcFjrfESjxv7l4kAFXDA+qP+IWY1vpo+ZAXTqZMxxgnXx
juoCbDxrDkAx2Lh8E/XBy3vBKleXY/3ibPWVgqEou9qdJ5Tqzh1GqlMCgYEA9DHC
q064pGE8+X7TrFyxP7XiXdh9FTGMSfqE0F2dDLONqhi0d4xSPc6g8wl8Kz+Y3E2D
e+143+06OT8Ew3eExM1w2JrNvbMyep0wbUoXqOwo3AZevK4rxyWkWrfw1qRIx6z5
6pJUAF+9HAch5D2W5CLgqOxUJRz3Yi+jDnwXAY0CgYAoAUYxgEXVFBm7eJSNARU5
vrhp2BzyCIIMhhmlutBjNPxGpTbsOePKoDLN5OAM4nA+wBC/ASJLYzoDSQAtFjwI
JFs18U7mn9BXPtL2Un3q52iqpIOiV5UYQU4lXe9CEyBa8+55Vm2AK63tSIYDGE6/
OsqQP4c1a2/47g30wF+PlQKBgQCiibSPnfhcwbR6RSbTpWb9hy1DVeP8BVzhqPRa
VNVCLQlwXL1SjX34Ud7jpj6V8uDmUlngVTKNqjOFAyNCj/05mZ0xL+keCXbiElq3
hAe3kmmn+j14zV2qUq3RDHosBHHFJqe6sOdk0FTpoP24FB6pf2WWSqe/hEZNfnPE
ImiVyQKBgDcsDve0XBkb2k9IEGx34n1pCzu0fJSkuSyp29Uo0JcqEPklmv6efr0n
DyZzLNqBGWosISfEzFNE5S8+cNC1lX/G15LtaH9x9Kz3jDGLMAHH2CcKa4EDXvm4
Alc8WiRf1yrL679aNr0xKK5H+HSbeLe6L4hIOjtgkjkIi40uUWEq
-----END RSA PRIVATE KEY-----`
	spCert = `-----BEGIN CERTIFICATE-----
MIIDRTCCAi2gAwIBAgIJAJzaVIQLMMfUMA0GCSqGSIb3DQEBBQUAMCAxHjAcBgNV
BAMTFW15c2VydmljZS5leGFtcGxlLmNvbTAeFw0xNjEwMjYwMTA1MTlaFw0xNzEw
MjYwMTA1MTlaMCAxHjAcBgNVBAMTFW15c2VydmljZS5leGFtcGxlLmNvbTCCASIw
DQYJKoZIhvcNAQEBBQADggEPADCCAQoCggEBAO/fF0kc+e4lOGJTH8QXteEjiV2t
VmOfTc134sPfTWLg+yxmwHzVQICPvwL+erlatQIx5nbg6cnqEawh6gfNKXyB07Gt
5cXBGqlGeFq/IeuYj25/VMa4tSELKLRB/patgZgszlIA3tVkzRJ97D/SJ3Vy/95Z
MNUQ72WftdztKus/vUXfssJBFGlVIJ+VQDVcB4MwfrUzRP26gVCj6btCgvxJgIQ6
aZPrvukkHzKXlGwVI5OFQp0sJCQGzGSv0lQsRRBvT0TlE3CqlWSUtb9rSToE4dIM
QsX/WhZBTWSdJJDAPBLFS1y+Mfbany0rh/uFuz7YhWbBHaD+9DAnBugLIrcCAwEA
AaOBgTB/MB0GA1UdDgQWBBRbuy2roSBuLgSuB7y+h2ZRnhp4NzBQBgNVHSMESTBH
gBRbuy2roSBuLgSuB7y+h2ZRnhp4N6EkpCIwIDEeMBwGA1UEAxMVbXlzZXJ2aWNl
LmV4YW1wbGUuY29tggkAnNpUhAswx9QwDAYDVR0TBAUwAwEB/zANBgkqhkiG9w0B
AQUFAAOCAQEAVt9EboqvVNeVU+NTaQgXSDoNFWURbzKDzBpOTEaitHjRzhDAvDkA
QYgfWk2xU/PtkkujEsfM96xJiAHfgTLc7MEsNdGRHB+QKBnW+PcrZ9L2HofzeRF+
K8yzGF2XDgXGBloekeg3UKIH7j5mXHKGxD0dgOH0mwlrNBwkGlYWFhw1+r3Ttngc
bVxBCQ591HG7PKZR/rfCbWNSfUl7xIVWICzHiPihDFBOVY7RXnXQhAZ7Duq2gPn/
EqGeg3l0ZavBef+X3tR0HaMARfHAAB8lx0KAwp9YJGCEp31XrkwP9H52S4195kzN
C3wXZAUYlPkbNc6KUBGHMlIj+umLeS3OXw==
-----END CERTIFICATE-----`
)
