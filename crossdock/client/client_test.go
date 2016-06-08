package client

import (
	"fmt"
	"net/url"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/uber/jaeger-client-go"
	"github.com/uber/jaeger-client-go/crossdock/common"
	"github.com/uber/jaeger-client-go/crossdock/server"
	"github.com/uber/jaeger-client-go/utils"
)

func TestClient(t *testing.T) {
	tracer, tCloser := jaeger.NewTracer(
		"crossdock",
		jaeger.NewConstSampler(false),
		jaeger.NewNullReporter())
	defer tCloser.Close()

	s := &server.Server{
		HostPortHTTP:     "127.0.0.1:0",
		HostPortTChannel: "127.0.0.1:0",
		Tracer:           tracer,
	}
	err := s.Start()
	require.NoError(t, err)
	defer s.Close()

	c := &Client{
		ClientHostPort:     "127.0.0.1:0",
		ServerPortHTTP:     s.GetPortHTTP(),
		ServerPortTChannel: s.GetPortTChannel(),
		hostMapper:         func(server string) string { return "localhost" },
	}
	err = c.Listen()
	require.NoError(t, err)

	var started sync.WaitGroup
	started.Add(1)
	go func() {
		started.Done()
		c.Serve()
	}()
	started.Wait()
	defer c.Close()

	exec(t, c, map[string]string{
		behaviorParam:         "trace",
		sampledParam:          "true",
		server1NameParam:      common.DefaultServiceName,
		server2NameParam:      common.DefaultServiceName,
		server2ClientParam:    "any",
		server2TransportParam: "tchannel",
		server3NameParam:      common.DefaultServiceName,
		server3ClientParam:    "any",
		server3TransportParam: "http",
	})
}

func exec(t *testing.T, c *Client, params map[string]string) {
	values := url.Values{}
	for k, v := range params {
		values.Add(k, v)
	}
	url := c.URL() + "/?" + values.Encode()
	entries := []map[string]string{}
	fmt.Printf("Executing %s\n", url)
	err := utils.GetJSON(url, &entries)
	require.NoError(t, err)
	assert.Equal(t, 1, len(entries), "expecting one success entry")
	fmt.Printf("Output: %+v\n", entries)
	for _, e := range entries {
		assert.Equal(t, "passed", e["status"], e["output"])
	}
}

func TestHostMapper(t *testing.T) {
	c := &Client{
		ClientHostPort:     "127.0.0.1:0",
		ServerPortHTTP:     "8080",
		ServerPortTChannel: "8081",
	}
	assert.Equal(t, "go", c.mapServiceToHost("go"))
	c.hostMapper = func(server string) string { return "localhost" }
	assert.Equal(t, "localhost", c.mapServiceToHost("go"))
}
