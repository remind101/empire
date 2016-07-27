// Copyright 2013 Ooyala, Inc.

package statsd

import (
	"fmt"
	"io"
	"net"
	"reflect"
	"strings"
	"testing"
)

var dogstatsdTests = []struct {
	GlobalNamespace string
	GlobalTags      []string
	Method          string
	Metric          string
	Value           interface{}
	Tags            []string
	Rate            float64
	Expected        string
}{
	{"", nil, "Gauge", "test.gauge", 1.0, nil, 1.0, "test.gauge:1.000000|g"},
	{"", nil, "Gauge", "test.gauge", 1.0, nil, 0.999999, "test.gauge:1.000000|g|@0.999999"},
	{"", nil, "Gauge", "test.gauge", 1.0, []string{"tagA"}, 1.0, "test.gauge:1.000000|g|#tagA"},
	{"", nil, "Gauge", "test.gauge", 1.0, []string{"tagA", "tagB"}, 1.0, "test.gauge:1.000000|g|#tagA,tagB"},
	{"", nil, "Gauge", "test.gauge", 1.0, []string{"tagA"}, 0.999999, "test.gauge:1.000000|g|@0.999999|#tagA"},
	{"", nil, "Count", "test.count", int64(1), []string{"tagA"}, 1.0, "test.count:1|c|#tagA"},
	{"", nil, "Count", "test.count", int64(-1), []string{"tagA"}, 1.0, "test.count:-1|c|#tagA"},
	{"", nil, "Histogram", "test.histogram", 2.3, []string{"tagA"}, 1.0, "test.histogram:2.300000|h|#tagA"},
	{"", nil, "Set", "test.set", "uuid", []string{"tagA"}, 1.0, "test.set:uuid|s|#tagA"},
	{"flubber.", nil, "Set", "test.set", "uuid", []string{"tagA"}, 1.0, "flubber.test.set:uuid|s|#tagA"},
	{"", []string{"tagC"}, "Set", "test.set", "uuid", []string{"tagA"}, 1.0, "test.set:uuid|s|#tagC,tagA"},
}

func assertNotPanics(t *testing.T, f func()) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatal(r)
		}
	}()
	f()
}

func TestClient(t *testing.T) {
	addr := "localhost:1201"
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		t.Fatal(err)
	}

	server, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()

	client, err := New(addr)
	if err != nil {
		t.Fatal(err)
	}

	for _, tt := range dogstatsdTests {
		client.Namespace = tt.GlobalNamespace
		client.Tags = tt.GlobalTags
		method := reflect.ValueOf(client).MethodByName(tt.Method)
		e := method.Call([]reflect.Value{
			reflect.ValueOf(tt.Metric),
			reflect.ValueOf(tt.Value),
			reflect.ValueOf(tt.Tags),
			reflect.ValueOf(tt.Rate)})[0]
		errInter := e.Interface()
		if errInter != nil {
			t.Fatal(errInter.(error))
		}

		bytes := make([]byte, 1024)
		n, err := server.Read(bytes)
		if err != nil {
			t.Fatal(err)
		}
		message := bytes[:n]
		if string(message) != tt.Expected {
			t.Errorf("Expected: %s. Actual: %s", tt.Expected, string(message))
		}
	}
}

