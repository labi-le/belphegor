package encrypter

import (
	"bytes"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"testing"
)

func pkAndPbKey(t *testing.T) (crypto.PrivateKey, crypto.PublicKey) {
	privateKey, cipherErr := rsa.GenerateKey(rand.Reader, 2048)
	if cipherErr != nil {
		t.Fatal(cipherErr)
	}

	return privateKey, privateKey.Public()
}

func TestCipher_Encrypt(t *testing.T) {
	tests := []struct {
		name    string
		fields  *Cipher
		args    []byte
		want    []byte
		wantErr bool
	}{
		{
			name:    "short text",
			fields:  NewCipher(pkAndPbKey(t)),
			args:    []byte("hello"),
			want:    []byte("hello"),
			wantErr: false,
		},

		{
			name:    "large text",
			fields:  NewCipher(pkAndPbKey(t)),
			args:    bytes.Repeat([]byte("hello"), 1000),
			want:    bytes.Repeat([]byte("hello"), 1000),
			wantErr: false,
		},

		{
			name:    "fish",
			fields:  NewCipher(pkAndPbKey(t)),
			args:    []byte("A new login is required since the authentication session expired."),
			want:    []byte("A new login is required since the authentication session expired."),
			wantErr: false,
		},

		{
			name:    "very large text",
			fields:  NewCipher(pkAndPbKey(t)),
			args:    bytes.Repeat([]byte("hello"), 100_000),
			want:    bytes.Repeat([]byte("hello"), 100_000),
			wantErr: false,
		},

		{
			name:    "1kk large text",
			fields:  NewCipher(pkAndPbKey(t)),
			args:    bytes.Repeat([]byte("hello"), 1_000_000),
			want:    bytes.Repeat([]byte("hello"), 1_000_000),
			wantErr: false,
		},

		{
			name:    "non ascii",
			fields:  NewCipher(pkAndPbKey(t)),
			args:    []byte("hello\x00world"),
			want:    []byte("hello\x00world"),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.fields.Sign(rand.Reader, tt.args, crypto.SHA256)
			if (err != nil) != tt.wantErr {
				t.Errorf("WriteEncrypted() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			got2, errDec := tt.fields.Decrypt(rand.Reader, got, crypto.SHA256)
			if errDec != nil {
				t.Errorf("Decrypt() error = %v", errDec)
			}

			if !bytes.Equal(tt.want, got2) {
				t.Errorf("Decrypt() = %v, want %v", got2, tt.want)
			}
		})
	}
}

func BenchmarkCipher(b *testing.B) {
	benchmarks := []struct {
		name string
		data []byte
	}{
		{
			name: "short_text",
			data: []byte("hello"),
		},
		{
			name: "large_text",
			data: bytes.Repeat([]byte("hello"), 1000),
		},
		{
			name: "very_large_text",
			data: bytes.Repeat([]byte("hello"), 100_000),
		},
		{
			name: "mega_large_text",
			data: bytes.Repeat([]byte("hello"), 1_000_000),
		},
	}

	b.ReportAllocs()

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
			if err != nil {
				b.Fatal(err)
			}
			cipher := NewCipher(privateKey, privateKey.Public())

			b.Run("EncryptDecrypt", func(b *testing.B) {
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					encrypted, err := cipher.Sign(rand.Reader, bm.data, crypto.SHA256)
					if err != nil {
						b.Fatal(err)
					}

					_, err = cipher.Decrypt(rand.Reader, encrypted, crypto.SHA256)
					if err != nil {
						b.Fatal(err)
					}
				}
				b.SetBytes(int64(len(bm.data)))
			})
		})
	}
}
