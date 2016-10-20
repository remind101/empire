package xmlsec

import (
	"encoding/xml"
	"strings"

	. "gopkg.in/check.v1"
)

type Envelope struct {
	Data      string
	Signature Signature `xml:"http://www.w3.org/2000/09/xmldsig# Signature"`
}

type XMLDSigTest struct {
	Key    []byte
	Cert   []byte
	DocStr []byte
}

var _ = Suite(&XMLDSigTest{})

func (testSuite *XMLDSigTest) SetUpTest(c *C) {
	testSuite.Key = []byte(`-----BEGIN RSA PRIVATE KEY-----
MIIBPAIBAAJBANPQbQ92nlbeg1Q5JNHSO1Yey46nZ7GJltLWw1ccSvp7pnvmfUm+
M521CpFpfr4EAE3UVBMoU9j/hqq3dFAc2H0CAwEAAQJBALFVCjmsAZyQ5jqZLO5N
qEfNuHZSSUol+xPBogFIOq3BWa269eNNcAK5or5g0XWWon7EPdyGT4qyDVH9KzXK
RLECIQDzm/Nj0epUGN51/rKJgRXWkXW/nfSCMO9fvQR6Ujoq3wIhAN6WeHK9vgWg
wBWqMdq5sR211+LlDH7rOUQ6rBpbsoQjAiEA7jzpfglgPPZFOOfo+oh/LuP6X3a+
FER/FQXpRyb7M8kCIETUrwZ8WkiPPxbz/Fqw1W5kjw/g2I5e2uSYaCP2eyuVAiEA
mOI6RhRyMqgxQyy0plJVjG1s4fdu92AWYy9AwYeyd/8=
-----END RSA PRIVATE KEY-----
`)
	testSuite.Cert = []byte(`-----BEGIN CERTIFICATE-----
MIIDpzCCA1GgAwIBAgIJAK+ii7kzrdqvMA0GCSqGSIb3DQEBBQUAMIGcMQswCQYD
VQQGEwJVUzETMBEGA1UECBMKQ2FsaWZvcm5pYTE9MDsGA1UEChM0WE1MIFNlY3Vy
aXR5IExpYnJhcnkgKGh0dHA6Ly93d3cuYWxla3NleS5jb20veG1sc2VjKTEWMBQG
A1UEAxMNQWxla3NleSBTYW5pbjEhMB8GCSqGSIb3DQEJARYSeG1sc2VjQGFsZWtz
ZXkuY29tMCAXDTE0MDUyMzE3NTUzNFoYDzIxMTQwNDI5MTc1NTM0WjCBxzELMAkG
A1UEBhMCVVMxEzARBgNVBAgTCkNhbGlmb3JuaWExPTA7BgNVBAoTNFhNTCBTZWN1
cml0eSBMaWJyYXJ5IChodHRwOi8vd3d3LmFsZWtzZXkuY29tL3htbHNlYykxKTAn
BgNVBAsTIFRlc3QgVGhpcmQgTGV2ZWwgUlNBIENlcnRpZmljYXRlMRYwFAYDVQQD
Ew1BbGVrc2V5IFNhbmluMSEwHwYJKoZIhvcNAQkBFhJ4bWxzZWNAYWxla3NleS5j
b20wXDANBgkqhkiG9w0BAQEFAANLADBIAkEA09BtD3aeVt6DVDkk0dI7Vh7Ljqdn
sYmW0tbDVxxK+nume+Z9Sb4znbUKkWl+vgQATdRUEyhT2P+Gqrd0UBzYfQIDAQAB
o4IBRTCCAUEwDAYDVR0TBAUwAwEB/zAsBglghkgBhvhCAQ0EHxYdT3BlblNTTCBH
ZW5lcmF0ZWQgQ2VydGlmaWNhdGUwHQYDVR0OBBYEFNf0xkZ3zjcEI60pVPuwDqTM
QygZMIHjBgNVHSMEgdswgdiAFP7k7FMk8JWVxxC14US1XTllWuN+oYG0pIGxMIGu
MQswCQYDVQQGEwJVUzETMBEGA1UECBMKQ2FsaWZvcm5pYTE9MDsGA1UEChM0WE1M
IFNlY3VyaXR5IExpYnJhcnkgKGh0dHA6Ly93d3cuYWxla3NleS5jb20veG1sc2Vj
KTEQMA4GA1UECxMHUm9vdCBDQTEWMBQGA1UEAxMNQWxla3NleSBTYW5pbjEhMB8G
CSqGSIb3DQEJARYSeG1sc2VjQGFsZWtzZXkuY29tggkAr6KLuTOt2q0wDQYJKoZI
hvcNAQEFBQADQQAOXBj0yICp1RmHXqnUlsppryLCW3pKBD1dkb4HWarO7RjA1yJJ
fBjXssrERn05kpBcrRfzou4r3DCgQFPhjxga
-----END CERTIFICATE-----
`)
	testSuite.DocStr = []byte(`<?xml version="1.0" encoding="UTF-8"?>
<!--
XML Security Library example: Simple signature template file for sign1 example.
-->
<Envelope xmlns="urn:envelope">
  <Data>
	Hello, World!
  </Data>
  <Signature xmlns="http://www.w3.org/2000/09/xmldsig#">
    <SignedInfo>
      <CanonicalizationMethod Algorithm="http://www.w3.org/TR/2001/REC-xml-c14n-20010315"/>
      <SignatureMethod Algorithm="http://www.w3.org/2000/09/xmldsig#rsa-sha1"/>
      <Reference URI="">
        <Transforms>
          <Transform Algorithm="http://www.w3.org/2000/09/xmldsig#enveloped-signature"/>
        </Transforms>
        <DigestMethod Algorithm="http://www.w3.org/2000/09/xmldsig#sha1"/>
        <DigestValue>9H/rQr2Axe9hYTV2n/tCp+3UIQQ=</DigestValue>
      </Reference>
    </SignedInfo>
    <SignatureValue></SignatureValue>
    <KeyInfo>
		<X509Data>
			<X509Certificate>MIIDpzCCA1GgAwIBAgIJAK+ii7kzrdqvMA0GCSqGSIb3DQEBBQUAMIGcMQswCQYD
VQQGEwJVUzETMBEGA1UECBMKQ2FsaWZvcm5pYTE9MDsGA1UEChM0WE1MIFNlY3Vy
aXR5IExpYnJhcnkgKGh0dHA6Ly93d3cuYWxla3NleS5jb20veG1sc2VjKTEWMBQG
A1UEAxMNQWxla3NleSBTYW5pbjEhMB8GCSqGSIb3DQEJARYSeG1sc2VjQGFsZWtz
ZXkuY29tMCAXDTE0MDUyMzE3NTUzNFoYDzIxMTQwNDI5MTc1NTM0WjCBxzELMAkG
A1UEBhMCVVMxEzARBgNVBAgTCkNhbGlmb3JuaWExPTA7BgNVBAoTNFhNTCBTZWN1
cml0eSBMaWJyYXJ5IChodHRwOi8vd3d3LmFsZWtzZXkuY29tL3htbHNlYykxKTAn
BgNVBAsTIFRlc3QgVGhpcmQgTGV2ZWwgUlNBIENlcnRpZmljYXRlMRYwFAYDVQQD
Ew1BbGVrc2V5IFNhbmluMSEwHwYJKoZIhvcNAQkBFhJ4bWxzZWNAYWxla3NleS5j
b20wXDANBgkqhkiG9w0BAQEFAANLADBIAkEA09BtD3aeVt6DVDkk0dI7Vh7Ljqdn
sYmW0tbDVxxK+nume+Z9Sb4znbUKkWl+vgQATdRUEyhT2P+Gqrd0UBzYfQIDAQAB
o4IBRTCCAUEwDAYDVR0TBAUwAwEB/zAsBglghkgBhvhCAQ0EHxYdT3BlblNTTCBH
ZW5lcmF0ZWQgQ2VydGlmaWNhdGUwHQYDVR0OBBYEFNf0xkZ3zjcEI60pVPuwDqTM
QygZMIHjBgNVHSMEgdswgdiAFP7k7FMk8JWVxxC14US1XTllWuN+oYG0pIGxMIGu
MQswCQYDVQQGEwJVUzETMBEGA1UECBMKQ2FsaWZvcm5pYTE9MDsGA1UEChM0WE1M
IFNlY3VyaXR5IExpYnJhcnkgKGh0dHA6Ly93d3cuYWxla3NleS5jb20veG1sc2Vj
KTEQMA4GA1UECxMHUm9vdCBDQTEWMBQGA1UEAxMNQWxla3NleSBTYW5pbjEhMB8G
CSqGSIb3DQEJARYSeG1sc2VjQGFsZWtzZXkuY29tggkAr6KLuTOt2q0wDQYJKoZI
hvcNAQEFBQADQQAOXBj0yICp1RmHXqnUlsppryLCW3pKBD1dkb4HWarO7RjA1yJJ
fBjXssrERn05kpBcrRfzou4r3DCgQFPhjxga</X509Certificate>
		</X509Data>
	</KeyInfo>
  </Signature>
</Envelope>
`)

}

