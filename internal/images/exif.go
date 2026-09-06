package images

import (
	"bufio"
	"encoding/binary"
	"errors"
	"io"
	"os"
	"strings"
	"time"
)

const (
	tagDateTime          = 0x0132
	tagExifIFD           = 0x8769
	tagDateTimeOriginal  = 0x9003
	tagDateTimeDigitized = 0x9004
)

const (
	exifTypeASCII = 2
	exifTypeLong  = 4
)

// exifTimeLayout is the EXIF datetime format; it carries no timezone, so the
// parsed values are only comparable against each other.
const exifTimeLayout = "2006:01:02 15:04:05"

// maxEXIFSegment caps how much of a file is inspected while looking for the
// EXIF block, so oversized or malformed files cannot be read into memory.
const maxEXIFSegment = 1 << 20

var errNoEXIF = errors.New("no exif capture time")

// CaptureTime reports the camera capture timestamp recorded in a file's EXIF
// data. ok is false when the file has no readable capture timestamp.
func CaptureTime(path string) (captured time.Time, ok bool) {
	file, err := os.Open(path)
	if err != nil {
		return time.Time{}, false
	}
	defer file.Close()

	tiff, err := jpegEXIFBlock(bufio.NewReader(file))
	if err != nil {
		return time.Time{}, false
	}

	captured, err = exifCaptureTime(tiff)
	if err != nil {
		return time.Time{}, false
	}
	return captured, true
}

// jpegEXIFBlock walks JPEG markers and returns the TIFF block of the APP1
// Exif segment.
func jpegEXIFBlock(reader *bufio.Reader) ([]byte, error) {
	var soi [2]byte
	if _, err := io.ReadFull(reader, soi[:]); err != nil {
		return nil, err
	}
	if soi[0] != 0xFF || soi[1] != 0xD8 {
		return nil, errNoEXIF
	}

	for {
		marker, err := nextJPEGMarker(reader)
		if err != nil {
			return nil, err
		}
		// Start of scan or end of image: no metadata segments remain.
		if marker == 0xDA || marker == 0xD9 {
			return nil, errNoEXIF
		}
		// Standalone markers carry no payload.
		if marker == 0x01 || (marker >= 0xD0 && marker <= 0xD8) {
			continue
		}

		var sizeBytes [2]byte
		if _, err := io.ReadFull(reader, sizeBytes[:]); err != nil {
			return nil, err
		}
		size := int(binary.BigEndian.Uint16(sizeBytes[:])) - 2
		if size < 0 || size > maxEXIFSegment {
			return nil, errNoEXIF
		}

		payload := make([]byte, size)
		if _, err := io.ReadFull(reader, payload); err != nil {
			return nil, err
		}
		if marker == 0xE1 && len(payload) > 6 && string(payload[:6]) == "Exif\x00\x00" {
			return payload[6:], nil
		}
	}
}

func nextJPEGMarker(reader *bufio.Reader) (byte, error) {
	for {
		value, err := reader.ReadByte()
		if err != nil {
			return 0, err
		}
		if value != 0xFF {
			continue
		}
		for value == 0xFF {
			value, err = reader.ReadByte()
			if err != nil {
				return 0, err
			}
		}
		if value == 0x00 {
			continue
		}
		return value, nil
	}
}

func exifCaptureTime(tiff []byte) (time.Time, error) {
	if len(tiff) < 8 {
		return time.Time{}, errNoEXIF
	}

	var order binary.ByteOrder
	switch string(tiff[:2]) {
	case "II":
		order = binary.LittleEndian
	case "MM":
		order = binary.BigEndian
	default:
		return time.Time{}, errNoEXIF
	}
	if order.Uint16(tiff[2:4]) != 42 {
		return time.Time{}, errNoEXIF
	}

	ifd0 := readIFD(tiff, order, int(order.Uint32(tiff[4:8])))

	var candidates []string
	if entry, found := ifd0[tagExifIFD]; found {
		if offset, ok := entry.longValue(order); ok {
			exifIFD := readIFD(tiff, order, int(offset))
			candidates = append(candidates,
				exifIFD[tagDateTimeOriginal].asciiValue(tiff, order),
				exifIFD[tagDateTimeDigitized].asciiValue(tiff, order),
			)
		}
	}
	candidates = append(candidates, ifd0[tagDateTime].asciiValue(tiff, order))

	for _, candidate := range candidates {
		if candidate == "" {
			continue
		}
		parsed, err := time.Parse(exifTimeLayout, candidate)
		if err != nil {
			continue
		}
		return parsed, nil
	}
	return time.Time{}, errNoEXIF
}

type ifdEntry struct {
	typ   uint16
	count uint32
	value []byte
}

func readIFD(tiff []byte, order binary.ByteOrder, offset int) map[uint16]ifdEntry {
	entries := make(map[uint16]ifdEntry)
	if offset < 8 || offset+2 > len(tiff) {
		return entries
	}

	count := int(order.Uint16(tiff[offset : offset+2]))
	pos := offset + 2
	for i := 0; i < count; i++ {
		if pos+12 > len(tiff) {
			break
		}
		tag := order.Uint16(tiff[pos : pos+2])
		entries[tag] = ifdEntry{
			typ:   order.Uint16(tiff[pos+2 : pos+4]),
			count: order.Uint32(tiff[pos+4 : pos+8]),
			value: tiff[pos+8 : pos+12],
		}
		pos += 12
	}
	return entries
}

func (e ifdEntry) asciiValue(tiff []byte, order binary.ByteOrder) string {
	if e.typ != exifTypeASCII || e.count == 0 || e.count > uint32(len(tiff)) {
		return ""
	}

	size := int(e.count)
	if size <= 4 {
		return strings.Trim(string(e.value[:size]), "\x00 ")
	}

	offset := int(order.Uint32(e.value))
	if offset < 8 || offset+size > len(tiff) {
		return ""
	}
	return strings.Trim(string(tiff[offset:offset+size]), "\x00 ")
}

func (e ifdEntry) longValue(order binary.ByteOrder) (uint32, bool) {
	if e.typ != exifTypeLong || e.count != 1 {
		return 0, false
	}
	return order.Uint32(e.value), true
}
