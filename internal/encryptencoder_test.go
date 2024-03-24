package internal

import (
	"bytes"
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
			fields:  NewCipher(),
			args:    []byte("hello"),
			want:    []byte("hello"),
			wantErr: false,
		},

		{
			name:    "large text",
			fields:  NewCipher(),
			args:    bytes.Repeat([]byte("hello"), 1000),
			want:    bytes.Repeat([]byte("hello"), 1000),
			wantErr: false,
		},

		{
			name:    "very large text",
			fields:  NewCipher(),
			args:    bytes.Repeat([]byte("hello"), 100_000),
			want:    bytes.Repeat([]byte("hello"), 100_000),
			wantErr: false,
		},

		{
			name:    "1kk large text",
			fields:  NewCipher(),
			args:    bytes.Repeat([]byte("hello"), 1_000_000),
			want:    bytes.Repeat([]byte("hello"), 1_000_000),
			wantErr: false,
		},

		{
			name:    "non ascii",
			fields:  NewCipher(),
			args:    []byte("hello\x00world"),
			want:    []byte("hello\x00world"),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.fields.Encrypt(tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("Encrypt() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			got2, errDec := tt.fields.Decrypt(got)
			if errDec != nil {
				t.Errorf("Decrypt() error = %v", errDec)
			}

			if !bytes.Equal(tt.want, got2) {
				t.Errorf("Decrypt() = %v, want %v", got2, tt.want)
			}
		})
	}
}
