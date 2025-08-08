// Structs P2PConnection and Management aim to add connection-oriented management
// capabilities to the knx-go library. See KNX Standard 03_05_02 Management Procedures.

package knx

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/LB-00/knx-go/knx/cemi"
)

// P2PConnection represents a point-to-point connection to a bus device.
type P2PConnection struct {
	tunnel     *Tunnel             // Underlying tunneling connection
	inbound    chan cemi.Message   // Filtered messages for this connection
	targetAddr cemi.IndividualAddr // Individual Address of the target bus device
	seqNumber  uint8               // Sequence number (4 bits)
	rateLimit  uint                // Rate limit for sending messages
	lastSend   time.Time           // Time of last sent message
	connected  bool                // Whether the connection is established
	done       chan struct{}
	wait       sync.WaitGroup
	mu         sync.Mutex
}

// NewP2PConnection creates a new point-to-point connection to a device.
func NewP2PConnection(tunnel *Tunnel, addr cemi.IndividualAddr) (*P2PConnection, error) {
	// Initialize the point-to-point connection structure.
	conn := &P2PConnection{
		tunnel:     tunnel,
		targetAddr: addr,
		seqNumber:  15, // Start with the maximum so the first increment will be 0.
		rateLimit:  20,
		lastSend:   time.Now().Add(-time.Second),
		done:       make(chan struct{}),
		inbound:    make(chan cemi.Message, 10),
	}

	// Attempt to connect to the device.
	err := conn.requestConn()
	if err != nil {
		return nil, err
	}

	// Start processing inbound messages.
	conn.wait.Add(1)
	go conn.serve()

	return conn, nil
}

