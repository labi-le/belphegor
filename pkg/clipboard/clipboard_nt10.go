//go:build windows

package clipboard

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"reflect"
	"runtime"
	"syscall"
	"time"
	"unicode/utf16"
	"unsafe"

	"golang.org/x/image/bmp"
)

var _ Eventful = &windows{}

func New() Eventful {
	return &windows{}
}

type windows struct{}

const (
	wmClipboardUpdate = 0x031D
	wmDestroy         = 0x0002
	className         = "BelphegorClipboardListener"
)

type wndClassEx struct {
	Size       uint32
	Style      uint32
	WndProc    uintptr
	ClsExtra   int32
	WndExtra   int32
	Instance   syscall.Handle
	Icon       syscall.Handle
	Cursor     syscall.Handle
	Background syscall.Handle
	MenuName   *uint16
	ClassName  *uint16
	IconSm     syscall.Handle
}

func (w *windows) Watch(ctx context.Context, update chan<- Update) error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	hInstance, _ := kernel32.NewProc("GetModuleHandleW").Call(0)
	clsNamePtr, _ := syscall.UTF16PtrFromString(className)

	wndProc := syscall.NewCallback(func(hwnd syscall.Handle, msg uint32, wparam, lparam uintptr) uintptr {
		switch msg {
		case wmClipboardUpdate:
			data := Read(FmtImage)
			if data == nil {
				data = Read(FmtText)
			}

			if len(data) > 0 {
				// Non-blocking send
				select {
				case update <- Update{Data: data}:
				default:
				}
			}
			return 0

		case wmDestroy:
			postQuitMessage.Call(0)
			return 0
		}

		ret, _, _ := defWindowProc.Call(uintptr(hwnd), uintptr(msg), wparam, lparam)
		return ret
	})

	wc := wndClassEx{
		Size:      uint32(unsafe.Sizeof(wndClassEx{})),
		Instance:  syscall.Handle(hInstance),
		WndProc:   wndProc,
		ClassName: clsNamePtr,
	}

	registerClassEx.Call(uintptr(unsafe.Pointer(&wc)))

	hwnd, _, _ := createWindowEx.Call(
		0,
		uintptr(unsafe.Pointer(clsNamePtr)),
		uintptr(unsafe.Pointer(clsNamePtr)),
		0, 0, 0, 0, 0,
		0,
		0, 0, 0,
	)

	if hwnd == 0 {
		return fmt.Errorf("failed to create window listener")
	}

	ret, _, _ := addClipboardFormatListener.Call(hwnd)
	if ret == 0 {
		destroyWindow.Call(hwnd)
		return fmt.Errorf("failed to add clipboard format listener")
	}

	done := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			postMessage.Call(hwnd, wmDestroy, 0, 0)
		case <-done:
		}
	}()

	var msg struct {
		Hwnd    syscall.Handle
		Message uint32
		WParam  uintptr
		LParam  uintptr
		Time    uint32
		Pt      struct{ X, Y int32 }
	}

	for {
		r, _, _ := getMessage.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
		if int32(r) <= 0 {
			break
		}
		translateMessage.Call(uintptr(unsafe.Pointer(&msg)))
		dispatchMessage.Call(uintptr(unsafe.Pointer(&msg)))
	}

	close(done)

	removeClipboardFormatListener.Call(hwnd)

	return nil
}

// Write implements Eventful.Write
func (w *windows) Write(p []byte) (n int, err error) {
	mime := http.DetectContentType(p)
	fmtType := FmtText
	if mime == "image/png" || mime == "image/jpeg" || mime == "image/gif" {
		fmtType = FmtImage
	}

	_, err = write(fmtType, p)
	if err != nil {
		return 0, err
	}
	return len(p), nil
}

// Name returns the implementation name
func (w *windows) Name() string {
	return "WindowsNT10"
}

var (
	debug          = false
	errUnavailable = errors.New("clipboard unavailable")
	errUnsupported = errors.New("unsupported format")
)

// Format represents the format of clipboard data.
type Format int

// All sorts of supported clipboard data
const (
	// FmtText indicates plain text clipboard format
	FmtText Format = iota
	// FmtImage indicates image/png clipboard format
	FmtImage
)

// Read returns a chunk of bytes of the clipboard data if it presents
// in the desired format t presents. Otherwise, it returns nil.
func Read(t Format) []byte {
	buf, err := read(t)
	if err != nil {
		return nil
	}
	return buf
}

func read(t Format) (buf []byte, err error) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	var format uintptr
	switch t {
	case FmtImage:
		format = cFmtDIBV5
	case FmtText:
		fallthrough
	default:
		format = cFmtUnicodeText
	}

	// check if clipboard is avaliable for the requested format
	r, _, err := isClipboardFormatAvailable.Call(format)
	if r == 0 {
		return nil, errUnavailable
	}

	// Retry loop for OpenClipboard
	for i := 0; i < 5; i++ {
		r, _, _ = openClipboard.Call(0)
		if r != 0 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if r == 0 {
		return nil, errors.New("failed to open clipboard")
	}
	defer closeClipboard.Call()

	switch format {
	case cFmtDIBV5:
		return readImage()
	case cFmtUnicodeText:
		fallthrough
	default:
		return readText()
	}
}

