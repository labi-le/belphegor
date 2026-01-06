//go:build windows

package windows

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	"image/png"
	"syscall"
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
	hMem, _, err := syscall.SyscallN(getClipboardData.Addr(), cFmtDIBV5)
	if hMem == 0 {
		return readImageDib()
	}
	p, _, err := syscall.SyscallN(gLock.Addr(), hMem)
	if p == 0 {
		if err != 0 {
			return nil, err
		}
		return nil, fmt.Errorf("global lock failed")
	}
	defer noCheck(syscall.SyscallN(gUnlock.Addr(), hMem))

	info := (*bitmapV5Header)(unsafe.Pointer(p))
	if info.BitCount != 32 {
		return nil, errUnsupported
	}

	size := int(info.Size)
	pixN := 4 * int(info.Width) * int(info.Height)

	pix := unsafe.Slice((*byte)(unsafe.Pointer(p+uintptr(size))), pixN)

	width := int(info.Width)
	height := int(info.Height)
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	stride := width

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			idx := 4 * (y*stride + x)
			yhat := height - 1 - y
			pixOffset := yhat*img.Stride + x*4

			img.Pix[pixOffset+0] = pix[idx+2]
			img.Pix[pixOffset+1] = pix[idx+1]
			img.Pix[pixOffset+2] = pix[idx+0]
			img.Pix[pixOffset+3] = pix[idx+3]
		}
	}
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes(), nil
}

func readImageDib() ([]byte, error) {
	const (
		fileHeaderLen = 14
		infoHeaderLen = 40
		cFmtDIB       = 8
	)

	hClipDat, _, err := syscall.SyscallN(getClipboardData.Addr(), cFmtDIB)
	if err != 0 {
		if hClipDat == 0 {
			return nil, errors.New("not dib format data: " + err.Error())
		}
	}
	if hClipDat == 0 {
		return nil, errors.New("not dib format data")
	}

	pMemBlk, _, err := syscall.SyscallN(gLock.Addr(), hClipDat)
	if pMemBlk == 0 {
		return nil, errors.New("failed to call global lock: " + err.Error())
	}
	defer noCheck(syscall.SyscallN(gUnlock.Addr(), hClipDat))

	bmpHeader := (*bitmapHeader)(unsafe.Pointer(pMemBlk))
	dataSize := bmpHeader.SizeImage + fileHeaderLen + infoHeaderLen

	if bmpHeader.SizeImage == 0 && bmpHeader.Compression == 0 {
		iSizeImage := bmpHeader.Height * ((bmpHeader.Width*uint32(bmpHeader.BitCount)/8 + 3) &^ 3)
		dataSize += iSizeImage
	}
	buf := new(bytes.Buffer)
	_ = binary.Write(buf, binary.LittleEndian, uint16('B')|(uint16('M')<<8))
	_ = binary.Write(buf, binary.LittleEndian, dataSize)
	_ = binary.Write(buf, binary.LittleEndian, uint32(0))
	_ = binary.Write(buf, binary.LittleEndian, uint32(fileHeaderLen+infoHeaderLen))

	j := 0
	for i := fileHeaderLen; i < int(dataSize); i++ {
		_ = binary.Write(buf, binary.BigEndian, *(*byte)(unsafe.Pointer(pMemBlk + uintptr(j))))
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
	r, _, err := syscall.SyscallN(emptyClipboard.Addr())
	if r == 0 {
		if err != 0 {
			return fmt.Errorf("failed to clear clipboard: %w", err)
		}
		return fmt.Errorf("failed to clear clipboard")
	}

	if len(buf) == 0 {
		return nil
	}

	img, decodeErr := png.Decode(bytes.NewReader(buf))
	if decodeErr != nil {
		return fmt.Errorf("input bytes is not PNG encoded: %w", err)
	}

	headerSize := unsafe.Sizeof(bitmapV5Header{})
	width := img.Bounds().Dx()
	height := img.Bounds().Dy()
	imageSize := 4 * width * height
	totalSize := uintptr(int(headerSize) + imageSize)

	hMem, _, err := syscall.SyscallN(gAlloc.Addr(), gmemMoveable, totalSize)
	if hMem == 0 {
		if err != 0 {
			return fmt.Errorf("failed to alloc global memory: %w", err)
		}
		return fmt.Errorf("failed to alloc global memory")
	}

	p, _, err := syscall.SyscallN(gLock.Addr(), hMem)
	if p == 0 {
		noCheck(syscall.SyscallN(gFree.Addr(), hMem))
		if err != 0 {
			return fmt.Errorf("failed to lock global memory: %w", err)
		}
		return fmt.Errorf("failed to lock global memory")
	}
	defer noCheck(syscall.SyscallN(gUnlock.Addr(), hMem))

	dst := unsafe.Slice((*byte)(unsafe.Pointer(p)), totalSize)

	info := bitmapV5Header{}
	info.Size = uint32(headerSize)
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

	headerBytes := unsafe.Slice((*byte)(unsafe.Pointer(&info)), headerSize)
	copy(dst, headerBytes)

	offset := int(headerSize)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			idx := offset + 4*(y*width+x)
			r, g, b, a := img.At(x, height-1-y).RGBA()
			// 0-0xffff
			dst[idx+2] = uint8(r >> 8)
			dst[idx+1] = uint8(g >> 8)
			dst[idx+0] = uint8(b >> 8)
			dst[idx+3] = uint8(a >> 8)
		}
	}

	v, _, err := syscall.SyscallN(setClipboardData.Addr(), cFmtDIBV5, hMem)
	if v == 0 {
		noCheck(syscall.SyscallN(gFree.Addr(), hMem))
		if err != 0 {
			return fmt.Errorf("failed to set text to clipboard: %w", err)
		}
		return fmt.Errorf("failed to set text to clipboard")
	}

	return nil
}
