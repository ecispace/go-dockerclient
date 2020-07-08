// Copyright 2014 go-dockerclient authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package docker

import (
	"bufio"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

var TimeLayout = "2006-01-02T15:04:05Z"

func Test_DockerEventsInited(t *testing.T) {
	timepoint := "2020-06-01T15:49:29Z"
	timepointObj,_ := time.Parse(TimeLayout,timepoint)
	timepointStamp := timepointObj.Unix()
	log.Print(timepointStamp)

	client, err := NewClientFromEnv()
	if err != nil {
		// handle err
	}

	eventsChan := make(chan *APIEvents)
	client.AddEventListener(timepointStamp, eventsChan)

	//errChan :=make(chan error)
	//client.EventHijack(timepointStamp, eventsChan, errChan)

	for {
		select {
		case event, _ := <-eventsChan:
			fmt.Println("--------- ", *event)
		}
	}
}


func Test_DockerEventWatched(t *testing.T) {
	client, err := NewClientFromEnv()
	if err != nil {
		// handle err
	}
	//imgs, err := client.ListImages(docker.ListImagesOptions{All: false})

	eventsChan := make(chan *APIEvents)
	client.AddEventListener(-1, eventsChan)

	for {
		select {
		case event, _ := <-eventsChan:
			fmt.Println("--------- ", *event)
		}
	}
}

func TestEventListeners(t *testing.T) {
	t.Parallel()
	testEventListeners("TestEventListeners", t, httptest.NewServer, NewClient)
}

func TestTLSEventListeners(t *testing.T) {
	t.Parallel()
	testEventListeners("TestTLSEventListeners", t, func(handler http.Handler) *httptest.Server {
		server := httptest.NewUnstartedServer(handler)

		cert, err := tls.LoadX509KeyPair("testing/data/server.pem", "testing/data/serverkey.pem")
		if err != nil {
			t.Fatalf("Error loading server key pair: %s", err)
		}

		caCert, err := ioutil.ReadFile("testing/data/ca.pem")
		if err != nil {
			t.Fatalf("Error loading ca certificate: %s", err)
		}
		caPool := x509.NewCertPool()
		if !caPool.AppendCertsFromPEM(caCert) {
			t.Fatalf("Could not add ca certificate")
		}

		server.TLS = &tls.Config{
			Certificates: []tls.Certificate{cert},
			RootCAs:      caPool,
		}
		server.StartTLS()
		return server
	}, func(url string) (*Client, error) {
		return NewTLSClient(url, "testing/data/cert.pem", "testing/data/key.pem", "testing/data/ca.pem")
	})
}

func testEventListeners(testName string, t *testing.T, buildServer func(http.Handler) *httptest.Server, buildClient func(string) (*Client, error)) {
	response := `{"action":"pull","type":"image","actor":{"id":"busybox:latest","attributes":{}},"time":1442421700,"timeNano":1442421700598988358}
{"action":"create","type":"container","actor":{"id":"5745704abe9caa5","attributes":{"image":"busybox"}},"time":1442421716,"timeNano":1442421716853979870}
{"action":"attach","type":"container","actor":{"id":"5745704abe9caa5","attributes":{"image":"busybox"}},"time":1442421716,"timeNano":1442421716894759198}
{"action":"start","type":"container","actor":{"id":"5745704abe9caa5","attributes":{"image":"busybox"}},"time":1442421716,"timeNano":1442421716983607193}
{"status":"create","id":"dfdf82bd3881","from":"base:latest","time":1374067924}
{"status":"start","id":"dfdf82bd3881","from":"base:latest","time":1374067924}
{"status":"stop","id":"dfdf82bd3881","from":"base:latest","time":1374067966}
{"status":"destroy","id":"dfdf82bd3881","from":"base:latest","time":1374067970}
{"Action":"create","Actor":{"Attributes":{"HAProxyMode":"http","HealthCheck":"HttpGet","HealthCheckArgs":"http://127.0.0.1:39051/status/check","ServicePort_8080":"17801","image":"datanerd.us/siteeng/sample-app-go:latest","name":"sample-app-client-go-69818c1223ddb5"},"ID":"a925eaf4084d5c3bcf337b2abb05f566ebb94276dff34f6effb00d8ecd380e16"},"Type":"container","from":"datanerd.us/siteeng/sample-app-go:latest","id":"a925eaf4084d5c3bcf337b2abb05f566ebb94276dff34f6effb00d8ecd380e16","status":"create","time":1459133932,"timeNano":1459133932961735842}`

	server := buildServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		rsc := bufio.NewScanner(strings.NewReader(response))
		for rsc.Scan() {
			w.Write(rsc.Bytes())
			w.(http.Flusher).Flush()
			time.Sleep(10 * time.Millisecond)
		}
	}))
	defer server.Close()

	wantedEvents := []APIEvents{
		{
			Action: "pull",
			Type:   "image",
			Actor: APIActor{
				ID:         "busybox:latest",
				Attributes: map[string]string{},
			},

			Status: "pull",
			ID:     "busybox:latest",

			Time:     1442421700,
			TimeNano: 1442421700598988358,
		},
		{
			Action: "create",
			Type:   "container",
			Actor: APIActor{
				ID: "5745704abe9caa5",
				Attributes: map[string]string{
					"image": "busybox",
				},
			},

			Status: "create",
			ID:     "5745704abe9caa5",
			From:   "busybox",

			Time:     1442421716,
			TimeNano: 1442421716853979870,
		},
		{
			Action: "attach",
			Type:   "container",
			Actor: APIActor{
				ID: "5745704abe9caa5",
				Attributes: map[string]string{
					"image": "busybox",
				},
			},

			Status: "attach",
			ID:     "5745704abe9caa5",
			From:   "busybox",

			Time:     1442421716,
			TimeNano: 1442421716894759198,
		},
		{
			Action: "start",
			Type:   "container",
			Actor: APIActor{
				ID: "5745704abe9caa5",
				Attributes: map[string]string{
					"image": "busybox",
				},
			},

			Status: "start",
			ID:     "5745704abe9caa5",
			From:   "busybox",

			Time:     1442421716,
			TimeNano: 1442421716983607193,
		},

		{
			Action: "create",
			Type:   "container",
			Actor: APIActor{
				ID: "dfdf82bd3881",
				Attributes: map[string]string{
					"image": "base:latest",
				},
			},

			Status: "create",
			ID:     "dfdf82bd3881",
			From:   "base:latest",

			Time: 1374067924,
		},
		{
			Action: "start",
			Type:   "container",
			Actor: APIActor{
				ID: "dfdf82bd3881",
				Attributes: map[string]string{
					"image": "base:latest",
				},
			},

			Status: "start",
			ID:     "dfdf82bd3881",
			From:   "base:latest",

			Time: 1374067924,
		},
		{
			Action: "stop",
			Type:   "container",
			Actor: APIActor{
				ID: "dfdf82bd3881",
				Attributes: map[string]string{
					"image": "base:latest",
				},
			},

			Status: "stop",
			ID:     "dfdf82bd3881",
			From:   "base:latest",

			Time: 1374067966,
		},
		{
			Action: "destroy",
			Type:   "container",
			Actor: APIActor{
				ID: "dfdf82bd3881",
				Attributes: map[string]string{
					"image": "base:latest",
				},
			},

			Status: "destroy",
			ID:     "dfdf82bd3881",
			From:   "base:latest",

			Time: 1374067970,
		},
		{
			Action:   "create",
			Type:     "container",
			Status:   "create",
			From:     "datanerd.us/siteeng/sample-app-go:latest",
			ID:       "a925eaf4084d5c3bcf337b2abb05f566ebb94276dff34f6effb00d8ecd380e16",
			Time:     1459133932,
			TimeNano: 1459133932961735842,
			Actor: APIActor{
				ID: "a925eaf4084d5c3bcf337b2abb05f566ebb94276dff34f6effb00d8ecd380e16",
				Attributes: map[string]string{
					"HAProxyMode":      "http",
					"HealthCheck":      "HttpGet",
					"HealthCheckArgs":  "http://127.0.0.1:39051/status/check",
					"ServicePort_8080": "17801",
					"image":            "datanerd.us/siteeng/sample-app-go:latest",
					"name":             "sample-app-client-go-69818c1223ddb5",
				},
			},
		},
	}

	client, err := buildClient(server.URL)
	if err != nil {
		t.Errorf("Failed to create client: %s", err)
	}
	client.SkipServerVersionCheck = true

	listener := make(chan *APIEvents, len(wantedEvents)+1)
	defer func() {
		if err = client.RemoveEventListener(listener); err != nil {
			t.Error(err)
		}
	}()

	err = client.AddEventListener(-1, listener)
	if err != nil {
		t.Errorf("Failed to add event listener: %s", err)
	}

	timeout := time.After(5 * time.Second)
	events := make([]APIEvents, 0, len(wantedEvents))

