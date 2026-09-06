package images

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
)

func TestCaptureTimeReadsDateTimeOriginal(t *testing.T) {
	path := filepath.Join(t.TempDir(), "a.jpg")
	writeJPEG(t, path, "2021:07:04 08:15:30")

	captured, ok := CaptureTime(path)
	if !ok {
		t.Fatalf("expected capture time for %s", path)
	}
	if got := captured.Format("2006-01-02 15:04:05"); got != "2021-07-04 08:15:30" {
		t.Fatalf("unexpected capture time %q", got)
	}
}

func TestCaptureTimeMissingEXIF(t *testing.T) {
	path := filepath.Join(t.TempDir(), "a.jpg")
	writeJPEG(t, path, "")

	if _, ok := CaptureTime(path); ok {
		t.Fatal("expected no capture time for stripped file")
	}
}

func TestSortByCaptureDateAscendingSendsUndatedToEnd(t *testing.T) {
	dir := t.TempDir()
	later := filepath.Join(dir, "a-later.jpg")
	earlier := filepath.Join(dir, "z-earlier.jpg")
	undatedB := filepath.Join(dir, "b-undated.jpg")
	undatedA := filepath.Join(dir, "a-undated.jpg")

	writeJPEG(t, later, "2021:07:04 10:00:00")
	writeJPEG(t, earlier, "2021:07:04 09:00:00")
	writeJPEG(t, undatedB, "")
	writeJPEG(t, undatedA, "")

	sorted := SortByCaptureDate([]string{later, undatedB, earlier, undatedA}, false)
	want := []string{earlier, later, undatedA, undatedB}
	for i := range want {
		if sorted[i] != want[i] {
			t.Fatalf("unexpected order %v, want %v", sorted, want)
		}
	}
}

func TestSortByCaptureDateDescendingKeepsUndatedByName(t *testing.T) {
	dir := t.TempDir()
	later := filepath.Join(dir, "z-later.jpg")
	earlier := filepath.Join(dir, "a-earlier.jpg")
	undatedB := filepath.Join(dir, "b-undated.jpg")
	undatedA := filepath.Join(dir, "a-undated.jpg")

	writeJPEG(t, later, "2021:07:04 10:00:00")
	writeJPEG(t, earlier, "2021:07:04 09:00:00")
	writeJPEG(t, undatedB, "")
	writeJPEG(t, undatedA, "")

	sorted := SortByCaptureDate([]string{undatedB, earlier, undatedA, later}, true)
	want := []string{later, earlier, undatedA, undatedB}
	for i := range want {
		if sorted[i] != want[i] {
			t.Fatalf("unexpected order %v, want %v", sorted, want)
		}
	}
}

// writeJPEG writes a minimal JPEG; a non-empty timestamp is embedded as EXIF
// DateTimeOriginal.
func writeJPEG(t *testing.T, path, timestamp string) {
	t.Helper()

	data := []byte{0xFF, 0xD8}
	if timestamp != "" {
		exif := append([]byte("Exif\x00\x00"), tiffBlock(timestamp)...)
		size := make([]byte, 2)
		binary.BigEndian.PutUint16(size, uint16(len(exif)+2))
		data = append(data, 0xFF, 0xE1)
		data = append(data, size...)
		data = append(data, exif...)
	}
	data = append(data, 0xFF, 0xD9)

	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func tiffBlock(timestamp string) []byte {
	value := append([]byte(timestamp), 0)

	const (
		ifd0Offset    = 8
		exifIFDOffset = 26
		valueOffset   = 44
	)

	block := make([]byte, valueOffset)
	copy(block, "II")
	binary.LittleEndian.PutUint16(block[2:4], 42)
	binary.LittleEndian.PutUint32(block[4:8], ifd0Offset)

	binary.LittleEndian.PutUint16(block[ifd0Offset:], 1)
	putIFDEntry(block[ifd0Offset+2:], tagExifIFD, exifTypeLong, 1, exifIFDOffset)

	binary.LittleEndian.PutUint16(block[exifIFDOffset:], 1)
	putIFDEntry(block[exifIFDOffset+2:], tagDateTimeOriginal, exifTypeASCII, uint32(len(value)), valueOffset)

	return append(block, value...)
}

func putIFDEntry(dst []byte, tag, typ uint16, count, value uint32) {
	binary.LittleEndian.PutUint16(dst[0:2], tag)
	binary.LittleEndian.PutUint16(dst[2:4], typ)
	binary.LittleEndian.PutUint32(dst[4:8], count)
	binary.LittleEndian.PutUint32(dst[8:12], value)
}
