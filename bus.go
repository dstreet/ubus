package ubus

import (
	"fmt"
	"sync"

	"github.com/google/uuid"
)

type Bus struct {
	handlers  map[string]*Handler
	transport MessageTransport
	matcher   EventMatcher
}

type handlerFn func(msg Message)

type MessageTransport interface {
	Subscribe(chan Message)
	Push(Message)
	Close() error
}

type EventMatcher interface {
	Matches(expected string, actual string) bool
}

const matchTransportEventName = "$ubus.transport"

func New(opts ...Option) *Bus {
	bus := &Bus{
		handlers: make(map[string]*Handler, 0),
		matcher:  &ExactMatchMatcher{},
	}

	for _, opt := range opts {
		opt(bus)
	}

	if bus.transport != nil {
		msgs := make(chan Message)
		bus.transport.Subscribe(msgs)

		go func() {
			for msg := range msgs {
				bus.emit(msg, true)
			}
		}()

		bus.On(matchTransportEventName, bus.transport.Push)
	}

	return bus
}

func (b *Bus) On(event string, fn handlerFn) *Handler {
	id, _ := uuid.NewRandom()
	idStr := id.String()

	h := &Handler{
		event: event,
		fn:    fn,
		remove: func() {
			delete(b.handlers, idStr)
		},
	}

	b.handlers[idStr] = h
	return h
}

func (b *Bus) Emit(msg Message) (done chan struct{}) {
	return b.emit(msg, false)
}

func (b *Bus) Close() error {
	if b.transport == nil {
		return nil
	}

	return b.transport.Close()
}

func (b *Bus) emit(msg Message, skipTransport bool) (done chan struct{}) {
	wg := sync.WaitGroup{}
	done = make(chan struct{})

	for _, h := range b.handlers {
		if (h.event == matchTransportEventName && !skipTransport) ||
			b.matcher.Matches(h.event, msg.Event) {
			wg.Add(1)
			go func() {
				defer func() {
					if r := recover(); r != nil {
						fmt.Printf("Handler for event %q panicked:\n%v\n", msg.Event, r)
					}

					wg.Done()
				}()

				h.fn(msg)
			}()
		}
	}

	go func() {
		wg.Wait()
		done <- struct{}{}
	}()

	return
}

type Handler struct {
	event  string
	fn     handlerFn
	remove func()
}

func (h *Handler) Off() {
	h.remove()
}

type ExactMatchMatcher struct{}

func (m *ExactMatchMatcher) Matches(expected string, actual string) bool {
	return expected == actual
}
