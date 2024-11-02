package data

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"github.com/labi-le/belphegor/internal/types"
	"github.com/labi-le/belphegor/pkg/encrypter"
	"google.golang.org/protobuf/proto"
	"testing"
)

func BenchmarkMessageOperations(b *testing.B) {
	b.Run("NewMessage", func(b *testing.B) {
		b.ReportAllocs()
		data := bytes.Repeat([]byte("test"), 100)
		meta := SelfMetaData()

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			NewMessage(data, meta)
		}
	})

	b.Run("Marshal/Unmarshal", func(b *testing.B) {
		b.ReportAllocs()
		msg := NewMessage([]byte("test"), SelfMetaData())

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			data, err := proto.Marshal(msg.ToProto())
			if err != nil {
				b.Fatal(err)
			}

			var protoMsg types.Message
			if err := proto.Unmarshal(data, &protoMsg); err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("Equal", func(b *testing.B) {
		b.ReportAllocs()
		msg1 := NewMessage([]byte("test1"), SelfMetaData())
		msg2 := NewMessage([]byte("test2"), SelfMetaData())
		// Освобождаем оба сообщения после теста

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = msg1.Duplicate(msg2)
		}
	})

	b.Run("WriteEncrypted", func(b *testing.B) {
		b.ReportAllocs()
		msg := NewMessage([]byte("test"), SelfMetaData())

		buf := &bytes.Buffer{}
		privKey, _ := rsa.GenerateKey(rand.Reader, 2048)
		cipher := encrypter.NewCipher(privKey, privKey.Public())

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			buf.Reset()
			_, err := msg.WriteEncrypted(cipher, buf)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}
