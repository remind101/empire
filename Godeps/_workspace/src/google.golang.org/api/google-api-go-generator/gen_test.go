package main

import (
	"bytes"
	"flag"
	"io/ioutil"
	"path/filepath"
	"testing"
)

var updateGolden = flag.Bool("update_golden", false, "If true, causes TestAPIs to update golden files")

func TestAPIs(t *testing.T) {
	names := []string{
		"arrayofarray-1",
		"arrayofmapofobjects",
		"arrayofmapofstrings",
		"blogger-3",
		"getwithoutbody",
		"mapofany",
		"mapofarrayofobjects",
		"mapofobjects",
		"mapofstrings-1",
		"quotednum",
		"resource-named-service", // blogger/v3/blogger-api.json + s/BlogUserInfo/Service/
		"variants",
	}
	for _, name := range names {
		api, err := apiFromFile(filepath.Join("testdata", name+".json"))
		if err != nil {
			t.Errorf("Error loading API testdata/%s.json: %v", name, err)
			continue
		}
		clean, err := api.GenerateCode()
		if err != nil {
			t.Errorf("Error generating code for %s: %v", name, err)
			continue
		}
		goldenFile := filepath.Join("testdata", name+".want")
		if *updateGolden {
			if err := ioutil.WriteFile(goldenFile, clean, 0644); err != nil {
				t.Fatal(err)
			}
		}
		want, err := ioutil.ReadFile(goldenFile)
		if err != nil {
			t.Error(err)
			continue
		}
		if !bytes.Equal(want, clean) {
			tf, _ := ioutil.TempFile("", "api-"+name+"-got-json.")
			tf.Write(clean)
			tf.Close()
			t.Errorf("Output for API %s differs: diff -u %s %s", name, goldenFile, tf.Name())
		}
	}
}

func TestScope(t *testing.T) {
	tests := [][]string{
		{
			"https://www.googleapis.com/auth/somescope",
			"SomescopeScope",
		},
		{
			"https://mail.google.com/somescope",
			"MailGoogleComSomescopeScope",
		},
		{
			"https://mail.google.com/",
			"MailGoogleComScope",
		},
	}
	for _, test := range tests {
		if got := scopeIdentifierFromURL(test[0]); got != test[1] {
			t.Errorf("scopeIdentifierFromURL(%q) got %q, want %q", test[0], got, test[1])
		}
	}
}
