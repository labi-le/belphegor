package icon

import _ "embed"

var (
	//go:embed "red.ico"
	Red []byte
	//go:embed "green.ico"
	Green []byte

	//go:embed "red.png"
	RedPNG []byte
	//go:embed "green.png"
	GreenPNG []byte

	//go:embed "krisa.png"
	FullSize []byte
)
