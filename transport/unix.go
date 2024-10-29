package transport

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"path"
	"syscall"

	"github.com/dstreet/ubus"
	"github.com/google/uuid"
)

type UnixTransport struct {
	socket   string
	listener net.Listener
	msgChan  chan ubus.Message
	clients  map[string]net.Conn
	rootDir  string
}

const (
	UnixEventOnline  string = "$ubus.unix.online"
	UnixEventOffline string = "$ubus.unix.offline"
)

func NewUnixTransport(socketDir string) (*UnixTransport, error) {
	id := uuid.NewString()

	if err := os.Mkdir(socketDir, 0755); err != nil {
		if !errors.Is(err, os.ErrExist) {
			return nil, fmt.Errorf("failed to create socket directory: %w", err)
		}
	}

	entries, err := os.ReadDir(socketDir)
	if err != nil {
		return nil, fmt.Errorf("failed to ready socket directory: %w", err)
	}

	socket := path.Join(socketDir, fmt.Sprintf("%s.sock", id))
	ut := &UnixTransport{
		rootDir: socketDir,
		socket:  socket,
		clients: make(map[string]net.Conn, 0),
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		ut.setupClient(path.Join(socketDir, entry.Name()))
	}

	return ut, nil
}

func (t *UnixTransport) Dial() (net.Conn, error) {
	return net.Dial("unix", t.socket)
}

func (t *UnixTransport) Listen(ready chan struct{}) error {
	listener, err := net.Listen("unix", t.socket)
	if err != nil {
		return fmt.Errorf("failed to listen to unix socket: %w", err)
	}

	t.listener = listener
	ready <- struct{}{}

	t.Push(ubus.Message{
		Event: UnixEventOnline,
		Data:  t.socket,
	})

	for {
		conn, err := listener.Accept()
		if err != nil {
			return fmt.Errorf("failed to handle connection: %w", err)
		}

		go t.handleConn(conn)
	}
}

func (t *UnixTransport) HasClients() bool {
	return len(t.clients) > 0
}

func (t *UnixTransport) Close() error {
	hasClients := t.HasClients()

	if err := os.Remove(t.socket); err != nil {
		return fmt.Errorf("failed to remove socket: %w", err)
	}

	t.Push(ubus.Message{
		Event: UnixEventOffline,
		Data:  t.socket,
	})

	t.teardownClients()

	if err := t.listener.Close(); err != nil {
		return fmt.Errorf("failed to close listener: %w", err)
	}

	if !hasClients {
		if err := os.RemoveAll(t.rootDir); err != nil {
			return fmt.Errorf("failed to remove root directory: %w", err)
		}
	}

	return nil
}

func (t *UnixTransport) Subscribe(msgChan chan ubus.Message) {
	t.msgChan = msgChan
}

func (t *UnixTransport) Push(msg ubus.Message) {
	bb, err := json.Marshal(msg)
	if err != nil {
		fmt.Println("failed to marshall message")
	}

	for s, c := range t.clients {
		if _, err := c.Write(append(bb, '\n')); err != nil {
			if errors.Is(err, syscall.EPIPE) {
				fmt.Println("Client is unexpectedly closed...")
				t.teardownClient(s)
				continue
			}

			fmt.Println("Failed to write to client...")
		}
	}
}

func (t *UnixTransport) handleConn(conn net.Conn) {
	defer conn.Close()

	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		msg := ubus.Message{}
		if err := json.Unmarshal(scanner.Bytes(), &msg); err != nil {
			fmt.Println("failed to parse message: %w", err)
		}

		switch msg.Event {
		case UnixEventOnline:
			t.handleOnlineMessage(msg)
		case UnixEventOffline:
			t.handleOfflineMessage(msg)
		}

		if t.msgChan != nil {
			t.msgChan <- msg
		}
	}
}

func (t *UnixTransport) handleOnlineMessage(msg ubus.Message) {
	socket, ok := msg.Data.(string)
	if !ok {
		fmt.Println("Unable to process online message data")
	}

	t.setupClient(socket)
}

func (t *UnixTransport) handleOfflineMessage(msg ubus.Message) {
	socket, ok := msg.Data.(string)
	if !ok {
		fmt.Println("Unable to process offline message data")
	}

	t.teardownClient(socket)
}

func (t *UnixTransport) setupClient(socket string) {
	client, err := net.Dial("unix", socket)
	if err != nil {
		fmt.Printf("Failed to create client: %v\n", err)
		return
	}

	t.clients[socket] = client
}

func (t *UnixTransport) teardownClient(socket string) {
	if c, ok := t.clients[socket]; ok {
		if err := c.Close(); err != nil {
			fmt.Printf("Failed to close client: %v", err)
		}

		delete(t.clients, socket)
	}
}

func (t *UnixTransport) teardownClients() {
	for s, c := range t.clients {
		if err := c.Close(); err != nil {
			fmt.Printf("Failed to close client: %v", err)
		}

		delete(t.clients, s)
	}
}
