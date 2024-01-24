package belphegor

import (
	"belphegor/pkg/clipboard"
	"errors"
	"github.com/rs/zerolog/log"
	"io"
	"net"
)

// NodeDataReceiver responsible for receiving data from the node.
type NodeDataReceiver struct {
	node      *Node
	conn      net.Conn
	cm        clipboard.Manager
	localChan Channel
}

// NewNodeDataReceiver creates a new instance NodeDataReceiver.
func NewNodeDataReceiver(node *Node, conn net.Conn, cp clipboard.Manager, channel Channel) *NodeDataReceiver {
	return &NodeDataReceiver{
		node:      node,
		conn:      conn,
		cm:        cp,
		localChan: channel,
	}
}

// Receive starts receiving data from the node.
func (ndr *NodeDataReceiver) Receive() {
	addr := castAddrPortFromConn(ndr.conn)
	defer func() {
		log.Info().Msgf("node %s disconnected", addr)
		ndr.node.storage.Delete(addr)
	}()

	for {
		msg, err := ndr.receiveMessage()
		if err != nil {
			ndr.handleReceiveError(err)
			break
		}

		ndr.node.SetLastMessage(msg)
		_ = ndr.cm.Set(msg.Data.Raw)
		ndr.localChan.Set(msg.Data.Raw)

		log.Debug().Msgf(
			"received %s from %s by hash %x",
			msg.Header.ID,
			addr,
			shortHash(msg.Data.Hash),
		)

		ndr.node.Broadcast(msg, addr)
	}
}

// receiveMessage receives a message from the node.
func (ndr *NodeDataReceiver) receiveMessage() (*Message, error) {
	msg := AcquireMessage([]byte{})
	err := decodeMessage(ndr.conn, msg)
	if err != nil {
		return nil, err
	}
	return msg, nil
}

// handleReceiveError handles errors when receiving data.
func (ndr *NodeDataReceiver) handleReceiveError(err error) {
	if errors.Is(err, io.EOF) {
		log.Trace().Msg("connection closed by EOF (similar to invalid message)")
		return
	}
	var opErr *net.OpError
	if errors.As(err, &opErr) {
		log.Trace().Err(opErr).Msg("connection closed by OpError")
	}
}
