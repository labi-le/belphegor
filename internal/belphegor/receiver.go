package belphegor

import (
	"belphegor/pkg/clipboard"
	"errors"
	"github.com/rs/zerolog/log"
	"io"
	"net"
	"os"
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
	remoteIP := Address(ndr.conn.RemoteAddr().(*net.TCPAddr).IP.String())
	defer func() {
		log.Info().Msgf("node %s disconnected", remoteIP)
		ndr.node.storage.Delete(remoteIP)
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

		//go debug(*msg)

		log.Debug().Msgf("received %s from %s by hash %x", msg.Header.ID, remoteIP, shortHash(msg.Data.Hash))

		ndr.node.Broadcast(msg, remoteIP)
	}
}

func debug(message *Message) {
	// get current dir
	dir, _ := os.Getwd()
	fp := dir + "/debug/" + message.Header.ID.String() + ".png"
	log.Trace().Msgf("writing debug file to %s", fp)
	err := os.WriteFile(fp, message.Data.Raw, 0644)
	if err != nil {
		log.Error().Err(err).Msg("failed to write debug file")
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