loop:
	for i := range wantedEvents {
		select {
		case msg, ok := <-listener:
			if !ok {
				break loop
			}
			events = append(events, *msg)
		case <-timeout:
			t.Fatalf("%s: timed out waiting on events after %d events", testName, i)
		}
	}
	cmpr := cmp.Comparer(func(e1, e2 APIEvents) bool {
		return e1.Action == e2.Action && e1.Actor.ID == e2.Actor.ID
	})
	if dff := cmp.Diff(events, wantedEvents, cmpr); dff != "" {
		t.Errorf("wrong events:\n%s", dff)
	}
}

func TestEventListenerReAdding(t *testing.T) {
	t.Parallel()
	endChan := make(chan bool)
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		<-endChan
	}))

	client, err := NewClient(server.URL)
	if err != nil {
		t.Errorf("Failed to create client: %s", err)
	}

	listener := make(chan *APIEvents, 10)
	if err := client.AddEventListener(-1, listener); err != nil {
		t.Errorf("Failed to add event listener: %s", err)
	}

	// Make sure EventHijack() is started with the current eventMonitoringState.
	time.Sleep(10 * time.Millisecond)

	if err := client.RemoveEventListener(listener); err != nil {
		t.Errorf("Failed to remove event listener: %s", err)
	}

	if err := client.AddEventListener(-1, listener); err != nil {
		t.Errorf("Failed to add event listener: %s", err)
	}

	endChan <- true

	// Give the goroutine of the first EventHijack() time to handle the EOF.
	time.Sleep(10 * time.Millisecond)
}
