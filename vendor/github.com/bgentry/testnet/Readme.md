# Testnet

This package provides helpers for testing interactions with an HTTP API in the
Go (#golang) programming language. It allows you to test that the expected HTTP
requests are received by the `httptest.Server`, and return mock responses.

Testnet was imported directly from the [CloudFoundry Go CLI][cf-cli]'s [test
helpers](cf-testnet). The [license](./LICENSE) has been copied exactly from the
source (though I attempted to fill in the correct owner in the boilerplate
copyright notice).

[cf-cli]: https://github.com/cloudfoundry/cli "CloudFoundry Go CLI"
[cf-testnet]: https://github.com/cloudfoundry/cli/commits/master/src/testhelpers/net/ "CloudFoundry Go CLI testhelpers/net"
