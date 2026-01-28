package channel_test

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/labi-le/belphegor/internal/channel"
	"github.com/labi-le/belphegor/internal/types/domain"
	"github.com/labi-le/belphegor/pkg/id"
	"github.com/labi-le/belphegor/pkg/mime"
)

func TestChannel_Send_Deduplication(t *testing.T) {
	ch := channel.New(1)
	msgID := id.New()

	payload := domain.Message{
		ID:            msgID,
		Data:          []byte("test"),
		MimeType:      mime.TypeText,
		ContentHash:   12345,
		ContentLength: 4,
	}
	event := domain.EventMessage{Payload: payload}

	go func() {
		ch.Send(event)
	}()

	select {
	case received := <-ch.Messages():
		if received.Payload.ID != msgID {
			t.Fatalf("expected msgID %d, got %d", msgID, received.Payload.ID)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for first message")
	}

	done := make(chan bool)
	go func() {
		ch.Send(event)
		close(done)
	}()

	select {
	case <-ch.Messages():
		t.Fatal("duplicate message was sent to channel, deduplication failed")
	case <-done:
	case <-time.After(100 * time.Millisecond):
	}

	last := ch.LastMsg()
	if last.Payload.ID != msgID {
		t.Errorf("LastMsg state mismatch. Want %d, got %d", msgID, last.Payload.ID)
	}
}

func TestChannel_History_Eviction(t *testing.T) {
	ch := channel.New(1)

	for i := 0; i < channel.HistorySize+1; i++ {
		msg := domain.Message{
			ID:            id.Unique(i + 1),
			Data:          []byte(fmt.Sprintf("file_%d", i)),
			MimeType:      mime.TypePath,
			ContentHash:   uint64(i + 1),
			ContentLength: 10,
		}

		go func() {
			select {
			case <-ch.Messages():
			case <-time.After(100 * time.Millisecond):
			}
		}()

		ch.Send(domain.EventMessage{Payload: msg})
		time.Sleep(10 * time.Millisecond)
	}

	if _, ok := ch.Get(1); ok {
		t.Fatal("expected ID 1 to be evicted from history, but it was found")
	}

	if msg, ok := ch.Get(2); !ok {
		t.Error("expected ID 2 to be present in history")
	} else if msg.Payload.ID != 2 {
		t.Errorf("got wrong message id, expected 2, got %d", msg.Payload.ID)
	}

	if msg, ok := ch.Get(6); !ok {
		t.Error("expected ID 6 (last) to be present")
	} else if msg.Payload.ID != 6 {
		t.Errorf("got wrong message id, expected 6, got %d", msg.Payload.ID)
	}
}

func TestChannel_Get_LookupLogic(t *testing.T) {
	ch := channel.New(1)

	msg1 := domain.EventMessage{
		Payload: domain.Message{
			ID:          100,
			MimeType:    mime.TypePath,
			ContentHash: 100,
		},
	}
	msg2 := domain.EventMessage{
		Payload: domain.Message{
			ID:          200,
			MimeType:    mime.TypeText,
			ContentHash: 200,
		},
	}

	go func() { <-ch.Messages() }()
	ch.Send(msg1)

	go func() { <-ch.Messages() }()
	ch.Send(msg2)

	got2, ok := ch.Get(200)
	if !ok || got2.Payload.ID != 200 {
		t.Error("failed to retrieve LastMsg via Get")
	}

	got1, ok := ch.Get(100)
	if !ok || got1.Payload.ID != 100 {
		t.Error("failed to retrieve historic message via Get")
	}

	if _, ok := ch.Get(999); ok {
		t.Error("retrieved non-existent message")
	}
}

func TestChannel_Announce_Deduplication(t *testing.T) {
	ch := channel.New(10)

	ann := domain.EventAnnounce{
		Payload: domain.Announce{
			ContentHash: 0xDEADBEEF,
		},
	}

	ch.Announce(ann)
	select {
	case <-ch.Announcements():
	default:
		t.Fatal("announce channel empty")
	}

	ch.Announce(ann)
	select {
	case <-ch.Announcements():
		t.Fatal("duplicate announce passed through")
	default:
	}
}

func TestChannel_Concurrency_Race(t *testing.T) {
	ch := channel.New(0)
	var wg sync.WaitGroup
	workers := 10
	iterations := 100

	done := make(chan struct{})
	go func() {
		for {
			select {
			case <-ch.Messages():
			case <-ch.Announcements():
			case <-done:
				return
			}
		}
	}()

	wg.Add(workers * 3)

	for i := 0; i < workers; i++ {
		go func(idVal int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				ch.Send(domain.EventMessage{
					Payload: domain.Message{
						ID:            id.Unique(j),
						MimeType:      mime.TypePath,
						ContentHash:   uint64(j),
						ContentLength: 10,
					},
				})
			}
		}(i)
	}

	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				ch.Announce(domain.EventAnnounce{
					Payload: domain.Announce{
						ContentHash: uint64(j),
					},
				})
			}
		}()
	}

	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				ch.LastMsg()
				ch.Get(id.Unique(j))
			}
		}()
	}

	wg.Wait()
	close(done)
}