func write(t Format, buf []byte) (<-chan struct{}, error) {
	errch := make(chan error)
	changed := make(chan struct{}, 1)
	go func() {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()

		// Retry loop for OpenClipboard
		var r uintptr
		for i := 0; i < 5; i++ {
			r, _, _ = openClipboard.Call(0)
			if r != 0 {
				break
			}
			time.Sleep(10 * time.Millisecond)
		}
		if r == 0 {
			errch <- errors.New("failed to open clipboard")
			return
		}

		switch t {
		case FmtImage:
			err := writeImage(buf)
			if err != nil {
				errch <- err
				closeClipboard.Call()
				return
			}
		case FmtText:
			fallthrough
		default:
			err := writeText(buf)
			if err != nil {
				errch <- err
				closeClipboard.Call()
				return
			}
		}
		closeClipboard.Call()

		errch <- nil
		close(changed)
	}()

	err := <-errch
	if err != nil {
		return nil, err
	}
	return changed, nil
}

func readText() (buf []byte, err error) {
	hMem, _, err := getClipboardData.Call(cFmtUnicodeText)
	if hMem == 0 {
		return nil, err
	}
	p, _, err := gLock.Call(hMem)
	if p == 0 {
		return nil, err
	}
	defer gUnlock.Call(hMem)

	// Find NUL terminator
	n := 0
	for ptr := unsafe.Pointer(p); *(*uint16)(ptr) != 0; n++ {
		ptr = unsafe.Pointer(uintptr(ptr) +
			unsafe.Sizeof(*((*uint16)(unsafe.Pointer(p)))))
	}

	var s []uint16
	h := (*reflect.SliceHeader)(unsafe.Pointer(&s))
	h.Data = p
	h.Len = n
	h.Cap = n
	return []byte(string(utf16.Decode(s))), nil
}

func writeText(buf []byte) error {
	r, _, err := emptyClipboard.Call()
	if r == 0 {
		return fmt.Errorf("failed to clear clipboard: %w", err)
	}

	if len(buf) == 0 {
		return nil
	}

	s, err := syscall.UTF16FromString(string(buf))
	if err != nil {
		return fmt.Errorf("failed to convert given string: %w", err)
	}

	hMem, _, err := gAlloc.Call(gmemMoveable, uintptr(len(s)*int(unsafe.Sizeof(s[0]))))
	if hMem == 0 {
		return fmt.Errorf("failed to alloc global memory: %w", err)
	}

	p, _, err := gLock.Call(hMem)
	if p == 0 {
		return fmt.Errorf("failed to lock global memory: %w", err)
	}
	defer gUnlock.Call(hMem)

	memMove.Call(p, uintptr(unsafe.Pointer(&s[0])),
		uintptr(len(s)*int(unsafe.Sizeof(s[0]))))

	v, _, err := setClipboardData.Call(cFmtUnicodeText, hMem)
	if v == 0 {
		gFree.Call(hMem)
		return fmt.Errorf("failed to set text to clipboard: %w", err)
	}

	return nil
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
	original_image, err := bmp.Decode(bmpBuf)
	if err != nil {
		return nil, err
	}
	err = png.Encode(&f, original_image)
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

const (
	cFmtBitmap      = 2
	cFmtUnicodeText = 13
	cFmtDIBV5       = 17
	cFmtDataObject  = 49161
	gmemMoveable    = 0x0002
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

// Win32 API
var (
	user32   = syscall.MustLoadDLL("user32")
	kernel32 = syscall.NewLazyDLL("kernel32")

	openClipboard              = user32.MustFindProc("OpenClipboard")
	closeClipboard             = user32.MustFindProc("CloseClipboard")
	emptyClipboard             = user32.MustFindProc("EmptyClipboard")
	getClipboardData           = user32.MustFindProc("GetClipboardData")
	setClipboardData           = user32.MustFindProc("SetClipboardData")
	isClipboardFormatAvailable = user32.MustFindProc("IsClipboardFormatAvailable")
	enumClipboardFormats       = user32.MustFindProc("EnumClipboardFormats")
	getClipboardSequenceNumber = user32.MustFindProc("GetClipboardSequenceNumber")
	registerClipboardFormatA   = user32.MustFindProc("RegisterClipboardFormatA")

	addClipboardFormatListener    = user32.NewProc("AddClipboardFormatListener")
	removeClipboardFormatListener = user32.NewProc("RemoveClipboardFormatListener")
	createWindowEx                = user32.NewProc("CreateWindowExW")
	defWindowProc                 = user32.NewProc("DefWindowProcW")
	registerClassEx               = user32.NewProc("RegisterClassExW")
	getMessage                    = user32.NewProc("GetMessageW")
	dispatchMessage               = user32.NewProc("DispatchMessageW")
	translateMessage              = user32.NewProc("TranslateMessageW")
	postQuitMessage               = user32.NewProc("PostQuitMessage")
	destroyWindow                 = user32.NewProc("DestroyWindow")
	postMessage                   = user32.NewProc("PostMessageW")

	gLock   = kernel32.NewProc("GlobalLock")
	gUnlock = kernel32.NewProc("GlobalUnlock")
	gAlloc  = kernel32.NewProc("GlobalAlloc")
	gFree   = kernel32.NewProc("GlobalFree")
	memMove = kernel32.NewProc("RtlMoveMemory")
)
