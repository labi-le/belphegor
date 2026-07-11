package channel_test

import (
	"strconv"
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
		last := i == len(msgs)-1

		if complete != last {
			t.Errorf("msg %d: complete=%v, want %v", i+1, complete, last)
		}
		if !last && data != nil {
			t.Errorf("msg %d: expected nil data before completion, got %q", i+1, data)
		}
		if last && string(data) != "part1\npart2\npart3" {
			t.Errorf("final msg: expected %q, got %q", "part1\npart2\npart3", string(data))
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

	data1, done1 := bc.Add(batch1[1])
	if !done1 || string(data1) != "a1\na2" {
		t.Errorf("batch1 not completed correctly: data=%q complete=%v", data1, done1)
	}

	data2, done2 := bc.Add(batch2[1])
	if !done2 || string(data2) != "b1\nb2" {
		t.Errorf("batch2 not completed correctly: data=%q complete=%v", data2, done2)
	}
}

// BenchmarkBatchCollector_Add measures assembling a complete file batch:
// N part inserts into the batch map followed by the sort+join on completion.
func BenchmarkBatchCollector_Add(b *testing.B) {
	for _, n := range []int{2, 10, 50} {
		parts := make([]domain.Message, n)
		for i := range parts {
			parts[i] = domain.Message{
				ID:         domain.MessageID(i + 1),
				BatchID:    domain.MessageID(9999),
				BatchTotal: uint32(n),
				Data:       []byte("/tmp/clipboard/file_" + strconv.Itoa(i)),
			}
		}

		b.Run(strconv.Itoa(n)+"_parts", func(b *testing.B) {
			bc := channel.NewBatchCollector()
			b.ReportAllocs()
			b.ResetTimer()

			for range b.N {
				for _, p := range parts {
					bc.Add(p)
				}
			}
		})
	}
}