func (testSuite *XMLDSigTest) TestSignAndVerify(c *C) {
	expectedSignedString := `<?xml version="1.0" encoding="UTF-8"?>
<!--
XML Security Library example: Simple signature template file for sign1 example.
-->
<Envelope xmlns="urn:envelope">
  <Data>
	Hello, World!
  </Data>
  <Signature xmlns="http://www.w3.org/2000/09/xmldsig#">
    <SignedInfo>
      <CanonicalizationMethod Algorithm="http://www.w3.org/TR/2001/REC-xml-c14n-20010315"/>
      <SignatureMethod Algorithm="http://www.w3.org/2000/09/xmldsig#rsa-sha1"/>
      <Reference URI="">
        <Transforms>
          <Transform Algorithm="http://www.w3.org/2000/09/xmldsig#enveloped-signature"/>
        </Transforms>
        <DigestMethod Algorithm="http://www.w3.org/2000/09/xmldsig#sha1"/>
        <DigestValue>9H/rQr2Axe9hYTV2n/tCp+3UIQQ=</DigestValue>
      </Reference>
    </SignedInfo>
    <SignatureValue>fDKK0so/zFcmmq2X+BaVFmS0t8KB7tyW53YN6n221OArzGCs4OyWsAjj/BUR+wNF
elOnt4fo2gPK1a3IVEhMGg==</SignatureValue>
    <KeyInfo>
		<X509Data>
			<X509Certificate>MIIDpzCCA1GgAwIBAgIJAK+ii7kzrdqvMA0GCSqGSIb3DQEBBQUAMIGcMQswCQYD
VQQGEwJVUzETMBEGA1UECBMKQ2FsaWZvcm5pYTE9MDsGA1UEChM0WE1MIFNlY3Vy
aXR5IExpYnJhcnkgKGh0dHA6Ly93d3cuYWxla3NleS5jb20veG1sc2VjKTEWMBQG
A1UEAxMNQWxla3NleSBTYW5pbjEhMB8GCSqGSIb3DQEJARYSeG1sc2VjQGFsZWtz
ZXkuY29tMCAXDTE0MDUyMzE3NTUzNFoYDzIxMTQwNDI5MTc1NTM0WjCBxzELMAkG
A1UEBhMCVVMxEzARBgNVBAgTCkNhbGlmb3JuaWExPTA7BgNVBAoTNFhNTCBTZWN1
cml0eSBMaWJyYXJ5IChodHRwOi8vd3d3LmFsZWtzZXkuY29tL3htbHNlYykxKTAn
BgNVBAsTIFRlc3QgVGhpcmQgTGV2ZWwgUlNBIENlcnRpZmljYXRlMRYwFAYDVQQD
Ew1BbGVrc2V5IFNhbmluMSEwHwYJKoZIhvcNAQkBFhJ4bWxzZWNAYWxla3NleS5j
b20wXDANBgkqhkiG9w0BAQEFAANLADBIAkEA09BtD3aeVt6DVDkk0dI7Vh7Ljqdn
sYmW0tbDVxxK+nume+Z9Sb4znbUKkWl+vgQATdRUEyhT2P+Gqrd0UBzYfQIDAQAB
o4IBRTCCAUEwDAYDVR0TBAUwAwEB/zAsBglghkgBhvhCAQ0EHxYdT3BlblNTTCBH
ZW5lcmF0ZWQgQ2VydGlmaWNhdGUwHQYDVR0OBBYEFNf0xkZ3zjcEI60pVPuwDqTM
QygZMIHjBgNVHSMEgdswgdiAFP7k7FMk8JWVxxC14US1XTllWuN+oYG0pIGxMIGu
MQswCQYDVQQGEwJVUzETMBEGA1UECBMKQ2FsaWZvcm5pYTE9MDsGA1UEChM0WE1M
IFNlY3VyaXR5IExpYnJhcnkgKGh0dHA6Ly93d3cuYWxla3NleS5jb20veG1sc2Vj
KTEQMA4GA1UECxMHUm9vdCBDQTEWMBQGA1UEAxMNQWxla3NleSBTYW5pbjEhMB8G
CSqGSIb3DQEJARYSeG1sc2VjQGFsZWtzZXkuY29tggkAr6KLuTOt2q0wDQYJKoZI
hvcNAQEFBQADQQAOXBj0yICp1RmHXqnUlsppryLCW3pKBD1dkb4HWarO7RjA1yJJ
fBjXssrERn05kpBcrRfzou4r3DCgQFPhjxga</X509Certificate>
		</X509Data>
	</KeyInfo>
  </Signature>
</Envelope>
`
	actualSignedString, err := Sign(testSuite.Key, testSuite.DocStr, SignatureOptions{})
	c.Assert(err, IsNil)
	c.Assert(string(actualSignedString), Equals, expectedSignedString)

	err = Verify(testSuite.Cert, actualSignedString, SignatureOptions{})
	c.Assert(err, IsNil)
}

