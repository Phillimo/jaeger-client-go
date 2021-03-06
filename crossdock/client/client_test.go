package client

import (
	"log"
	"net/url"
	"testing"

	"github.com/crossdock/crossdock-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/uber/jaeger-client-go"
	"github.com/uber/jaeger-client-go/crossdock/common"
	"github.com/uber/jaeger-client-go/crossdock/server"
)

func TestCrossdock(t *testing.T) {
	log.Println("Starting crossdock test")

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
	err = c.AsyncStart()
	require.NoError(t, err)
	defer c.Close()

	crossdock.Wait(t, s.URL(), 10)
	crossdock.Wait(t, c.URL(), 10)

	behaviors := []struct {
		name string
		axes map[string][]string
	}{
		{
			name: behaviorTrace,
			axes: map[string][]string{
				server1NameParam:      {common.DefaultServiceName},
				sampledParam:          {"true", "false"},
				server2NameParam:      {common.DefaultServiceName},
				server2TransportParam: {transportHTTP, transportTChannel, transportDummy},
				server3NameParam:      {common.DefaultServiceName},
				server3TransportParam: {transportHTTP, transportTChannel},
			},
		},
	}

	for _, bb := range behaviors {
		for _, entry := range crossdock.Combinations(bb.axes) {
			entryArgs := url.Values{}
			for k, v := range entry {
				entryArgs.Set(k, v)
			}
			// test via real HTTP call
			crossdock.Call(t, c.URL(), bb.name, entryArgs)
		}
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
