package ubus_test

import (
	"testing"

	"github.com/dstreet/ubus"
	"github.com/stretchr/testify/assert"
)

func TestBus(t *testing.T) {
	b := ubus.New()

	h1Calls := []ubus.Message{}
	h1 := b.On("test", func(msg ubus.Message) {
		h1Calls = append(h1Calls, msg)
	})

	h2Calls := []ubus.Message{}
	b.On("test", func(msg ubus.Message) {
		h2Calls = append(h2Calls, msg)
	})

	sent := []ubus.Message{
		{Event: "test", Data: "one", Headers: map[string]any{"header-one": "value"}},
		{Event: "test", Data: "two"},
		{Event: "test", Data: "three"},
	}

	<-b.Emit(sent[0])
	<-b.Emit(sent[1])
	assert.Equal(t, []ubus.Message{sent[0], sent[1]}, h1Calls)
	assert.Equal(t, []ubus.Message{sent[0], sent[1]}, h2Calls)

	h1.Off()

	<-b.Emit(ubus.Message{Event: "test", Data: "three"})
	assert.Equal(t, []ubus.Message{sent[0], sent[1]}, h1Calls)
	assert.Equal(t, []ubus.Message{sent[0], sent[1], sent[2]}, h2Calls)
}

func TestBus_Recover(t *testing.T) {
	b := ubus.New()

	b.On("test", func(msg ubus.Message) {
		panic("uh oh!")
	})

	assert.NotPanics(t, func() {
		<-b.Emit(ubus.Message{Event: "test"})
	})
}