func TestBufferedClient(t *testing.T) {
	addr := "localhost:1201"
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		t.Fatal(err)
	}

	server, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()

	conn, err := net.DialUDP("udp", nil, udpAddr)
	if err != nil {
		t.Fatal(err)
	}

	bufferLength := 5
	client := &Client{
		conn:         conn,
		commands:     make([]string, 0, bufferLength),
		bufferLength: bufferLength,
	}

	client.Namespace = "foo."
	client.Tags = []string{"dd:2"}

	client.Count("cc", 1, nil, 1)
	client.Gauge("gg", 10, nil, 1)
	client.Histogram("hh", 1, nil, 1)
	client.Set("ss", "ss", nil, 1)

	if len(client.commands) != 4 {
		t.Errorf("Expected client to have buffered 4 commands, but found %d\n", len(client.commands))
	}

	client.Set("ss", "xx", nil, 1)
	err = client.flush()
	if err != nil {
		t.Errorf("Error sending: %s", err)
	}

	if len(client.commands) != 0 {
		t.Errorf("Expecting send to flush commands, but found %d\n", len(client.commands))
	}

	buffer := make([]byte, 4096)
	n, err := io.ReadAtLeast(server, buffer, 1)
	result := string(buffer[:n])

	if err != nil {
		t.Error(err)
	}

	expected := []string{
		`foo.cc:1|c|#dd:2`,
		`foo.gg:10.000000|g|#dd:2`,
		`foo.hh:1.000000|h|#dd:2`,
		`foo.ss:ss|s|#dd:2`,
		`foo.ss:xx|s|#dd:2`,
	}

	for i, res := range strings.Split(result, "\n") {
		if res != expected[i] {
			t.Errorf("Got `%s`, expected `%s`", res, expected[i])
		}
	}

	client.Event(&Event{Title: "title1", Text: "text1", Priority: Normal, AlertType: Success, Tags: []string{"tagg"}})
	client.SimpleEvent("event1", "text1")

	if len(client.commands) != 2 {
		t.Errorf("Expected to find %d commands, but found %d\n", 2, len(client.commands))
	}

	err = client.flush()

	if err != nil {
		t.Errorf("Error sending: %s", err)
	}

	if len(client.commands) != 0 {
		t.Errorf("Expecting send to flush commands, but found %d\n", len(client.commands))
	}

	buffer = make([]byte, 1024)
	n, err = io.ReadAtLeast(server, buffer, 1)
	result = string(buffer[:n])

	if err != nil {
		t.Error(err)
	}

	if n == 0 {
		t.Errorf("Read 0 bytes but expected more.")
	}

	expected = []string{
		`_e{6,5}:title1|text1|p:normal|t:success|#dd:2,tagg`,
		`_e{6,5}:event1|text1|#dd:2`,
	}

	for i, res := range strings.Split(result, "\n") {
		if res != expected[i] {
			t.Errorf("Got `%s`, expected `%s`", res, expected[i])
		}
	}

}

