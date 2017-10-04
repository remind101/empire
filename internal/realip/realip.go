package realip

import (
	"context"
	"log"
	"net"
	"net/http"
	"sort"
	"strings"
)

var DefaultResolver = &Resolver{}

var cidrs []*net.IPNet

func init() {
	lancidrs := []string{
		"127.0.0.1/8", "10.0.0.0/8", "169.254.0.0/16", "172.16.0.0/12", "192.168.0.0/16", "::1/128", "fc00::/7",
	}

	cidrs = make([]*net.IPNet, len(lancidrs))

	for i, it := range lancidrs {
		_, cidrnet, err := net.ParseCIDR(it)
		if err != nil {
			log.Fatalf("ParseCIDR error: %v", err) // assuming I did it right above
		}

		cidrs[i] = cidrnet
	}
}

func isLocalAddress(addr string) bool {
	for i := range cidrs {
		myaddr := net.ParseIP(addr)
		if cidrs[i].Contains(myaddr) {
			return true
		}
	}

	return false
}

// Request.RemoteAddress contains port, which we want to remove i.e.:
// "[::1]:58292" => "[::1]"
func ipAddrFromRemoteAddr(s string) string {
	idx := strings.LastIndex(s, ":")
	if idx == -1 {
		return s
	}
	return s[:idx]
}

// Resolver is used to resolve the real ip address for an http request.
type Resolver struct {
	// Can be set to true if you have a trusted proxy or load balancer
	// setting this header.
	XRealIp bool

	// Can be set to true if you have a trusted proxy or load balancer
	// appending ip's to this header.
	XForwardedFor bool
}

// RealIP return client's real public IP address
// from http request headers.
func (r *Resolver) RealIP(req *http.Request) string {
	var hdrRealIP string
	if r.XRealIp {
		hdrRealIP = req.Header.Get("X-Real-Ip")
	}

	var hdrForwardedFor string
	if r.XForwardedFor {
		hdrForwardedFor = req.Header.Get("X-Forwarded-For")
	}

	if len(hdrForwardedFor) == 0 && len(hdrRealIP) == 0 {
		return ipAddrFromRemoteAddr(req.RemoteAddr)
	}

	// X-Forwarded-For is potentially a list of addresses separated with ","
	forwarded := sort.StringSlice(strings.Split(hdrForwardedFor, ","))

	// This will ignore the X-Forwarded-For entries matching a load balancer
	// and use the first (from right to left) untrused address as the real
	// ip. This is done to prevent spoofing X-Forwarded-For.
	//
	// For example, let's say you wanted try to spoof your ip to make it
	// look like a request came from an office ip (204.28.121.211). You
	// would make a request like this:
	//
	//   curl https://www.example.com/debug -H "X-Forwarded-For: 204.28.121.211"
	//
	// The load balancer would then tag on the connecting ip, as well as the
	// ip address of the load balancer itself. The application would receive
	// an X-Forwarded-For header like the following:
	//
	//   "X-Forwarded-For": [
	//     "204.28.121.211, 49.228.250.246, 10.128.21.180"
	//   ]
	//
	// This will look at each ip from right to left, and use the first
	// "untrusted" address as the real ip.
	//
	// 1. The first ip, 10.128.21.180, is the loadbalancer ip address and is
	//    considered trusted because it's a LAN cidr.
	// 2. The second ip, 49.228.250.246, is untrusted, so this is determined to
	//    be the real ip address.
	//
	// By doing this, the spoofed ip (204.28.121.211) is ignored.
	reverse(forwarded)

	for _, addr := range forwarded {
		// return first non-local address
		addr = strings.TrimSpace(addr)
		if len(addr) > 0 && !isLocalAddress(addr) {
			return addr
		}
	}

	return hdrRealIP
}

// Middleware is a simple http.Handler middleware that extracts the RealIP from
// the request and set it on the request context. Anything downstream can then
// simply call realip.RealIP to extract the real ip from the request.
func Middleware(h http.Handler, r *Resolver) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		ip := r.RealIP(req)
		h.ServeHTTP(w, req.WithContext(context.WithValue(req.Context(), realIPKey, ip)))
	})
}

// Extracts the real ip from the request context.
func RealIP(req *http.Request) string {
	ip, ok := req.Context().Value(realIPKey).(string)
	if !ok {
		// Fallback to a secure resolver.
		return DefaultResolver.RealIP(req)
	}
	return ip
}

// reverse reverses a slice of strings.
func reverse(ss []string) {
	last := len(ss) - 1
	for i := 0; i < len(ss)/2; i++ {
		ss[i], ss[last-i] = ss[last-i], ss[i]
	}
}

type key int

var realIPKey key = 0