func (testSuite *XMLDSigTest) TestConstructFromSignature(c *C) {
	// Try again but this time construct the message from a struct having a Signature member
	doc := Envelope{Data: "Hello, World!"}
	doc.Signature = DefaultSignature(testSuite.Cert)
	docStr, err := xml.MarshalIndent(doc, "", "  ")
	c.Assert(err, IsNil)
	actualSignedString, err := Sign(testSuite.Key, docStr, SignatureOptions{})
	c.Assert(err, IsNil)

	expectedSignedString := `<?xml version="1.0"?>
<Envelope>
  <Data>Hello, World!</Data>
  <Signature xmlns="http://www.w3.org/2000/09/xmldsig#">
    <SignedInfo>
      <CanonicalizationMethod Algorithm="http://www.w3.org/TR/2001/REC-xml-c14n-20010315"/>
      <SignatureMethod Algorithm="http://www.w3.org/2000/09/xmldsig#rsa-sha1"/>
      <Reference>
        <Transforms>
          <Transform Algorithm="http://www.w3.org/2000/09/xmldsig#enveloped-signature"/>
        </Transforms>
        <DigestMethod Algorithm="http://www.w3.org/2000/09/xmldsig#sha1"/>
        <DigestValue>sEenIPkW9ssFSB9t4UU6VUrytqc=</DigestValue>
      </Reference>
    </SignedInfo>
    <SignatureValue>chSWfpQBIQraySsUHzs5N51+ruelu2HMHh5Mnd3EjcLqFBVD0f23kmXUp7zVhCVD
vCfqu9yXDYKVOBI57F0Efg==</SignatureValue>
    <KeyInfo>
      <X509Data>
        <X509Certificate>MIIDpzCCA1GgAwIBAgIJAK+ii7kzrdqvMA0GCSqGSIb3DQEBBQUAMIGcMQswCQYDVQQGEwJVUzETMBEGA1UECBMKQ2FsaWZvcm5pYTE9MDsGA1UEChM0WE1MIFNlY3VyaXR5IExpYnJhcnkgKGh0dHA6Ly93d3cuYWxla3NleS5jb20veG1sc2VjKTEWMBQGA1UEAxMNQWxla3NleSBTYW5pbjEhMB8GCSqGSIb3DQEJARYSeG1sc2VjQGFsZWtzZXkuY29tMCAXDTE0MDUyMzE3NTUzNFoYDzIxMTQwNDI5MTc1NTM0WjCBxzELMAkGA1UEBhMCVVMxEzARBgNVBAgTCkNhbGlmb3JuaWExPTA7BgNVBAoTNFhNTCBTZWN1cml0eSBMaWJyYXJ5IChodHRwOi8vd3d3LmFsZWtzZXkuY29tL3htbHNlYykxKTAnBgNVBAsTIFRlc3QgVGhpcmQgTGV2ZWwgUlNBIENlcnRpZmljYXRlMRYwFAYDVQQDEw1BbGVrc2V5IFNhbmluMSEwHwYJKoZIhvcNAQkBFhJ4bWxzZWNAYWxla3NleS5jb20wXDANBgkqhkiG9w0BAQEFAANLADBIAkEA09BtD3aeVt6DVDkk0dI7Vh7LjqdnsYmW0tbDVxxK+nume+Z9Sb4znbUKkWl+vgQATdRUEyhT2P+Gqrd0UBzYfQIDAQABo4IBRTCCAUEwDAYDVR0TBAUwAwEB/zAsBglghkgBhvhCAQ0EHxYdT3BlblNTTCBHZW5lcmF0ZWQgQ2VydGlmaWNhdGUwHQYDVR0OBBYEFNf0xkZ3zjcEI60pVPuwDqTMQygZMIHjBgNVHSMEgdswgdiAFP7k7FMk8JWVxxC14US1XTllWuN+oYG0pIGxMIGuMQswCQYDVQQGEwJVUzETMBEGA1UECBMKQ2FsaWZvcm5pYTE9MDsGA1UEChM0WE1MIFNlY3VyaXR5IExpYnJhcnkgKGh0dHA6Ly93d3cuYWxla3NleS5jb20veG1sc2VjKTEQMA4GA1UECxMHUm9vdCBDQTEWMBQGA1UEAxMNQWxla3NleSBTYW5pbjEhMB8GCSqGSIb3DQEJARYSeG1sc2VjQGFsZWtzZXkuY29tggkAr6KLuTOt2q0wDQYJKoZIhvcNAQEFBQADQQAOXBj0yICp1RmHXqnUlsppryLCW3pKBD1dkb4HWarO7RjA1yJJfBjXssrERn05kpBcrRfzou4r3DCgQFPhjxga</X509Certificate>
      </X509Data>
    </KeyInfo>
  </Signature>
</Envelope>
`
	c.Assert(string(actualSignedString), Equals, expectedSignedString)

	err = Verify(testSuite.Cert, actualSignedString, SignatureOptions{})
	c.Assert(err, IsNil)
}

