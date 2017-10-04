package realip

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
)

func TestIsLocalAddr(t *testing.T) {
	testData := map[string]bool{
		"127.0.0.0":   true,
		"10.0.0.0":    true,
		"169.254.0.0": true,
		"192.168.0.0": true,
		"::1":         true,
		"fc00::":      true,

		"172.15.0.0": false,
		"172.16.0.0": true,
		"172.31.0.0": true,
		"172.32.0.0": false,

		"147.12.56.11": false,
	}

	for addr, isLocal := range testData {
		if isLocalAddress(addr) != isLocal {
			format := "%s should "
			if !isLocal {
				format += "not "
			}
			format += "be local address"

			t.Errorf(format, addr)
		}
	}
}

func TestIpAddrFromRemoteAddr(t *testing.T) {
	testData := map[string]string{
		"127.0.0.1:8888": "127.0.0.1",
		"ip:port":        "ip",
		"ip":             "ip",
		"12:34::0":       "12:34:",
	}

	for remoteAddr, expectedAddr := range testData {
		if actualAddr := ipAddrFromRemoteAddr(remoteAddr); actualAddr != expectedAddr {
			t.Errorf("ipAddrFromRemoteAddr of %s should be %s but get %s", remoteAddr, expectedAddr, actualAddr)
		}
	}
}

func TestRealIP(t *testing.T) {
	newRequest := func(remoteAddr, hdrRealIP, hdrForwardedFor string) *http.Request {
		h := http.Header{}
		h["X-Real-Ip"] = []string{hdrRealIP}
		h["X-Forwarded-For"] = []string{hdrForwardedFor}
		return &http.Request{
			RemoteAddr: remoteAddr,
			Header:     h,
		}
	}

	remoteAddr := "144.12.54.87"
	anotherRemoteAddr := "119.14.55.11"
	localAddr := "127.0.0.0"

	trustAll := &Resolver{
		XRealIp:       true,
		XForwardedFor: true,
	}

	testData := []struct {
		expected string
		request  *http.Request
		resolver *Resolver
	}{
		{remoteAddr, newRequest(remoteAddr, "", ""), trustAll},              // no header
		{remoteAddr, newRequest("", "", remoteAddr), trustAll},              // X-Forwarded-For: remoteAddr
		{remoteAddr, newRequest("", remoteAddr, ""), trustAll},              // X-RealIP: remoteAddr
		{localAddr, newRequest(localAddr, remoteAddr, ""), DefaultResolver}, // X-RealIP: remoteAddr (untrusted)

		// X-Forwarded-For: localAddr, remoteAddr, anotherRemoteAddr
		{anotherRemoteAddr, newRequest("", "", strings.Join([]string{localAddr, remoteAddr, anotherRemoteAddr, localAddr}, ", ")), trustAll},
	}

	for i, v := range testData {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			if actual := v.resolver.RealIP(v.request); v.expected != actual {
				t.Errorf("expected %s but got %s", v.expected, actual)
			}
		})
	}
}
