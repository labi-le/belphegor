package metadata_test

import (
	"testing"

	"github.com/labi-le/belphegor/internal/metadata"
)

func TestIsMajorDifference(t *testing.T) {
	tests := []struct {
		name string
		v1   string
		v2   string
		want bool
	}{
		{"freshest on left is never a difference", "freshest", "v1.2.3", false},
		{"freshest on right is never a difference", "v1.2.3", "freshest", false},
		{"same major differs only in minor/patch", "v1.2.3", "v1.9.0", false},
		{"different major", "v1.2.3", "v2.0.0", true},
		{"v-dot prefix normalized", "v.1.0.0", "1.5.2", false},
		{"bare versions different major", "3.0.0", "4.0.0", true},
		{"both unparseable collapse to zero", "garbage", "alsobad", false},
		{"unparseable vs valid major", "garbage", "v2.0.0", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := metadata.IsMajorDifference(tt.v1, tt.v2); got != tt.want {
				t.Fatalf("IsMajorDifference(%q, %q) = %v, want %v", tt.v1, tt.v2, got, tt.want)
			}
		})
	}
}