func (testSuite *XMLDSigTest) TestVerifyFailsWhenMessageModified(c *C) {
	// break the document and notice that the signature is invalid
	signedStr, err := Sign(testSuite.Key, testSuite.DocStr, SignatureOptions{})
	c.Assert(err, IsNil)

	err = Verify(testSuite.Cert, signedStr, SignatureOptions{})
	c.Assert(err, IsNil)

	signedStr = []byte(strings.Replace(string(signedStr), "Hello", "Goodbye", 1))
	err = Verify(testSuite.Cert, []byte(signedStr), SignatureOptions{})
	c.Assert(err, Equals, ErrVerificationFailed)
}

func (testSuite *XMLDSigTest) TestInvalidXML(c *C) {
	_, err := Sign(testSuite.Key, []byte("<invalid xml"), SignatureOptions{})
	c.Assert(err, ErrorMatches, ".*Couldn't find end of Start Tag.*")

	_, err = Sign(testSuite.Key, []byte("<invalid></invalid>"), SignatureOptions{})
	c.Assert(err, ErrorMatches, "cannot find start node")

	_, err = Sign([]byte("XXX"), testSuite.DocStr, SignatureOptions{})
	c.Assert(err, ErrorMatches, "failed to load pem key")

	err = Verify(testSuite.Cert, []byte("<invalid xml"), SignatureOptions{})
	c.Assert(err, ErrorMatches, ".*Couldn't find end of Start Tag.*")

	err = Verify(testSuite.Cert, []byte("<invalid></invalid>"), SignatureOptions{})
	c.Assert(err, ErrorMatches, "cannot find start node")

	err = Verify([]byte("XXX"), testSuite.DocStr, SignatureOptions{})
	c.Assert(err, ErrorMatches, ".*xmlSecOpenSSLAppKeyLoadMemory.*")

	err = Verify(testSuite.Key, testSuite.DocStr, SignatureOptions{})
	c.Assert(err, ErrorMatches, ".*xmlSecOpenSSLAppKeyLoadMemory.*")

	err = Verify(testSuite.Cert, testSuite.DocStr, SignatureOptions{})
	c.Assert(err, ErrorMatches, "signature verification failed")
}

