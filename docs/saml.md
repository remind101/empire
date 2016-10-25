# SAML

If your organization is using an IdP (identity provider) that supports SAML, you can configure SSO in Empire using [SAML 2.0](https://en.wikipedia.org/wiki/SAML_2.0).

### How SAML authentication works

1. A User starts a Service Provider (SP) initiated login, by visiting `https://<empire>/saml/login`.
2. Empire generates a SAML AuthnRequest and redirects the user to the IdP to authenticate.
2. The IdP authenticates the user and generates a signed SAML Response, then posts this to `https://<empire>/saml/acs`.
3. Empire decodes the SAML response, verifies the signature, and validates the assertions.
4. Empire then generates a JWT API token, and presents it to the user to add to `~/.netrc`.

### Configuring SAML

In order to use the SAML authentication backend, you first have to configure your IdP. We'll use OneLogin as an example, and assume that our Empire instance is located at https://empire.acme-inc.com.

When using OneLogin, you'll want to use the "SAML Test Connector (IdP w/ attr w/ sign response)" app.

![](http://i.imgur.com/d3vKELJ.png)

Go to the **Configuration** tab, and enter the following:

* **Audience**: `https://empire.acme-inc.com/saml/metadata`
* **Recipient**: `https://empire.acme-inc.com/saml/acs`
* **ACS (Consumer) URL Validator**: `^https:\/\/empire\.acme-inc\.com\/saml\/acs`
* **ACS (Consumer) URL**: `https://empire.acme-inc.com/saml/acs`

![](http://i.imgur.com/em76B5u.png)

Now, go to the **SSO** tab and copy the **Issuer URL**. We'll set the `EMPIRE_SAML_METADATA` environment variable to this value.

Next, generate an RSA key and cert Empire to use to sign SAML requests:

```console
$ openssl req -x509 -newkey rsa:2048 -keyout saml.key -out saml.cert -days 365 -nodes -subj "/CN=empire.acme-inc.com"
```

Use these as the values to `EMPIRE_SAML_KEY` and `EMPIRE_SAML_CERT`.

Finally, set the `EMPIRE_SERVER_AUTH` value to `saml` to enable the SAML backend.

A correctly configured SAML authentication backend would look like:

```
EMPIRE_SERVER_AUTH=saml
EMPIRE_SAML_METADATA=https://app.onelogin.com/saml/metadata/1234
EMPIRE_SAML_KEY=file:///etc/empire/saml.key
EMPIRE_SAML_CERT=file:///etc/empire/saml.cert
EMPIRE_URL=https://empire.acme-inc.com
```

### Assertions

Currently, Empire only looks at the NameID assertion, which it uses as the internal Empire user identifier. This is what will show up in Empire events when a user performs an action.
