package node

import (
	"context"
	"errors"
	"testing"

	"github.com/labi-le/belphegor/internal/transport"
)

type mockTransport struct{}

func (m *mockTransport) Listen(ctx context.Context, addr string) (transport.Listener, error) {
	return nil, nil
}
func (m *mockTransport) Dial(ctx context.Context, addr string) (transport.Connection, error) {
	return nil, errors.New("should not be called")
}

func TestConnectTo_MaxPeers(t *testing.T) {
	tr := &mockTransport{}
	opts := Options{MaxPeers: 0} // limit is 0
	peers := &Storage{}
	n := New(tr, nil, peers, nil, opts)

	ctx := context.Background()
	err := n.ConnectTo(ctx, "localhost:1234")
	if !errors.Is(err, ErrMaxPeersReached) {
		t.Errorf("expected ErrMaxPeersReached, got %v", err)
	}
}
