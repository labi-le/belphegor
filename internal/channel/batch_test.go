package channel_test

import (
	"testing"

	"github.com/labi-le/belphegor/internal/channel"
	"github.com/labi-le/belphegor/internal/types/domain"
)

func TestBatchCollector_New(t *testing.T) {
	bc := channel.NewBatchCollector()
	if bc == nil {
		t.Fatal("NewBatchCollector returned nil")
	}
}

func TestBatchCollector_Add_Incomplete(t *testing.T) {
	bc := channel.NewBatchCollector()

	msg := domain.Message{
		ID:         1,
		BatchID:    100,
		BatchTotal: 3,
		Data:       []byte("part1"),
	}

	data, complete := bc.Add(msg)
	if data != nil {
		t.Error("expected nil data for incomplete batch")
	}
	if complete {
		t.Error("expected complete=false for incomplete batch")
	}
}

func TestBatchCollector_Add_Complete(t *testing.T) {
	bc := channel.NewBatchCollector()

	msgs := []domain.Message{
		{ID: 1, BatchID: 100, BatchTotal: 3, Data: []byte("part1")},
		{ID: 2, BatchID: 100, BatchTotal: 3, Data: []byte("part2")},
		{ID: 3, BatchID: 100, BatchTotal: 3, Data: []byte("part3")},
	}

	for i, msg := range msgs {
		data, complete := bc.Add(msg)
		if i < 2 {
			if data != nil {
				t.Errorf("expected nil data on msg %d", i+1)
			}
			if complete {
				t.Errorf("expected complete=false on msg %d", i+1)
			}
		} else {
			if data == nil {
				t.Error("expected joined data on final msg")
			}
			if !complete {
				t.Error("expected complete=true on final msg")
			}
			expected := "part1\npart2\npart3"
			if string(data) != expected {
				t.Errorf("expected %q, got %q", expected, string(data))
			}
		}
	}
}

func TestBatchCollector_Add_EmptyData(t *testing.T) {
	bc := channel.NewBatchCollector()

	msgs := []domain.Message{
		{ID: 1, BatchID: 100, BatchTotal: 2, Data: []byte{}},
		{ID: 2, BatchID: 100, BatchTotal: 2, Data: []byte("data")},
	}

	_, _ = bc.Add(msgs[0])
	data, complete := bc.Add(msgs[1])

	if !complete {
		t.Error("expected complete=true")
	}
	if string(data) != "data" {
		t.Errorf("expected 'data', got %q", string(data))
	}
}

func TestBatchCollector_MultipleBatches(t *testing.T) {
	bc := channel.NewBatchCollector()

	batch1 := []domain.Message{
		{ID: 1, BatchID: 1, BatchTotal: 2, Data: []byte("a1")},
		{ID: 2, BatchID: 1, BatchTotal: 2, Data: []byte("a2")},
	}
	batch2 := []domain.Message{
		{ID: 3, BatchID: 2, BatchTotal: 2, Data: []byte("b1")},
		{ID: 4, BatchID: 2, BatchTotal: 2, Data: []byte("b2")},
	}

	_, _ = bc.Add(batch1[0])
	_, _ = bc.Add(batch2[0])
	_, _ = bc.Add(batch1[1])
	_, _ = bc.Add(batch2[1])
}
