package node_test

import (
	"testing"
	"time"

	"github.com/labi-le/belphegor/internal/node"
	"github.com/labi-le/belphegor/internal/types/domain"
	"github.com/labi-le/belphegor/pkg/clipboard/eventful"
	"github.com/labi-le/belphegor/pkg/network"
	"github.com/rs/zerolog"
)

func TestTransport_String(t *testing.T) {
	tests := []struct {
		name     string
		t        node.Transport
		expected string
	}{
		{
			name:     "TCP",
			t:        node.TransportTCP,
			expected: node.TransportTCP.String(),
		},
		{
			name:     "QUIC",
			t:        node.TransportQUIC,
			expected: node.TransportQUIC.String(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.t.String(); got != tt.expected {
				t.Errorf("Transport.String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestTransport_Set(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
		want    node.Transport
	}{
		{
			name:    "valid TCP",
			input:   "tcp",
			wantErr: false,
			want:    node.TransportTCP,
		},
		{
			name:    "valid QUIC",
			input:   "quic",
			wantErr: false,
			want:    node.TransportQUIC,
		},
		{
			name:    "invalid transport",
			input:   "udp",
			wantErr: true,
			want:    node.Transport("udp"),
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: true,
			want:    node.Transport(""),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var tr node.Transport
			err := tr.Set(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Transport.Set() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && tr != tt.want {
				t.Errorf("Transport.Set() = %v, want %v", tr, tt.want)
			}
		})
	}
}

func TestTransport_Type(t *testing.T) {
	var tr node.Transport
	if got := tr.Type(); got != "string" {
		t.Errorf("Transport.Type() = %v, want string", got)
	}
}

func TestTransport_Valid(t *testing.T) {
	tests := []struct {
		name  string
		t     node.Transport
		valid bool
	}{
		{
			name:  "TCP valid",
			t:     node.TransportTCP,
			valid: true,
		},
		{
			name:  "QUIC valid",
			t:     node.TransportQUIC,
			valid: true,
		},
		{
			name:  "invalid",
			t:     node.Transport("udp"),
			valid: false,
		},
		{
			name:  "empty",
			t:     node.Transport(""),
			valid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var tr node.Transport
			tr = tt.t
			isValid := tr == node.TransportTCP || tr == node.TransportQUIC
			if isValid != tt.valid {
				t.Errorf("Transport valid check = %v, want %v", isValid, tt.valid)
			}
		})
	}
}

func TestOptions_MarshalZerologObject(t *testing.T) {
	logger := zerolog.New(nil)
	evt := logger.Info()

	opts := node.Options{
		ListenPort: 8080,
		Transport:  node.TransportTCP,
		KeepAlive:  time.Minute,
		Deadline: network.Deadline{
			Read:  30 * time.Second,
			Write: 30 * time.Second,
		},
		Discovering: node.DiscoverOptions{
			Enable:   true,
			Delay:    10 * time.Second,
			MaxPeers: 5,
		},
		Metadata: domain.Device{
			ID:   12345,
			Name: "test-device",
			Arch: "amd64",
		},
		MaxPeers: 10,
		Secret:   "secret123",
		Clip: eventful.Options{
			AllowCopyFiles:    true,
			MaxClipboardFiles: 10,
			MaxFileSize:       1024,
		},
	}

	opts.MarshalZerologObject(evt)
}

func TestOptions_Validated(t *testing.T) {
	metadata := domain.Device{
		ID:   12345,
		Name: "test",
		Arch: "amd64",
	}

	deadline := network.Deadline{Read: time.Hour, Write: time.Minute}

	clipOpts := eventful.Options{MaxClipboardFiles: 20}

	tests := []struct {
		name  string
		opts  node.Options
		check func(*testing.T, node.Options)
	}{
		{
			name: "valid port",
			opts: node.Options{ListenPort: 8080},
			check: func(t *testing.T, o node.Options) {
				if o.ListenPort != 8080 {
					t.Errorf("ListenPort = %v, want 8080", o.ListenPort)
				}
			},
		},
		{
			name: "invalid port zero",
			opts: node.Options{ListenPort: 0},
			check: func(t *testing.T, o node.Options) {
				if o.ListenPort <= 0 {
					t.Error("ListenPort should be set to default")
				}
			},
		},
		{
			name: "invalid port negative",
			opts: node.Options{ListenPort: -1},
			check: func(t *testing.T, o node.Options) {
				if o.ListenPort <= 0 {
					t.Error("ListenPort should be set to default")
				}
			},
		},
		{
			name: "invalid port too high",
			opts: node.Options{ListenPort: 70000},
			check: func(t *testing.T, o node.Options) {
				if o.ListenPort <= 0 || o.ListenPort > 65535 {
					t.Error("ListenPort should be set to default")
				}
			},
		},
		{
			name: "valid transport TCP",
			opts: node.Options{Transport: node.TransportTCP},
			check: func(t *testing.T, o node.Options) {
				if o.Transport != node.TransportTCP {
					t.Errorf("Transport = %v, want tcp", o.Transport)
				}
			},
		},
		{
			name: "valid transport QUIC",
			opts: node.Options{Transport: node.TransportQUIC},
			check: func(t *testing.T, o node.Options) {
				if o.Transport != node.TransportQUIC {
					t.Errorf("Transport = %v, want quic", o.Transport)
				}
			},
		},
		{
			name: "invalid transport",
			opts: node.Options{Transport: node.Transport("udp")},
			check: func(t *testing.T, o node.Options) {
				if o.Transport != node.TransportQUIC {
					t.Error("Transport should be set to default")
				}
			},
		},
		{
			name: "empty transport",
			opts: node.Options{Transport: node.Transport("")},
			check: func(t *testing.T, o node.Options) {
				if o.Transport != node.TransportQUIC {
					t.Error("Transport should be set to default")
				}
			},
		},
		{
			name: "valid KeepAlive",
			opts: node.Options{KeepAlive: time.Hour},
			check: func(t *testing.T, o node.Options) {
				if o.KeepAlive != time.Hour {
					t.Errorf("KeepAlive = %v, want hour", o.KeepAlive)
				}
			},
		},
		{
			name: "invalid KeepAlive zero",
			opts: node.Options{KeepAlive: 0},
			check: func(t *testing.T, o node.Options) {
				if o.KeepAlive <= 0 {
					t.Error("KeepAlive should be set to default")
				}
			},
		},
		{
			name: "invalid KeepAlive negative",
			opts: node.Options{KeepAlive: -1},
			check: func(t *testing.T, o node.Options) {
				if o.KeepAlive <= 0 {
					t.Error("KeepAlive should be set to default")
				}
			},
		},
		{
			name: "valid Deadline.Read",
			opts: node.Options{Deadline: deadline},
			check: func(t *testing.T, o node.Options) {
				if o.Deadline.Read != time.Hour {
					t.Errorf("Deadline.Read = %v, want hour", o.Deadline.Read)
				}
			},
		},
		{
			name: "invalid Deadline.Read zero",
			opts: node.Options{Deadline: network.Deadline{Read: 0, Write: time.Minute}},
			check: func(t *testing.T, o node.Options) {
				if o.Deadline.Read <= 0 {
					t.Error("Deadline.Read should be set to default")
				}
			},
		},
		{
			name: "invalid Deadline.Write zero",
			opts: node.Options{Deadline: network.Deadline{Read: time.Minute, Write: 0}},
			check: func(t *testing.T, o node.Options) {
				if o.Deadline.Write <= 0 {
					t.Error("Deadline.Write should be set to default")
				}
			},
		},
		{
			name: "valid Discovering.MaxPeers",
			opts: node.Options{Discovering: node.DiscoverOptions{MaxPeers: 20}},
			check: func(t *testing.T, o node.Options) {
				if o.Discovering.MaxPeers != 20 {
					t.Errorf("Discovering.MaxPeers = %v, want 20", o.Discovering.MaxPeers)
				}
			},
		},
		{
			name: "invalid Discovering.MaxPeers zero",
			opts: node.Options{Discovering: node.DiscoverOptions{MaxPeers: 0}},
			check: func(t *testing.T, o node.Options) {
				if o.Discovering.MaxPeers <= 0 {
					t.Error("Discovering.MaxPeers should be set to default")
				}
			},
		},
		{
			name: "valid Discovering.Delay",
			opts: node.Options{Discovering: node.DiscoverOptions{Delay: time.Hour}},
			check: func(t *testing.T, o node.Options) {
				if o.Discovering.Delay != time.Hour {
					t.Errorf("Discovering.Delay = %v, want hour", o.Discovering.Delay)
				}
			},
		},
		{
			name: "invalid Discovering.Delay zero",
			opts: node.Options{Discovering: node.DiscoverOptions{Delay: 0}},
			check: func(t *testing.T, o node.Options) {
				if o.Discovering.Delay <= 0 {
					t.Error("Discovering.Delay should be set to default")
				}
			},
		},
		{
			name: "valid Metadata",
			opts: node.Options{Metadata: metadata},
			check: func(t *testing.T, o node.Options) {
				if o.Metadata != metadata {
					t.Error("Metadata should be preserved")
				}
			},
		},
		{
			name: "zero Metadata",
			opts: node.Options{Metadata: domain.Device{}},
			check: func(t *testing.T, o node.Options) {
				if o.Metadata == (domain.Device{}) {
					t.Error("Metadata should be set to default")
				}
			},
		},
		{
			name: "valid MaxPeers",
			opts: node.Options{MaxPeers: 50},
			check: func(t *testing.T, o node.Options) {
				if o.MaxPeers != 50 {
					t.Errorf("MaxPeers = %v, want 50", o.MaxPeers)
				}
			},
		},
		{
			name: "invalid MaxPeers zero",
			opts: node.Options{MaxPeers: 0},
			check: func(t *testing.T, o node.Options) {
				if o.MaxPeers <= 0 {
					t.Error("MaxPeers should be set to default")
				}
			},
		},
		{
			name: "valid Clip.MaxClipboardFiles",
			opts: node.Options{Clip: clipOpts},
			check: func(t *testing.T, o node.Options) {
				if o.Clip.MaxClipboardFiles != 20 {
					t.Errorf("Clip.MaxClipboardFiles = %v, want 20", o.Clip.MaxClipboardFiles)
				}
			},
		},
		{
			name: "invalid Clip.MaxClipboardFiles zero",
			opts: node.Options{Clip: eventful.Options{MaxClipboardFiles: 0}},
			check: func(t *testing.T, o node.Options) {
				if o.Clip.MaxClipboardFiles <= 0 {
					t.Error("Clip.MaxClipboardFiles should be set to default")
				}
			},
		},
		{
			name: "all defaults",
			opts: node.Options{},
			check: func(t *testing.T, o node.Options) {
				if o.ListenPort <= 0 {
					t.Error("ListenPort should be set")
				}
				if o.Transport != node.TransportQUIC {
					t.Error("Transport should be set")
				}
				if o.KeepAlive <= 0 {
					t.Error("KeepAlive should be set")
				}
				if o.Deadline.Read <= 0 || o.Deadline.Write <= 0 {
					t.Error("Deadline should be set")
				}
				if o.Notifier == nil {
					t.Error("Notifier should be set")
				}
				if o.Discovering.MaxPeers <= 0 {
					t.Error("Discovering.MaxPeers should be set")
				}
				if o.Discovering.Delay <= 0 {
					t.Error("Discovering.Delay should be set")
				}
				if o.Metadata == (domain.Device{}) {
					t.Error("Metadata should be set")
				}
				if o.MaxPeers <= 0 {
					t.Error("MaxPeers should be set")
				}
				if o.Clip.MaxClipboardFiles <= 0 {
					t.Error("Clip.MaxClipboardFiles should be set")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validated := tt.opts.Validated()
			tt.check(t, validated)
		})
	}
}

func TestDefaultOptions(t *testing.T) {
	opts := node.DefaultOptions()
	t.Logf("Transport value: %q", opts.Transport)
	t.Logf("Expected TransportQUIC: %q", node.TransportQUIC)

	if opts.ListenPort <= 0 {
		t.Error("ListenPort should be set")
	}
	if opts.Transport != node.TransportQUIC {
		t.Errorf("Transport should be QUIC, got %v", opts.Transport)
	}
	if opts.KeepAlive <= 0 {
		t.Error("KeepAlive should be set")
	}
	if opts.Deadline.Read <= 0 || opts.Deadline.Write <= 0 {
		t.Error("Deadline should be set")
	}
	if opts.Notifier == nil {
		t.Error("Notifier should be set")
	}
	if !opts.Discovering.Enable {
		t.Error("Discovering.Enable should be true")
	}
	if opts.Discovering.Delay <= 0 {
		t.Error("Discovering.Delay should be set")
	}
	if opts.Discovering.MaxPeers <= 0 {
		t.Error("Discovering.MaxPeers should be set")
	}
	if opts.Metadata == (domain.Device{}) {
		t.Error("Metadata should be set")
	}
	if opts.MaxPeers <= 0 {
		t.Error("MaxPeers should be set")
	}
	if opts.FileSavePath == "" {
		t.Error("FileSavePath should be set")
	}
	if !opts.Clip.AllowCopyFiles {
		t.Error("Clip.AllowCopyFiles should be true")
	}
	if opts.Clip.MaxClipboardFiles <= 0 {
		t.Error("Clip.MaxClipboardFiles should be set")
	}
	if opts.Clip.MaxFileSize <= 0 {
		t.Error("Clip.MaxFileSize should be set")
	}
}

func FuzzTransport_Set(f *testing.F) {
	f.Fuzz(func(t *testing.T, s string) {
		var tr node.Transport
		_ = tr.Set(s)
	})
}
