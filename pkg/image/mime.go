package image

var imageMimeType = map[string]struct {
}{
	"image/png":  {},
	"image/jpeg": {},
	"image/bmp":  {},
	"image/gif":  {},
}

func HasPicture(mimeType string) bool {
	_, ok := imageMimeType[mimeType]
	return ok
}
