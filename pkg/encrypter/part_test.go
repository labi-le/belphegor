package encrypter

import (
	"bytes"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"testing"
)

func Test_explodeBySize(t *testing.T) {
	type args struct {
		src  []byte
		size int
	}
	tests := []struct {
		name string
		args args
		want int
	}{
		{
			name: "one",
			args: args{
				src:  make([]byte, 1),
				size: 10,
			},
			want: 1,
		},

		{
			name: "empty",
			args: args{
				src:  make([]byte, 0),
				size: 10,
			},
			want: 0,
		},

		{
			name: "large",
			args: args{
				src:  make([]byte, 100),
				size: 10,
			},
			want: 10,
		},

		{
			name: "remainder",
			args: args{
				src:  make([]byte, 101),
				size: 10,
			},
			want: 11,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := explodeBySize(tt.args.src, tt.args.size); len(got) != tt.want {
				t.Errorf("explodeBySize() = %v, want %v", got, tt.want)
			}
		})
	}
}
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
