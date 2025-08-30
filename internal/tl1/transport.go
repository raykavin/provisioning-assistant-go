package tl1

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
	"time"
)

const (
	// Connection constants
	DefaultConnectionTimeout = 30 * time.Second
	ReadBufferSize           = 4096
	CommandTerminator        = ";"
	ConnectionCheckTimeout   = 500 * time.Millisecond
)

var (
	ErrNotConnected    = errors.New("not connected to server")
	ErrConnectionLost  = errors.New("connection lost")
	ErrReadTimeout     = errors.New("read timeout")
	ErrInvalidResponse = errors.New("invalid response format")
)

// TL1Transport represents a TL1 protocol transport layer
type TL1Transport struct {
	hostname string
	port     uint16
	conn     net.Conn
	mu       sync.RWMutex
	closed   bool
}

// NewTL1Transport creates a new TL1Transport instance and establishes connection
func NewTransport(hostname string, port uint16) (*TL1Transport, error) {
	if hostname == "" {
		return nil, errors.New("hostname cannot be empty")
	}
	if port == 0 {
		return nil, errors.New("port must be greater than 0")
	}

	tl1 := &TL1Transport{
		hostname: hostname,
		port:     port,
	}

	if err := tl1.connect(); err != nil {
		return nil, fmt.Errorf("failed to establish initial connection: %w", err)
	}

	return tl1, nil
}

// connect establishes a TCP connection to the TL1 server
func (t *TL1Transport) connect() error {
	address := net.JoinHostPort(t.hostname, fmt.Sprint(t.port))

	conn, err := net.DialTimeout("tcp", address, DefaultConnectionTimeout)
	if err != nil {
		return fmt.Errorf("failed to connect to %s: %w", address, err)
	}

	t.conn = conn
	t.closed = false
	return nil
}

// isConnectionAlive checks if the connection is still alive by attempting a non-blocking read
func (t *TL1Transport) isConnectionAlive() error {
	if t.conn == nil {
		return ErrNotConnected
	}

	// Set a short read deadline to check connection status
	if err := t.conn.SetReadDeadline(time.Now().Add(ConnectionCheckTimeout)); err != nil {
		return fmt.Errorf("failed to set read deadline: %w", err)
	}

	// Try to read one byte without blocking
	buffer := make([]byte, 1)
	_, err := t.conn.Read(buffer)

	// Reset deadline
	t.conn.SetReadDeadline(time.Time{})

	if err != nil {
		if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
			// Timeout is expected and means connection is alive
			return nil
		}
		return fmt.Errorf("connection check failed: %w", err)
	}

	return nil
}

// ensureConnection verifies connection health and reconnects if necessary
func (t *TL1Transport) ensureConnection() error {
	if t.closed {
		return ErrNotConnected
	}

	if err := t.isConnectionAlive(); err != nil {
		// If connection is dead, try to reconnect
		if !errors.Is(err, ErrNotConnected) {
			if reconnectErr := t.connect(); reconnectErr != nil {
				return fmt.Errorf("reconnection failed: %w", reconnectErr)
			}
		} else {
			return err
		}
	}

	return nil
}

// readResponse reads the complete response from the connection until terminator is found
func (t *TL1Transport) readResponse() (string, error) {
	if t.conn == nil {
		return "", ErrNotConnected
	}

	reader := bufio.NewReader(t.conn)
	var response strings.Builder
	buffer := make([]byte, ReadBufferSize)

	for {
		n, err := reader.Read(buffer)
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return "", fmt.Errorf("failed to read response: %w", err)
		}

		chunk := string(buffer[:n])
		response.WriteString(chunk)

		// Check if we've received the complete command (terminated by semicolon)
		if strings.HasSuffix(strings.TrimSpace(chunk), CommandTerminator) {
			break
		}
	}

	result := response.String()
	if result == "" {
		return "", ErrInvalidResponse
	}

	return result, nil
}

// Cmd sends a command to the TL1 server and returns the response
func (t *TL1Transport) Cmd(command string) (string, error) {
	if command == "" {
		return "", errors.New("command cannot be empty")
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	if t.closed {
		return "", ErrNotConnected
	}

	// Ensure we have a valid connection
	if err := t.ensureConnection(); err != nil {
		return "", fmt.Errorf("connection check failed: %w", err)
	}

	// Send the command
	if _, err := t.conn.Write([]byte(command)); err != nil {
		return "", fmt.Errorf("failed to send command: %w", err)
	}

	// Read and return the response
	response, err := t.readResponse()
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	return response, nil
}

// Send sends a command with context support for cancellation/timeout
func (t *TL1Transport) Send(ctx context.Context, command string) (string, error) {
	if command == "" {
		return "", errors.New("command cannot be empty")
	}

	// Create a channel to receive the result
	resultChan := make(chan struct {
		response string
		err      error
	}, 1)

	// Execute command in goroutine
	go func() {
		response, err := t.Cmd(command)
		resultChan <- struct {
			response string
			err      error
		}{response, err}
	}()

	// Wait for either completion or context cancellation
	select {
	case result := <-resultChan:
		return result.response, result.err
	case <-ctx.Done():
		return "", fmt.Errorf("command cancelled: %w", ctx.Err())
	}
}

// Reconnect forces a reconnection to the TL1 server
func (t *TL1Transport) Reconnect() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.conn != nil {
		t.conn.Close()
	}

	return t.connect()
}

// Close closes the connection to the TL1 server
func (t *TL1Transport) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.closed = true

	if t.conn != nil {
		err := t.conn.Close()
		t.conn = nil
		return err
	}

	return nil
}

// IsConnected returns true if the transport is connected to the server
func (t *TL1Transport) IsConnected() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if t.closed || t.conn == nil {
		return false
	}

	return t.isConnectionAlive() == nil
}

// GetAddress returns the connection address
func (t *TL1Transport) GetAddress() string {
	return net.JoinHostPort(t.hostname, fmt.Sprint(t.port))
}
