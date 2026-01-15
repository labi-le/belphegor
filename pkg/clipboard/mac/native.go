//go:build darwin

package mac

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"time"
	"unsafe"

	"github.com/ebitengine/purego"
	"github.com/ebitengine/purego/objc"
	"github.com/labi-le/belphegor/pkg/clipboard/eventful"
	"github.com/labi-le/belphegor/pkg/mime"
)

var _ eventful.Eventful = &Clipboard{}

func init() {
	_, err := purego.Dlopen("/System/Library/Frameworks/AppKit.framework/AppKit", purego.RTLD_GLOBAL|purego.RTLD_LAZY)
	if err != nil {
		panic(fmt.Errorf("mac clipboard: failed to load AppKit: %w", err))
	}
}

type Clipboard struct {
	dedup eventful.Deduplicator
}

func (m *Clipboard) Watch(ctx context.Context, update chan<- eventful.Update) error {
	defer close(update)

	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	clsNSPasteboard := objc.GetClass("NSPasteboard")
	clsNSString := objc.GetClass("NSString")

	selGeneralPasteboard := objc.RegisterName("generalPasteboard")
	selChangeCount := objc.RegisterName("changeCount")
	selStringForType := objc.RegisterName("stringForType:")
	selDataForType := objc.RegisterName("dataForType:")
	selPropertyListForType := objc.RegisterName("propertyListForType:")
	selUTF8String := objc.RegisterName("UTF8String")
	selBytes := objc.RegisterName("bytes")
	selLength := objc.RegisterName("length")
	selObjectAtIndex := objc.RegisterName("objectAtIndex:")
	selCount := objc.RegisterName("count")

	nsTypeText := makeNSString(clsNSString, "public.utf8-plain-text")
	nsTypePNG := makeNSString(clsNSString, "public.png")
	nsTypeFile := makeNSString(clsNSString, "NSFilenamesPboardType")

	pb := objc.ID(clsNSPasteboard).Send(selGeneralPasteboard)

	lastCount := pb.Send(selChangeCount)

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			currentCount := pb.Send(selChangeCount)
			if currentCount != lastCount {
				lastCount = currentCount

				nsData := pb.Send(selDataForType, nsTypePNG)
				if nsData != 0 {
					length := nsData.Send(selLength)
					if length > 0 {
						bytesPtr := nsData.Send(selBytes)
						data := unsafe.Slice((*byte)(unsafe.Pointer(bytesPtr)), int(length))

						if h, ok := m.dedup.Check(data); ok {
							dataCopy := make([]byte, len(data))
							copy(dataCopy, data)

							update <- eventful.Update{
								MimeType: mime.TypeImage,
								Data:     dataCopy,
								Hash:     h,
							}
							continue
						}
					}
				}

				nsList := pb.Send(selPropertyListForType, nsTypeFile)
				if nsList != 0 {
					count := nsList.Send(selCount)
					if count > 0 {
						nsStrPath := nsList.Send(selObjectAtIndex, 0)
						if nsStrPath != 0 {
							utf8Ptr := nsStrPath.Send(selUTF8String)
							if utf8Ptr != 0 {
								data := cStringToGoBytes(uintptr(utf8Ptr))

								if h, ok := m.dedup.Check(data); ok {
									dataCopy := make([]byte, len(data))
									copy(dataCopy, data)

									update <- eventful.Update{
										MimeType: mime.TypePath,
										Data:     dataCopy,
										Hash:     h,
									}
									continue
								}
							}
						}
					}
				}

				nsStr := pb.Send(selStringForType, nsTypeText)
				if nsStr != 0 {
					utf8Ptr := nsStr.Send(selUTF8String)
					if utf8Ptr != 0 {
						data := cStringToGoBytes(uintptr(utf8Ptr))

						if h, ok := m.dedup.Check(data); ok {
							dataCopy := make([]byte, len(data))
							copy(dataCopy, data)

							update <- eventful.Update{
								MimeType: mime.TypeText,
								Data:     dataCopy,
								Hash:     h,
							}
						}
					}
				}
			}
		}
	}
}

func (m *Clipboard) Write(t mime.Type, src []byte) (int, error) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	clsNSPasteboard := objc.GetClass("NSPasteboard")
	clsNSString := objc.GetClass("NSString")
	clsNSData := objc.GetClass("NSData")
	clsNSArray := objc.GetClass("NSArray")

	selGeneralPasteboard := objc.RegisterName("generalPasteboard")
	selClearContents := objc.RegisterName("clearContents")
	selSetString := objc.RegisterName("setString:forType:")
	selSetData := objc.RegisterName("setData:forType:")
	selSetPropertyList := objc.RegisterName("setPropertyList:forType:")
	selDataWithBytes := objc.RegisterName("dataWithBytes:length:")
	selArrayWithObject := objc.RegisterName("arrayWithObject:")

	pb := objc.ID(clsNSPasteboard).Send(selGeneralPasteboard)
	pb.Send(selClearContents)

	var ret uintptr

	switch t {
	case mime.TypeImage:
		nsTypePNG := makeNSString(clsNSString, "public.png")

		var bytesPtr unsafe.Pointer
		if len(src) > 0 {
			bytesPtr = unsafe.Pointer(&src[0])
		}
		nsData := objc.ID(clsNSData).Send(selDataWithBytes, uintptr(bytesPtr), uintptr(len(src)))

		ret = uintptr(pb.Send(selSetData, nsData, nsTypePNG))

	case mime.TypePath:
		nsTypeFile := makeNSString(clsNSString, "NSFilenamesPboardType")
		nsTypeText := makeNSString(clsNSString, "public.utf8-plain-text")
		nsStrPath := makeNSString(clsNSString, string(src))

		nsArray := objc.ID(clsNSArray).Send(selArrayWithObject, nsStrPath)

		ret = uintptr(pb.Send(selSetPropertyList, nsArray, nsTypeFile))

		pb.Send(selSetString, nsStrPath, nsTypeText)

	default:
		nsStrContent := makeNSString(clsNSString, string(src))
		nsTypeText := makeNSString(clsNSString, "public.utf8-plain-text")

		ret = uintptr(pb.Send(selSetString, nsStrContent, nsTypeText))
	}

	if ret == 0 {
		return 0, errors.New("failed to set clipboard content")
	}

	m.dedup.Mark(src)

	return len(src), nil
}

func makeNSString(clsNSString objc.Class, str string) objc.ID {
	selStringWithUTF8String := objc.RegisterName("stringWithUTF8String:")
	return objc.ID(clsNSString).Send(selStringWithUTF8String, str)
}

func cStringToGoBytes(ptr uintptr) []byte {
	if ptr == 0 {
		return nil
	}
	var length int
	for {
		if *(*byte)(unsafe.Pointer(ptr + uintptr(length))) == 0 {
			break
		}
		length++
	}
	return unsafe.Slice((*byte)(unsafe.Pointer(ptr)), length)
}