func (testSuite *XMLDSigTest) TestVerifySAMLSignature(c *C) {
	cert := []byte(`-----BEGIN CERTIFICATE-----
MIIEDjCCAvagAwIBAgIBADANBgkqhkiG9w0BAQUFADBnMQswCQYDVQQGEwJVUzEV
MBMGA1UECBMMUGVubnN5bHZhbmlhMRMwEQYDVQQHEwpQaXR0c2J1cmdoMREwDwYD
VQQKEwhUZXN0U2hpYjEZMBcGA1UEAxMQaWRwLnRlc3RzaGliLm9yZzAeFw0wNjA4
MzAyMTEyMjVaFw0xNjA4MjcyMTEyMjVaMGcxCzAJBgNVBAYTAlVTMRUwEwYDVQQI
EwxQZW5uc3lsdmFuaWExEzARBgNVBAcTClBpdHRzYnVyZ2gxETAPBgNVBAoTCFRl
c3RTaGliMRkwFwYDVQQDExBpZHAudGVzdHNoaWIub3JnMIIBIjANBgkqhkiG9w0B
AQEFAAOCAQ8AMIIBCgKCAQEArYkCGuTmJp9eAOSGHwRJo1SNatB5ZOKqDM9ysg7C
yVTDClcpu93gSP10nH4gkCZOlnESNgttg0r+MqL8tfJC6ybddEFB3YBo8PZajKSe
3OQ01Ow3yT4I+Wdg1tsTpSge9gEz7SrC07EkYmHuPtd71CHiUaCWDv+xVfUQX0aT
NPFmDixzUjoYzbGDrtAyCqA8f9CN2txIfJnpHE6q6CmKcoLADS4UrNPlhHSzd614
kR/JYiks0K4kbRqCQF0Dv0P5Di+rEfefC6glV8ysC8dB5/9nb0yh/ojRuJGmgMWH
gWk6h0ihjihqiu4jACovUZ7vVOCgSE5Ipn7OIwqd93zp2wIDAQABo4HEMIHBMB0G
A1UdDgQWBBSsBQ869nh83KqZr5jArr4/7b+QazCBkQYDVR0jBIGJMIGGgBSsBQ86
9nh83KqZr5jArr4/7b+Qa6FrpGkwZzELMAkGA1UEBhMCVVMxFTATBgNVBAgTDFBl
bm5zeWx2YW5pYTETMBEGA1UEBxMKUGl0dHNidXJnaDERMA8GA1UEChMIVGVzdFNo
aWIxGTAXBgNVBAMTEGlkcC50ZXN0c2hpYi5vcmeCAQAwDAYDVR0TBAUwAwEB/zAN
BgkqhkiG9w0BAQUFAAOCAQEAjR29PhrCbk8qLN5MFfSVk98t3CT9jHZoYxd8QMRL
I4j7iYQxXiGJTT1FXs1nd4Rha9un+LqTfeMMYqISdDDI6tv8iNpkOAvZZUosVkUo
93pv1T0RPz35hcHHYq2yee59HJOco2bFlcsH8JBXRSRrJ3Q7Eut+z9uo80JdGNJ4
/SJy5UorZ8KazGj16lfJhOBXldgrhppQBb0Nq6HKHguqmwRfJ+WkxemZXzhediAj
Geka8nz8JjwxpUjAiSWYKLtJhGEaTqCYxCCX2Dw+dOTqUzHOZ7WKv4JXPK5G/Uhr
8K/qhmFT2nIQi538n6rVYLeWj8Bbnl+ev0peYzxFyF5sQA==
-----END CERTIFICATE-----`)
	doc := []byte("<saml2:Assertion xmlns:saml2=\"urn:oasis:names:tc:SAML:2.0:assertion\" xmlns:xs=\"http://www.w3.org/2001/XMLSchema\" ID=\"_f6f518e2c236c9c558f7a8bc6387b103\" IssueInstant=\"2015-11-29T21:29:09.991Z\" Version=\"2.0\"><saml2:Issuer Format=\"urn:oasis:names:tc:SAML:2.0:nameid-format:entity\">https://idp.testshib.org/idp/shibboleth</saml2:Issuer><ds:Signature xmlns:ds=\"http://www.w3.org/2000/09/xmldsig#\"><ds:SignedInfo><ds:CanonicalizationMethod Algorithm=\"http://www.w3.org/2001/10/xml-exc-c14n#\"></ds:CanonicalizationMethod><ds:SignatureMethod Algorithm=\"http://www.w3.org/2001/04/xmldsig-more#rsa-sha256\"></ds:SignatureMethod><ds:Reference URI=\"#_f6f518e2c236c9c558f7a8bc6387b103\"><ds:Transforms><ds:Transform Algorithm=\"http://www.w3.org/2000/09/xmldsig#enveloped-signature\"></ds:Transform><ds:Transform Algorithm=\"http://www.w3.org/2001/10/xml-exc-c14n#\"><ec:InclusiveNamespaces xmlns:ec=\"http://www.w3.org/2001/10/xml-exc-c14n#\" PrefixList=\"xs\"></ec:InclusiveNamespaces></ds:Transform></ds:Transforms><ds:DigestMethod Algorithm=\"http://www.w3.org/2001/04/xmlenc#sha256\"></ds:DigestMethod><ds:DigestValue>VwEKsGObmOM6y22Nstadwz1fq6dnQ2aDmERPMuEteds=</ds:DigestValue></ds:Reference></ds:SignedInfo><ds:SignatureValue>gcROTzJ7HgTu/LQprki8v9J5y4et2np48hYspgmygZRvRawzxfQDgB0MBvDIBG78J5XSd401g7E999JUEh4JtSMAig1THbeWhyITGHU1Vpl2xAR5Ma0vCMLjVIleeuFHhStFBNqKirNfulfhEa7Q5THVGKrVsNuIaP/yc10Gf8AyHfCIOf/ZQGiU3Srp/pKZLXPkSKTEZIq5tAOl+pA0maFBvb4+EkMPB6E66HiXknHL9KdNh8bPcq+EkqjhtHWOy341F8W9iy6MJYGuO9ksxdiY6FK5SqmPHlgoJqXx7Et2vYME6opIgFYB6m1KW6kWgVcF0VyIzJbkXq3yTi0b5g==</ds:SignatureValue><ds:KeyInfo><ds:X509Data><ds:X509Certificate>MIIEDjCCAvagAwIBAgIBADANBgkqhkiG9w0BAQUFADBnMQswCQYDVQQGEwJVUzEVMBMGA1UECBMM\nUGVubnN5bHZhbmlhMRMwEQYDVQQHEwpQaXR0c2J1cmdoMREwDwYDVQQKEwhUZXN0U2hpYjEZMBcG\nA1UEAxMQaWRwLnRlc3RzaGliLm9yZzAeFw0wNjA4MzAyMTEyMjVaFw0xNjA4MjcyMTEyMjVaMGcx\nCzAJBgNVBAYTAlVTMRUwEwYDVQQIEwxQZW5uc3lsdmFuaWExEzARBgNVBAcTClBpdHRzYnVyZ2gx\nETAPBgNVBAoTCFRlc3RTaGliMRkwFwYDVQQDExBpZHAudGVzdHNoaWIub3JnMIIBIjANBgkqhkiG\n9w0BAQEFAAOCAQ8AMIIBCgKCAQEArYkCGuTmJp9eAOSGHwRJo1SNatB5ZOKqDM9ysg7CyVTDClcp\nu93gSP10nH4gkCZOlnESNgttg0r+MqL8tfJC6ybddEFB3YBo8PZajKSe3OQ01Ow3yT4I+Wdg1tsT\npSge9gEz7SrC07EkYmHuPtd71CHiUaCWDv+xVfUQX0aTNPFmDixzUjoYzbGDrtAyCqA8f9CN2txI\nfJnpHE6q6CmKcoLADS4UrNPlhHSzd614kR/JYiks0K4kbRqCQF0Dv0P5Di+rEfefC6glV8ysC8dB\n5/9nb0yh/ojRuJGmgMWHgWk6h0ihjihqiu4jACovUZ7vVOCgSE5Ipn7OIwqd93zp2wIDAQABo4HE\nMIHBMB0GA1UdDgQWBBSsBQ869nh83KqZr5jArr4/7b+QazCBkQYDVR0jBIGJMIGGgBSsBQ869nh8\n3KqZr5jArr4/7b+Qa6FrpGkwZzELMAkGA1UEBhMCVVMxFTATBgNVBAgTDFBlbm5zeWx2YW5pYTET\nMBEGA1UEBxMKUGl0dHNidXJnaDERMA8GA1UEChMIVGVzdFNoaWIxGTAXBgNVBAMTEGlkcC50ZXN0\nc2hpYi5vcmeCAQAwDAYDVR0TBAUwAwEB/zANBgkqhkiG9w0BAQUFAAOCAQEAjR29PhrCbk8qLN5M\nFfSVk98t3CT9jHZoYxd8QMRLI4j7iYQxXiGJTT1FXs1nd4Rha9un+LqTfeMMYqISdDDI6tv8iNpk\nOAvZZUosVkUo93pv1T0RPz35hcHHYq2yee59HJOco2bFlcsH8JBXRSRrJ3Q7Eut+z9uo80JdGNJ4\n/SJy5UorZ8KazGj16lfJhOBXldgrhppQBb0Nq6HKHguqmwRfJ+WkxemZXzhediAjGeka8nz8Jjwx\npUjAiSWYKLtJhGEaTqCYxCCX2Dw+dOTqUzHOZ7WKv4JXPK5G/Uhr8K/qhmFT2nIQi538n6rVYLeW\nj8Bbnl+ev0peYzxFyF5sQA==</ds:X509Certificate></ds:X509Data></ds:KeyInfo></ds:Signature><saml2:Subject><saml2:NameID Format=\"urn:oasis:names:tc:SAML:2.0:nameid-format:transient\" NameQualifier=\"https://idp.testshib.org/idp/shibboleth\" SPNameQualifier=\"https://15661444.ngrok.io/saml2/metadata\">_5c425656721b41a6cfa4a9c96225e082</saml2:NameID><saml2:SubjectConfirmation Method=\"urn:oasis:names:tc:SAML:2.0:cm:bearer\"><saml2:SubjectConfirmationData Address=\"75.144.86.91\" InResponseTo=\"id-3d21faf29a101222d740735fa512f161\" NotOnOrAfter=\"2015-11-29T21:34:09.991Z\" Recipient=\"https://15661444.ngrok.io/saml2/acs\"></saml2:SubjectConfirmationData></saml2:SubjectConfirmation></saml2:Subject><saml2:Conditions NotBefore=\"2015-11-29T21:29:09.991Z\" NotOnOrAfter=\"2015-11-29T21:34:09.991Z\"><saml2:AudienceRestriction><saml2:Audience>https://15661444.ngrok.io/saml2/metadata</saml2:Audience></saml2:AudienceRestriction></saml2:Conditions><saml2:AuthnStatement AuthnInstant=\"2015-11-29T21:29:09.715Z\" SessionIndex=\"_57adf921604642bd4e1dce7f308734f0\"><saml2:SubjectLocality Address=\"75.144.86.91\"></saml2:SubjectLocality><saml2:AuthnContext><saml2:AuthnContextClassRef>urn:oasis:names:tc:SAML:2.0:ac:classes:PasswordProtectedTransport</saml2:AuthnContextClassRef></saml2:AuthnContext></saml2:AuthnStatement><saml2:AttributeStatement><saml2:Attribute FriendlyName=\"uid\" Name=\"urn:oid:0.9.2342.19200300.100.1.1\" NameFormat=\"urn:oasis:names:tc:SAML:2.0:attrname-format:uri\"><saml2:AttributeValue xmlns:xsi=\"http://www.w3.org/2001/XMLSchema-instance\" xsi:type=\"xs:string\">myself</saml2:AttributeValue></saml2:Attribute><saml2:Attribute FriendlyName=\"eduPersonAffiliation\" Name=\"urn:oid:1.3.6.1.4.1.5923.1.1.1.1\" NameFormat=\"urn:oasis:names:tc:SAML:2.0:attrname-format:uri\"><saml2:AttributeValue xmlns:xsi=\"http://www.w3.org/2001/XMLSchema-instance\" xsi:type=\"xs:string\">Member</saml2:AttributeValue><saml2:AttributeValue xmlns:xsi=\"http://www.w3.org/2001/XMLSchema-instance\" xsi:type=\"xs:string\">Staff</saml2:AttributeValue></saml2:Attribute><saml2:Attribute FriendlyName=\"eduPersonPrincipalName\" Name=\"urn:oid:1.3.6.1.4.1.5923.1.1.1.6\" NameFormat=\"urn:oasis:names:tc:SAML:2.0:attrname-format:uri\"><saml2:AttributeValue xmlns:xsi=\"http://www.w3.org/2001/XMLSchema-instance\" xsi:type=\"xs:string\">myself@testshib.org</saml2:AttributeValue></saml2:Attribute><saml2:Attribute FriendlyName=\"sn\" Name=\"urn:oid:2.5.4.4\" NameFormat=\"urn:oasis:names:tc:SAML:2.0:attrname-format:uri\"><saml2:AttributeValue xmlns:xsi=\"http://www.w3.org/2001/XMLSchema-instance\" xsi:type=\"xs:string\">And I</saml2:AttributeValue></saml2:Attribute><saml2:Attribute FriendlyName=\"eduPersonScopedAffiliation\" Name=\"urn:oid:1.3.6.1.4.1.5923.1.1.1.9\" NameFormat=\"urn:oasis:names:tc:SAML:2.0:attrname-format:uri\"><saml2:AttributeValue xmlns:xsi=\"http://www.w3.org/2001/XMLSchema-instance\" xsi:type=\"xs:string\">Member@testshib.org</saml2:AttributeValue><saml2:AttributeValue xmlns:xsi=\"http://www.w3.org/2001/XMLSchema-instance\" xsi:type=\"xs:string\">Staff@testshib.org</saml2:AttributeValue></saml2:Attribute><saml2:Attribute FriendlyName=\"givenName\" Name=\"urn:oid:2.5.4.42\" NameFormat=\"urn:oasis:names:tc:SAML:2.0:attrname-format:uri\"><saml2:AttributeValue xmlns:xsi=\"http://www.w3.org/2001/XMLSchema-instance\" xsi:type=\"xs:string\">Me Myself</saml2:AttributeValue></saml2:Attribute><saml2:Attribute FriendlyName=\"eduPersonEntitlement\" Name=\"urn:oid:1.3.6.1.4.1.5923.1.1.1.7\" NameFormat=\"urn:oasis:names:tc:SAML:2.0:attrname-format:uri\"><saml2:AttributeValue xmlns:xsi=\"http://www.w3.org/2001/XMLSchema-instance\" xsi:type=\"xs:string\">urn:mace:dir:entitlement:common-lib-terms</saml2:AttributeValue></saml2:Attribute><saml2:Attribute FriendlyName=\"cn\" Name=\"urn:oid:2.5.4.3\" NameFormat=\"urn:oasis:names:tc:SAML:2.0:attrname-format:uri\"><saml2:AttributeValue xmlns:xsi=\"http://www.w3.org/2001/XMLSchema-instance\" xsi:type=\"xs:string\">Me Myself And I</saml2:AttributeValue></saml2:Attribute><saml2:Attribute FriendlyName=\"eduPersonTargetedID\" Name=\"urn:oid:1.3.6.1.4.1.5923.1.1.1.10\" NameFormat=\"urn:oasis:names:tc:SAML:2.0:attrname-format:uri\"><saml2:AttributeValue><saml2:NameID Format=\"urn:oasis:names:tc:SAML:2.0:nameid-format:persistent\" NameQualifier=\"https://idp.testshib.org/idp/shibboleth\" SPNameQualifier=\"https://15661444.ngrok.io/saml2/metadata\">8F+M9ovyaYNwCId0pVkVsnZYRDo=</saml2:NameID></saml2:AttributeValue></saml2:Attribute><saml2:Attribute FriendlyName=\"telephoneNumber\" Name=\"urn:oid:2.5.4.20\" NameFormat=\"urn:oasis:names:tc:SAML:2.0:attrname-format:uri\"><saml2:AttributeValue xmlns:xsi=\"http://www.w3.org/2001/XMLSchema-instance\" xsi:type=\"xs:string\">555-5555</saml2:AttributeValue></saml2:Attribute></saml2:AttributeStatement></saml2:Assertion>")

	err := Verify(cert, doc, SignatureOptions{
		XMLID: []XMLIDOption{{
			ElementName:      "Assertion",
			ElementNamespace: "urn:oasis:names:tc:SAML:2.0:assertion",
			AttributeName:    "ID",
		}},
	})
	c.Assert(err, IsNil)
}