func TestJoinMaxSize(t *testing.T) {
	c := Client{}
	elements := []string{"abc", "abcd", "ab", "xyz", "foobaz", "x", "wwxxyyzz"}
	res, n := c.joinMaxSize(elements, " ", 8)

	if len(res) != len(n) && len(res) != 4 {
		t.Errorf("Was expecting 4 frames to flush but got: %v - %v", n, res)
	}
	if n[0] != 2 {
		t.Errorf("Was expecting 2 elements in first frame but got: %v", n[0])
	}
	if string(res[0]) != "abc abcd" {
		t.Errorf("Join should have returned \"abc abcd\" in frame, but found: %s", res[0])
	}
	if n[1] != 2 {
		t.Errorf("Was expecting 2 elements in second frame but got: %v - %v", n[1], n)
	}
	if string(res[1]) != "ab xyz" {
		t.Errorf("Join should have returned \"ab xyz\" in frame, but found: %s", res[1])
	}
	if n[2] != 2 {
		t.Errorf("Was expecting 2 elements in third frame but got: %v - %v", n[2], n)
	}
	if string(res[2]) != "foobaz x" {
		t.Errorf("Join should have returned \"foobaz x\" in frame, but found: %s", res[2])
	}
	if n[3] != 1 {
		t.Errorf("Was expecting 1 element in fourth frame but got: %v - %v", n[3], n)
	}
	if string(res[3]) != "wwxxyyzz" {
		t.Errorf("Join should have returned \"wwxxyyzz\" in frame, but found: %s", res[3])
	}

	res, n = c.joinMaxSize(elements, " ", 11)

	if len(res) != len(n) && len(res) != 3 {
		t.Errorf("Was expecting 3 frames to flush but got: %v - %v", n, res)
	}
	if n[0] != 3 {
		t.Errorf("Was expecting 3 elements in first frame but got: %v", n[0])
	}
	if string(res[0]) != "abc abcd ab" {
		t.Errorf("Join should have returned \"abc abcd ab\" in frame, but got: %s", res[0])
	}
	if n[1] != 2 {
		t.Errorf("Was expecting 2 elements in second frame but got: %v", n[1])
	}
	if string(res[1]) != "xyz foobaz" {
		t.Errorf("Join should have returned \"xyz foobaz\" in frame, but got: %s", res[1])
	}
	if n[2] != 2 {
		t.Errorf("Was expecting 2 elements in third frame but got: %v", n[2])
	}
	if string(res[2]) != "x wwxxyyzz" {
		t.Errorf("Join should have returned \"x wwxxyyzz\" in frame, but got: %s", res[2])
	}

	res, n = c.joinMaxSize(elements, "    ", 8)

	if len(res) != len(n) && len(res) != 7 {
		t.Errorf("Was expecting 7 frames to flush but got: %v - %v", n, res)
	}
	if n[0] != 1 {
		t.Errorf("Separator is long, expected a single element in frame but got: %d - %v", n[0], res)
	}
	if string(res[0]) != "abc" {
		t.Errorf("Join should have returned \"abc\" in first frame, but got: %s", res)
	}
	if n[1] != 1 {
		t.Errorf("Separator is long, expected a single element in frame but got: %d - %v", n[1], res)
	}
	if string(res[1]) != "abcd" {
		t.Errorf("Join should have returned \"abcd\" in second frame, but got: %s", res[1])
	}
	if n[2] != 1 {
		t.Errorf("Separator is long, expected a single element in third frame but got: %d - %v", n[2], res)
	}
	if string(res[2]) != "ab" {
		t.Errorf("Join should have returned \"ab\" in third frame, but got: %s", res[2])
	}
	if n[3] != 1 {
		t.Errorf("Separator is long, expected a single element in fourth frame but got: %d - %v", n[3], res)
	}
	if string(res[3]) != "xyz" {
		t.Errorf("Join should have returned \"xyz\" in fourth frame, but got: %s", res[3])
	}
	if n[4] != 1 {
		t.Errorf("Separator is long, expected a single element in fifth frame but got: %d - %v", n[4], res)
	}
	if string(res[4]) != "foobaz" {
		t.Errorf("Join should have returned \"foobaz\" in fifth frame, but got: %s", res[4])
	}
	if n[5] != 1 {
		t.Errorf("Separator is long, expected a single element in sixth frame but got: %d - %v", n[5], res)
	}
	if string(res[5]) != "x" {
		t.Errorf("Join should have returned \"x\" in sixth frame, but got: %s", res[5])
	}
	if n[6] != 1 {
		t.Errorf("Separator is long, expected a single element in seventh frame but got: %d - %v", n[6], res)
	}
	if string(res[6]) != "wwxxyyzz" {
		t.Errorf("Join should have returned \"wwxxyyzz\" in seventh frame, but got: %s", res[6])
	}

	res, n = c.joinMaxSize(elements[4:], " ", 6)
	if len(res) != len(n) && len(res) != 3 {
		t.Errorf("Was expecting 3 frames to flush but got: %v - %v", n, res)

	}
	if n[0] != 1 {
		t.Errorf("Element should just fit in frame - expected single element in frame: %d - %v", n[0], res)
	}
	if string(res[0]) != "foobaz" {
		t.Errorf("Join should have returned \"foobaz\" in first frame, but got: %s", res[0])
	}
	if n[1] != 1 {
		t.Errorf("Single element expected in frame, but got. %d - %v", n[1], res)
	}
	if string(res[1]) != "x" {
		t.Errorf("Join should' have returned \"x\" in second frame, but got: %s", res[1])
	}
	if n[2] != 1 {
		t.Errorf("Even though element is greater then max size we still try to send it. %d - %v", n[2], res)
	}
	if string(res[2]) != "wwxxyyzz" {
		t.Errorf("Join should have returned \"wwxxyyzz\" in third frame, but got: %s", res[2])
	}
}

