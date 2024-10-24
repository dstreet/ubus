package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/dstreet/ubus"
	"github.com/dstreet/ubus/transport"
)

func main() {
	name := os.Args[1]

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGKILL)
	defer cancel()

	transport, err := transport.NewUnixTransport("/tmp/ubus")
	if err != nil {
		log.Fatalf("failed to create transport: %v", err)
	}

	ready := make(chan struct{})
	go transport.Listen(ready)

	<-ready

	b := ubus.New(ubus.WithTransport(transport))

	b.On("ping", func(msg ubus.Message) {
		d := msg.Data.(string)
		fmt.Printf("received ping with: %s\n", d)
	})

	go func() {
		for {
			time.Sleep(2 * time.Second)
			b.Emit(ubus.Message{Event: "ping", Data: name})
		}
	}()

	<-ctx.Done()
	transport.Close()
}
