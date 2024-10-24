package transport_test

import (
	"os"
	"testing"

	"github.com/dstreet/ubus"
	"github.com/dstreet/ubus/transport"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUnixTransport(t *testing.T) {
	dir, err := os.MkdirTemp(os.TempDir(), "*")
	require.NoError(t, err)
	defer os.Remove(dir)

	t1, err := transport.NewUnixTransport(dir)
	require.NoError(t, err)
	defer t1.Close()
	t1Ready := make(chan struct{})
	go t1.Listen(t1Ready)

	<-t1Ready
	t1Chan := make(chan ubus.Message)
	t1.Subscribe(t1Chan)

	t2, err := transport.NewUnixTransport(dir)
	require.NoError(t, err)
	defer t2.Close()
	t2Ready := make(chan struct{})
	go t2.Listen(t2Ready)

	<-t2Ready
	t2Chan := make(chan ubus.Message)
	t2.Subscribe(t2Chan)

	var rm ubus.Message
	var sm ubus.Message

	rm = <-t1Chan
	assert.Equal(t, transport.UnixEventOnline, rm.Event)

	sm = ubus.Message{Event: "t2.test", Data: "one"}
	t2.Push(sm)
	rm = <-t1Chan
	assert.Equal(t, sm, rm)

	sm = ubus.Message{Event: "t1.test", Data: "one"}
	t1.Push(sm)
	rm = <-t2Chan
	assert.Equal(t, sm, rm)

	assert.NoError(t, t1.Close())
	rm = <-t2Chan
	assert.Equal(t, transport.UnixEventOffline, rm.Event)
}
