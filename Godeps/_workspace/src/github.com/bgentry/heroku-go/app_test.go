package heroku

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/bgentry/testnet"
)

//
// AppInfo()
//

const appMarshaled = `{
	"archived_at": "2012-01-01T12:00:00Z",
	"buildpack_provided_description": "Ruby/Rack",
	"created_at": "2012-01-01T12:00:00Z",
	"git_url": "git@heroku.com/example.git",
	"id": "01234567-89ab-cdef-0123-456789abcdef",
	"maintenance": true,
	"name": "example",
	"owner": {
		"email": "username@example.com",
		"id": "01234567-89ab-cdef-0123-456789abcdef"
	},
	"region": {
		"id": "01234567-89ab-cdef-0123-456789abcdef",
		"name": "us"
	},
	"released_at": "2012-01-01T12:00:00Z",
	"repo_size": 1,
	"slug_size": 1,
	"stack": {
		"id": "01234567-89ab-cdef-0123-456789abcdef",
		"name": "cedar"
	},
	"updated_at": "2012-01-01T12:00:00Z",
	"web_url": "http://example.herokuapp.com"
}`

func TestAppUnmarshal(t *testing.T) {
	var app App
	err := json.Unmarshal([]byte(appMarshaled), &app)
	if err != nil {
		t.Fatal(err)
	}

	if app.ArchivedAt == nil {
		t.Error("expected ArchivedAt to be set, was nil")
	} else if *app.ArchivedAt != time.Unix(int64(1325419200), int64(0)).UTC() {
		t.Errorf("expected ArchivedAt to be 2012-01-01T12:00:00Z, was %s", *app.ArchivedAt)
	}
	testStringsEqual(t, "app.Id", "01234567-89ab-cdef-0123-456789abcdef", app.Id)
	testStringsEqual(t, "app.Name", "example", app.Name)
	testStringsEqual(t, "app.Owner.Email", "username@example.com", app.Owner.Email)
	testStringsEqual(t, "app.Region.Name", "us", app.Region.Name)
	testStringsEqual(t, "app.Stack.Name", "cedar", app.Stack.Name)
}

var appInfoResponse = testnet.TestResponse{
	Status: 200,
	Body:   appMarshaled,
}
var appInfoRequest = newTestRequest("GET", "/apps/example", "", appInfoResponse)

func TestAppInfoSuccess(t *testing.T) {
	ts, handler, c := newTestServerAndClient(t, appInfoRequest)
	defer ts.Close()

	app, err := c.AppInfo("example")
	if err != nil {
		t.Fatal(err)
	}
	if !handler.AllRequestsCalled() {
		t.Errorf("not all expected requests were called")
	}
	if app == nil {
		t.Fatal("no app object returned")
	}
	var emptyapp App
	if *app == emptyapp {
		t.Errorf("returned app is empty")
	}
}

//
// AppList()
//

var appListResponse = testnet.TestResponse{
	Status: 200,
	Body:   "[" + appMarshaled + "]",
}
var appListRequest = newTestRequest("GET", "/apps", "", appListResponse)

func TestAppListSuccess(t *testing.T) {
	appListRequest.Header.Set("Range", "..; max=1")

	ts, handler, c := newTestServerAndClient(t, appListRequest)
	defer ts.Close()

	lr := ListRange{Max: 1}
	apps, err := c.AppList(&lr)
	if err != nil {
		t.Fatal(err)
	}
	if !handler.AllRequestsCalled() {
		t.Errorf("not all expected requests were called")
	}
	if len(apps) != 1 {
		t.Fatalf("expected 1 app, got %d", len(apps))
	}
	var emptyapp App
	if apps[0] == emptyapp {
		t.Errorf("returned app is empty")
	}
}

//
// AppCreate()
//

func TestAppCreateMarshal(t *testing.T) {
	nameval := "example"
	regionval := "us"
	stackval := "cedar"

	var marshalTests = []struct {
		values AppCreateOpts
		output string
	}{
		{AppCreateOpts{}, `{}`},
		{AppCreateOpts{Name: &nameval}, `{"name":"example"}`},
		{AppCreateOpts{Region: &regionval}, `{"region":"us"}`},
		{AppCreateOpts{Stack: &stackval}, `{"stack":"cedar"}`},
		{AppCreateOpts{
			Name:   &nameval,
			Region: &regionval,
			Stack:  &stackval,
		}, `{"name":"example","region":"us","stack":"cedar"}`},
	}

	for _, body := range marshalTests {
		bytes, err := json.Marshal(body.values)
		if err != nil {
			t.Fatal(err)
		}
		if string(bytes) != body.output {
			t.Errorf("expected %s, got %s", body.output, string(bytes))
		}
	}
}

