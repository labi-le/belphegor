//go:build windows
// +build windows

package clipboard

import (
	"bytes"
	"os"
	"testing"
)

func Test_windows_Set(t *testing.T) {
	type args struct {
		data []byte
	}
	tests := []struct {
		name    string
		args    args
		want    []byte
		wantErr bool
	}{
		{
			name:    "default test",
			args:    args{data: getTestFile(t)},
			want:    getTestFile(t),
			wantErr: false,
		}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewManager()
			if err := p.Set(tt.args.data); (err != nil) != tt.wantErr {
				t.Errorf("Set() error = %v, wantErr %v", err, tt.wantErr)
			}
			got, err := p.Get()
			if (err != nil) != tt.wantErr {
				t.Errorf("Get() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if bytes.Equal(got, tt.want) {
				t.Errorf("Get() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func getTestFile(t *testing.T) []byte {
	file, err := os.ReadFile("clipboard_windows_test.png")
	if err != nil {
		t.Fatal(err)
	}

	return file
}
