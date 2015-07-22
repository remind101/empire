// Copyright 2014 Eric Holmes.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package hookshot_test

import (
	"net/http"

	"github.com/ejholmes/hookshot"
)

func HandlePing(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(200)
	w.Write([]byte(`Pong`))
}

func Example() {
	r := hookshot.NewRouter()
	r.HandleFunc("ping", HandlePing)

	http.ListenAndServe(":8080", r)
}