func TestAppCreateSuccess(t *testing.T) {
	appCreateResponse := testnet.TestResponse{
		Status: 201,
		Body:   appMarshaled,
	}
	appCreateRequest := newTestRequest("POST", "/apps", "", appCreateResponse)
	appCreateRequest.Matcher = testnet.RequestBodyMatcherWithContentType("", "")

	ts, handler, c := newTestServerAndClient(t, appCreateRequest)
	defer ts.Close()

	app, err := c.AppCreate(nil)
	if err != nil {
		t.Fatal(err)
	}
	if app == nil {
		t.Fatal("no app object returned")
	}
	var emptyapp App
	if *app == emptyapp {
		t.Errorf("returned app is empty")
	}

	if !handler.AllRequestsCalled() {
		t.Errorf("not all expected requests were called")
	}
}

func TestAppCreateSuccessWithOpts(t *testing.T) {
	appCreateResponse := testnet.TestResponse{
		Status: 201,
		Body:   appMarshaled,
	}
	appCreateRequestBody := `{"name":"example", "region":"us", "stack":"cedar"}`
	appCreateRequest := newTestRequest("POST", "/apps", appCreateRequestBody, appCreateResponse)

	nameval := "example"
	regionval := "us"
	stackval := "cedar"
	appCreateRequestOptions := AppCreateOpts{Name: &nameval, Region: &regionval, Stack: &stackval}

	ts, handler, c := newTestServerAndClient(t, appCreateRequest)
	defer ts.Close()

	app, err := c.AppCreate(&appCreateRequestOptions)
	if err != nil {
		t.Fatal(err)
	}
	if app == nil {
		t.Fatal("no app object returned")
	}
	var emptyapp App
	if *app == emptyapp {
		t.Errorf("returned app is empty")
	}

	if !handler.AllRequestsCalled() {
		t.Errorf("not all expected requests were called")
	}
}

//
// AppDelete()
//

var appDeleteResponse = testnet.TestResponse{
	Status: 200,
	Body:   appMarshaled,
}
var appDeleteRequest = newTestRequest("DELETE", "/apps/example", "", appDeleteResponse)

func TestAppDeleteSuccess(t *testing.T) {
	ts, handler, c := newTestServerAndClient(t, appDeleteRequest)
	defer ts.Close()

	err := c.AppDelete("example")
	if err != nil {
		t.Fatal(err)
	}
	if !handler.AllRequestsCalled() {
		t.Errorf("not all expected requests were called")
	}
}

//
// AppUpdate()
//

func TestAppUpdateMarshal(t *testing.T) {
	maintval := true
	nameval := "example"

	var marshalTests = []struct {
		values AppUpdateOpts
		output string
	}{
		{AppUpdateOpts{Maintenance: &maintval}, `{"maintenance":true}`},
		{AppUpdateOpts{Name: &nameval}, `{"name":"example"}`},
		{AppUpdateOpts{Maintenance: &maintval, Name: &nameval}, `{"maintenance":true,"name":"example"}`},
	}

	for _, body := range marshalTests {
		bytes, err := json.Marshal(body.values)
		if err != nil {
			t.Fatal(err)
		}
		if string(bytes) != body.output {
			t.Errorf("expected %s, got %s", body.output, string(bytes))
		}
	}
}

func TestAppUpdateSuccess(t *testing.T) {
	appUpdateResponse := testnet.TestResponse{
		Status: 201,
		Body:   appMarshaled,
	}
	appUpdateRequestBody := `{"maintenance":true, "name":"example"}`
	appUpdateRequest := newTestRequest("PATCH", "/apps/example", appUpdateRequestBody, appUpdateResponse)

	maintval := true
	nameval := "example"
	appUpdateRequestOptions := AppUpdateOpts{Maintenance: &maintval, Name: &nameval}

	ts, handler, c := newTestServerAndClient(t, appUpdateRequest)
	defer ts.Close()

	app, err := c.AppUpdate("example", &appUpdateRequestOptions)
	if err != nil {
		t.Fatal(err)
	}
	if app == nil {
		t.Fatal("no app object returned")
	}
	var emptyapp App
	if *app == emptyapp {
		t.Errorf("returned app is empty")
	}

	if !handler.AllRequestsCalled() {
		t.Errorf("not all expected requests were called")
	}
}