// Send sends a cEMI telegram over the point-to-point connection to the device
// and waits for a response matching the expected command.
func (conn *P2PConnection) Send(req cemi.Message, exp cemi.APCI, t time.Duration) (cemi.Message, error) {
	if !conn.connected {
		return nil, errors.New("not connected to device")
	}

	// Set the sequence number in the request.
	seq := conn.nextSeqNum()
	err := conn.setSeqNum(req, seq)
	if err != nil {
		return nil, err
	}

	conn.applyRateLimit()

	// Send the cEMI frame through the tunnel.
	err = conn.tunnel.Send(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	// TODO: Retry once?
	err = conn.awaitAck()
	if err != nil {
		return nil, err
	}

	// Wait for a response from the device.
	timeout := time.After(t) // 6 * time.Second

	for {
		select {
		// The response has timed out.
		case <-timeout:
			return nil, errors.New("response timed out")

		// The connection has been closed.
		case <-conn.done:
			return nil, errors.New("connection was closed")

		// A response has been received.
		case res := <-conn.inbound:
			// Messages other than Indication primitives can be ignored.
			ind, ok := res.(*cemi.LDataInd)
			if !ok {
				continue
			}

			app, ok := ind.LData.Data.(*cemi.AppData)
			if !ok {
				continue
			}
			if app.Command != exp {
				continue
			}

			conn.applyRateLimit()

			// Send an Ack to the device.
			req := cemi.NewAck(conn.tunnel.SourceAddr(), ind.LData.Source, app.SeqNumber)
			err := conn.tunnel.Send(req)
			if err != nil {
				return nil, fmt.Errorf("failed to send ACK: %w", err)
			}

			return ind, nil
		}
	}
}

// Disconnect closes the point-to-point connection to the device.
func (conn *P2PConnection) Disconnect() error {
	conn.mu.Lock()
	if !conn.connected {
		conn.mu.Unlock()
		return nil
	}
	conn.mu.Unlock()

	conn.applyRateLimit()

	// Create and send a T_DISCONNECT request.
	req := cemi.NewDiscReq(conn.tunnel.SourceAddr(), conn.targetAddr)
	err := conn.tunnel.Send(req)

	// TODO: Wait for L_Data.con with T_DISCONNECT?

	// Mark as disconnected regardless of the send success.
	conn.mu.Lock()
	conn.connected = false
	conn.mu.Unlock()

	// Signal to stop the processor goroutine.
	select {
	case <-conn.done:
		// Already closed.
	default:
		close(conn.done)
	}

	// Wait for the processor goroutine to finish.
	conn.wait.Wait()

	return err
}

// Inbound returns the channel for receiving messages from the connection.
func (conn *P2PConnection) Inbound() <-chan cemi.Message {
	return conn.inbound
}

// Connect establishes the connection to the device.
func (conn *P2PConnection) requestConn() error {
	conn.mu.Lock()
	if conn.connected {
		conn.mu.Unlock()
		return errors.New("already connected to device")
	}
	conn.mu.Unlock()

	// Create and send a T_CONNECT request.
	req := cemi.NewConnReq(conn.tunnel.SourceAddr(), conn.targetAddr)
	err := conn.tunnel.Send(req)
	if err != nil {
		return err
	}

	// Setup timeout.
	timeout := time.After(conn.tunnel.config.ResponseTimeout)

	// Cycle until a confirmation is received.
	for {
		select {
		// Timeout reached.
		case <-timeout:
			return errResponseTimeout

		// A message has been received or the channel has been closed.
		case msg, open := <-conn.tunnel.inbound:
			if !open {
				return errors.New("tunnel was closed before a connection could be established")
			}

			// We're only interested in a L_Data.con wrapping a T_CONNECT.
			if con, ok := msg.(*cemi.LDataCon); ok {
				if _, ok := con.LData.Data.(*cemi.ControlConn); !ok {
					continue
				}

				// The connection was established successfully.
				conn.connected = true
				return nil
			}
		}
	}
}

// serve processes messages from the tunnels inbound channel.
func (conn *P2PConnection) serve() {
	defer conn.wait.Done()
	defer close(conn.inbound)

	for {
		select {
		// Connection is being closed.
		case <-conn.done:
			return

		// A message has been received or the tunnel is closed.
		case msg, open := <-conn.tunnel.Inbound():
			if !open {
				conn.handleTunnelClosed()
				return
			}

			if conn.handleDisconnect(msg) {
				continue
			}

			select {
			case conn.inbound <- msg:
				// Successfully forwarded the message.

			default:
				// Inbound channel is full, log a warning.
				fmt.Printf("Warning: P2PConn inbound channel for %s is full, discarding message: %T\n", conn.targetAddr, msg)
			}
		}
	}
}

// handleDisconnect processes a disconnect requests received from the tunnel.
func (conn *P2PConnection) handleDisconnect(msg cemi.Message) bool {
	// We only care about L_Data.ind messages.
	ind, ok := msg.(*cemi.LDataInd)
	if !ok {
		return false
	}

	// Ensure the message is for this connection.
	if ind.LData.Destination != uint16(conn.targetAddr) || ind.LData.Source != conn.tunnel.SourceAddr() {
		return false
	}

	// Check if the message is a disconnect request.
	if _, ok := ind.LData.Data.(*cemi.ControlDisc); ok {
		if !conn.connected {
			return true
		}

		conn.Disconnect()

		// Signal disconnection.
		select {
		case <-conn.done:
			// Already closed.
		default:
			close(conn.done)
		}

		return true
	}

	return false
}

// handleTunnelClosed handles the case when the tunnel's inbound channel is closed.
func (conn *P2PConnection) handleTunnelClosed() {

	// Mark the connection as disconnected.
	conn.mu.Lock()
	conn.connected = false
	conn.mu.Unlock()

	// Signal that the connection is closed.
	select {
	case <-conn.done:
		// Already closed.
	default:
		close(conn.done)
	}
}

// nextSeqNum increments the sequence number for the connection.
func (conn *P2PConnection) nextSeqNum() uint8 {
	conn.mu.Lock()
	defer conn.mu.Unlock()

	// Enforce the 4-bit sequence number limit.
	conn.seqNumber = (conn.seqNumber + 1) % 16
	seq := conn.seqNumber
	return seq
}

// awaitAck waits for a T_Ack from the device after sending a request.
func (conn *P2PConnection) awaitAck() error {
	timeout := time.After(conn.tunnel.config.ResponseTimeout)

	for {
		select {
		// The ACK has timed out.
		case <-timeout:
			return errors.New("timed out while waiting for ACK")

		// The connection has been closed.
		case <-conn.done:
			return errors.New("connection was closed")

		// A response has been received.
		case res := <-conn.inbound:
			// The Ack must be encapsulated in an indication primitive.
			ind, ok := res.(*cemi.LDataInd)
			if !ok {
				continue
			}

			ack, ok := ind.LData.Data.(*cemi.ControlAck)
			if !ok {
				continue
			}

			if ack.SeqNumber != conn.seqNumber {
				return fmt.Errorf(
					"ack sequence number %d must match request sequence number %d",
					ack.SeqNumber, conn.seqNumber,
				)
			}

			return nil
		}
	}
}

// setSeqNum sets the sequence number in the request, effectively turning it into a
// T_DATA_CONNECTED telegram.
func (conn *P2PConnection) setSeqNum(req cemi.Message, seq uint8) error {

	// Set the sequence number in the request.
	ind, ok := req.(*cemi.LDataReq)
	if !ok {
		return fmt.Errorf("expected LDataReq, got %T", req)
	}

	app, ok := ind.LData.Data.(*cemi.AppData)
	if !ok {
		return fmt.Errorf("expected AppData, got %T", ind.LData.Data)
	}

	app.Numbered = true
	app.SeqNumber = seq

	return nil
}

// applyRateLimit ensures the connections rate limit is respected.
func (conn *P2PConnection) applyRateLimit() {
	conn.mu.Lock()
	interval := time.Second / time.Duration(conn.rateLimit)
	elapsed := time.Since(conn.lastSend)
	if elapsed < interval {
		wait := interval - elapsed
		conn.mu.Unlock()
		time.Sleep(wait)
		conn.mu.Lock()
	}
	conn.lastSend = time.Now()
	conn.mu.Unlock()
}

// Management handles point-to-point connections to individual devices.
type Management struct {
	tunnel      *Tunnel
	connections map[cemi.IndividualAddr]*P2PConnection
	mu          sync.Mutex
	done        chan struct{}
}

// NewManagement creates a new Management instance with the given tunnel.
func NewManagement(tunnel *Tunnel) *Management {
	return &Management{
		tunnel:      tunnel,
		connections: make(map[cemi.IndividualAddr]*P2PConnection),
		mu:          sync.Mutex{},
		done:        make(chan struct{}),
	}
}

// Close stops all management operations and closes all connections.
func (m *Management) Close() {
	// Signal that the management is closing.
	close(m.done)

	// Close all connections.
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, conn := range m.connections {
		conn.Disconnect()
	}
}

// Connect establishes a new point-to-point connection to a device.
func (m *Management) Connect(addr cemi.IndividualAddr) (*P2PConnection, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Return the connection if it already exists.
	conn, exists := m.connections[addr]
	if exists {
		if !conn.connected {
			delete(m.connections, addr)
		} else {
			return conn, nil
		}
	}

	// Create a new connection.
	conn, err := NewP2PConnection(m.tunnel, addr)
	if err != nil {
		return nil, err
	}

	// Store the connection.
	m.connections[addr] = conn

	return conn, nil
}

// Disconnect closes the point-to-point connection to a device if it exists.
func (m *Management) Disconnect(addr cemi.IndividualAddr) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	conn, exists := m.connections[addr]
	if !exists {
		return fmt.Errorf("connection not found")
	}

	err := conn.Disconnect()
	if err != nil {
		return err
	}

	delete(m.connections, addr)
	return nil
}

// Connection returns an existing point-to-point connection if it exists,
// or nil if it does not.
func (m *Management) GetConnection(addr cemi.IndividualAddr) *P2PConnection {
	m.mu.Lock()
	defer m.mu.Unlock()

	conn, exists := m.connections[addr]
	if !exists {
		return nil
	}

	return conn
}