func testSendMsg(t *testing.T) {
	c := Client{bufferLength: 1}
	err := c.sendMsg(strings.Repeat("x", OptimalPayloadSize))
	if err != nil {
		t.Errorf("Expected no error to be returned if message size is smaller or equal to MaxPayloadSize, got: %s", err.Error())
	}
	err = c.sendMsg(strings.Repeat("x", OptimalPayloadSize+1))
	if err == nil {
		t.Error("Expected error to be returned if message size is bigger that MaxPayloadSize")
	}
}

func TestNilSafe(t *testing.T) {
	var c *Client
	assertNotPanics(t, func() { c.Close() })
	assertNotPanics(t, func() { c.Count("", 0, nil, 1) })
	assertNotPanics(t, func() { c.Histogram("", 0, nil, 1) })
	assertNotPanics(t, func() { c.Gauge("", 0, nil, 1) })
	assertNotPanics(t, func() { c.Set("", "", nil, 1) })
	assertNotPanics(t, func() { c.send("", "", nil, 1) })
}

func TestEvents(t *testing.T) {
	matrix := []struct {
		event   *Event
		encoded string
	}{
		{
			NewEvent("Hello", "Something happened to my event"),
			`_e{5,30}:Hello|Something happened to my event`,
		}, {
			&Event{Title: "hi", Text: "okay", AggregationKey: "foo"},
			`_e{2,4}:hi|okay|k:foo`,
		}, {
			&Event{Title: "hi", Text: "okay", AggregationKey: "foo", AlertType: Info},
			`_e{2,4}:hi|okay|k:foo|t:info`,
		}, {
			&Event{Title: "hi", Text: "w/e", AlertType: Error, Priority: Normal},
			`_e{2,3}:hi|w/e|p:normal|t:error`,
		}, {
			&Event{Title: "hi", Text: "uh", Tags: []string{"host:foo", "app:bar"}},
			`_e{2,2}:hi|uh|#host:foo,app:bar`,
		},
	}

	for _, m := range matrix {
		r, err := m.event.Encode()
		if err != nil {
			t.Errorf("Error encoding: %s\n", err)
			continue
		}
		if r != m.encoded {
			t.Errorf("Expected `%s`, got `%s`\n", m.encoded, r)
		}
	}

	e := NewEvent("", "hi")
	if _, err := e.Encode(); err == nil {
		t.Errorf("Expected error on empty Title.")
	}

	e = NewEvent("hi", "")
	if _, err := e.Encode(); err == nil {
		t.Errorf("Expected error on empty Text.")
	}

	e = NewEvent("hello", "world")
	s, err := e.Encode("tag1", "tag2")
	if err != nil {
		t.Error(err)
	}
	expected := "_e{5,5}:hello|world|#tag1,tag2"
	if s != expected {
		t.Errorf("Expected %s, got %s", expected, s)
	}
	if len(e.Tags) != 0 {
		t.Errorf("Modified event in place illegally.")
	}
}

// These benchmarks show that using a buffer instead of sprintf-ing together
// a bunch of intermediate strings is 4-5x faster

func BenchmarkFormatNew(b *testing.B) {
	b.StopTimer()
	c := &Client{}
	c.Namespace = "foo.bar."
	c.Tags = []string{"app:foo", "host:bar"}
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		c.format("system.cpu.idle", "10", []string{"foo"}, 1)
		c.format("system.cpu.load", "0.1", nil, 0.9)
	}
}

// Old formatting function, added to client for tests
func (c *Client) formatOld(name, value string, tags []string, rate float64) string {
	if rate < 1 {
		value = fmt.Sprintf("%s|@%f", value, rate)
	}
	if c.Namespace != "" {
		name = fmt.Sprintf("%s%s", c.Namespace, name)
	}

	tags = append(c.Tags, tags...)
	if len(tags) > 0 {
		value = fmt.Sprintf("%s|#%s", value, strings.Join(tags, ","))
	}

	return fmt.Sprintf("%s:%s", name, value)

}

func BenchmarkFormatOld(b *testing.B) {
	b.StopTimer()
	c := &Client{}
	c.Namespace = "foo.bar."
	c.Tags = []string{"app:foo", "host:bar"}
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		c.formatOld("system.cpu.idle", "10", []string{"foo"}, 1)
		c.formatOld("system.cpu.load", "0.1", nil, 0.9)
	}
}
