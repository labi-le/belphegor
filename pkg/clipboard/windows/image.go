package windows

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"reflect"
	"unsafe"

	"golang.org/x/image/bmp"
)

type bitmapV5Header struct {
	Size          uint32
	Width         int32
	Height        int32
	Planes        uint16
	BitCount      uint16
	Compression   uint32
	SizeImage     uint32
	XPelsPerMeter int32
	YPelsPerMeter int32
	ClrUsed       uint32
	ClrImportant  uint32
	RedMask       uint32
	GreenMask     uint32
	BlueMask      uint32
	AlphaMask     uint32
	CSType        uint32
	Endpoints     struct {
		CiexyzRed, CiexyzGreen, CiexyzBlue struct {
			CiexyzX, CiexyzY, CiexyzZ int32
		}
	}
	GammaRed    uint32
	GammaGreen  uint32
	GammaBlue   uint32
	Intent      uint32
	ProfileData uint32
	ProfileSize uint32
	Reserved    uint32
}

type bitmapHeader struct {
	Size          uint32
	Width         uint32
	Height        uint32
	PLanes        uint16
	BitCount      uint16
	Compression   uint32
	SizeImage     uint32
	XPelsPerMeter uint32
	YPelsPerMeter uint32
	ClrUsed       uint32
	ClrImportant  uint32
}

func readImage() ([]byte, error) {
	hMem, _, err := getClipboardData.Call(cFmtDIBV5)
	if hMem == 0 {
		return readImageDib()
	}
	p, _, err := gLock.Call(hMem)
	if p == 0 {
		return nil, err
	}
	defer gUnlock.Call(hMem)

	info := (*bitmapV5Header)(unsafe.Pointer(p))
	if info.BitCount != 32 {
		return nil, errUnsupported
	}

	var data []byte
	sh := (*reflect.SliceHeader)(unsafe.Pointer(&data))
	sh.Data = uintptr(p)
	sh.Cap = int(info.Size + 4*uint32(info.Width)*uint32(info.Height))
	sh.Len = int(info.Size + 4*uint32(info.Width)*uint32(info.Height))
	img := image.NewRGBA(image.Rect(0, 0, int(info.Width), int(info.Height)))
	offset := int(info.Size)
	stride := int(info.Width)
	for y := 0; y < int(info.Height); y++ {
		for x := 0; x < int(info.Width); x++ {
			idx := offset + 4*(y*stride+x)
			xhat := (x + int(info.Width)) % int(info.Width)
			yhat := int(info.Height) - 1 - y
			r := data[idx+2]
			g := data[idx+1]
			b := data[idx+0]
			a := data[idx+3]
			img.SetRGBA(xhat, yhat, color.RGBA{r, g, b, a})
		}
	}
	var buf bytes.Buffer
	png.Encode(&buf, img)
	return buf.Bytes(), nil
}

func readImageDib() ([]byte, error) {
	const (
		fileHeaderLen = 14
		infoHeaderLen = 40
		cFmtDIB       = 8
	)

	hClipDat, _, err := getClipboardData.Call(cFmtDIB)
	if err != nil {
		return nil, errors.New("not dib format data: " + err.Error())
	}
	pMemBlk, _, err := gLock.Call(hClipDat)
	if pMemBlk == 0 {
		return nil, errors.New("failed to call global lock: " + err.Error())
	}
	defer gUnlock.Call(hClipDat)

	bmpHeader := (*bitmapHeader)(unsafe.Pointer(pMemBlk))
	dataSize := bmpHeader.SizeImage + fileHeaderLen + infoHeaderLen

	if bmpHeader.SizeImage == 0 && bmpHeader.Compression == 0 {
		iSizeImage := bmpHeader.Height * ((bmpHeader.Width*uint32(bmpHeader.BitCount)/8 + 3) &^ 3)
		dataSize += iSizeImage
	}
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, uint16('B')|(uint16('M')<<8))
	binary.Write(buf, binary.LittleEndian, uint32(dataSize))
	binary.Write(buf, binary.LittleEndian, uint32(0))
	const sizeof_colorbar = 0
	binary.Write(buf, binary.LittleEndian, uint32(fileHeaderLen+infoHeaderLen+sizeof_colorbar))
	j := 0
	for i := fileHeaderLen; i < int(dataSize); i++ {
		binary.Write(buf, binary.BigEndian, *(*byte)(unsafe.Pointer(pMemBlk + uintptr(j))))
		j++
	}
	return bmpToPng(buf)
}

func bmpToPng(bmpBuf *bytes.Buffer) (buf []byte, err error) {
	var f bytes.Buffer
	originalImage, err := bmp.Decode(bmpBuf)
	if err != nil {
		return nil, err
	}
	err = png.Encode(&f, originalImage)
	if err != nil {
		return nil, err
	}
	return f.Bytes(), nil
}

func writeImage(buf []byte) error {
	r, _, err := emptyClipboard.Call()
	if r == 0 {
		return fmt.Errorf("failed to clear clipboard: %w", err)
	}
	if len(buf) == 0 {
		return nil
	}

	img, err := png.Decode(bytes.NewReader(buf))
	if err != nil {
		return fmt.Errorf("input bytes is not PNG encoded: %w", err)
	}

	offset := unsafe.Sizeof(bitmapV5Header{})
	width := img.Bounds().Dx()
	height := img.Bounds().Dy()
	imageSize := 4 * width * height

	data := make([]byte, int(offset)+imageSize)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			idx := int(offset) + 4*(y*width+x)
			r, g, b, a := img.At(x, height-1-y).RGBA()
			data[idx+2] = uint8(r)
			data[idx+1] = uint8(g)
			data[idx+0] = uint8(b)
			data[idx+3] = uint8(a)
		}
	}

	info := bitmapV5Header{}
	info.Size = uint32(offset)
	info.Width = int32(width)
	info.Height = int32(height)
	info.Planes = 1
	info.Compression = 0 // BI_RGB
	info.SizeImage = uint32(4 * info.Width * info.Height)
	info.RedMask = 0xff0000
	info.GreenMask = 0xff00
	info.BlueMask = 0xff
	info.AlphaMask = 0xff000000
	info.BitCount = 32
	info.CSType = 0x73524742
	info.Intent = 4

	infob := make([]byte, int(unsafe.Sizeof(info)))
	for i, v := range *(*[unsafe.Sizeof(info)]byte)(unsafe.Pointer(&info)) {
		infob[i] = v
	}
	copy(data[:], infob[:])

	hMem, _, err := gAlloc.Call(gmemMoveable,
		uintptr(len(data)*int(unsafe.Sizeof(data[0]))))
	if hMem == 0 {
		return fmt.Errorf("failed to alloc global memory: %w", err)
	}

	p, _, err := gLock.Call(hMem)
	if p == 0 {
		return fmt.Errorf("failed to lock global memory: %w", err)
	}
	defer gUnlock.Call(hMem)

	memMove.Call(p, uintptr(unsafe.Pointer(&data[0])),
		uintptr(len(data)*int(unsafe.Sizeof(data[0]))))

	v, _, err := setClipboardData.Call(cFmtDIBV5, hMem)
	if v == 0 {
		gFree.Call(hMem)
		return fmt.Errorf("failed to set text to clipboard: %w", err)
	}

	return nil
}
