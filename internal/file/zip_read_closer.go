package file

import (
	"archive/zip"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
)

const (
	directoryEndLen         = 22
	directory64LocLen       = 20
	directory64EndLen       = 56
	directory64LocSignature = 0x07064b50
	directory64EndSignature = 0x06064b50
)

type ZipReadCloser struct {
	*zip.Reader
	io.Closer
}

func OpenZip(filepath string) (*ZipReadCloser, error) {
	f, err := os.Open(filepath)
	if err != nil {
		return nil, err
	}
	fi, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, err
	}

	offset, err := findArchiveStartOffset(f, fi.Size())
	if err != nil {
		return nil, fmt.Errorf("cannot find beginning of zip archive=%q : %w", filepath, err)
	}

	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return nil, fmt.Errorf("unable to seek to beginning of archive: %w", err)
	}

	size := fi.Size() - int64(offset)

	r, err := zip.NewReader(io.NewSectionReader(f, int64(offset), size), size)
	if err != nil {
		return nil, fmt.Errorf("unable to open ZipReadCloser @ %q: %w", filepath, err)
	}

	return &ZipReadCloser{
		Reader: r,
		Closer: f,
	}, nil
}

type readBuf []byte

func (b *readBuf) uint16() uint16 {
	v := binary.LittleEndian.Uint16(*b)
	*b = (*b)[2:]
	return v
}

func (b *readBuf) uint32() uint32 {
	v := binary.LittleEndian.Uint32(*b)
	*b = (*b)[4:]
	return v
}

func (b *readBuf) uint64() uint64 {
	v := binary.LittleEndian.Uint64(*b)
	*b = (*b)[8:]
	return v
}

type directoryEnd struct {
	diskNbr            uint32
	dirDiskNbr         uint32
	dirRecordsThisDisk uint64
	directoryRecords   uint64
	directorySize      uint64
	directoryOffset    uint64
}

//

func findArchiveStartOffset(r io.ReaderAt, size int64) (startOfArchive uint64, err error) {

	var buf []byte
	var directoryEndOffset int64
	for i, bLen := range []int64{1024, 65 * 1024} {
		if bLen > size {
			bLen = size
		}
		buf = make([]byte, int(bLen))
		if _, err := r.ReadAt(buf, size-bLen); err != nil && err != io.EOF {
			return 0, err
		}
		if p := findSignatureInBlock(buf); p >= 0 {
			buf = buf[p:]
			directoryEndOffset = size - bLen + int64(p)
			break
		}
		if i == 1 || bLen == size {
			return 0, zip.ErrFormat
		}
	}

	if buf == nil {

		return 0, zip.ErrFormat
	}

	b := readBuf(buf[4:])
	d := &directoryEnd{
		diskNbr:            uint32(b.uint16()),
		dirDiskNbr:         uint32(b.uint16()),
		dirRecordsThisDisk: uint64(b.uint16()),
		directoryRecords:   uint64(b.uint16()),
		directorySize:      uint64(b.uint32()),
		directoryOffset:    uint64(b.uint32()),
	}

	if d.directoryRecords == 0xffff || d.directorySize == 0xffff || d.directoryOffset == 0xffffffff {
		p, err := findDirectory64End(r, directoryEndOffset)
		if err == nil && p >= 0 {
			directoryEndOffset = p
			err = readDirectory64End(r, p, d)
		}
		if err != nil {
			return 0, err
		}
		startOfArchive = 0
	}

	startOfArchive = uint64(directoryEndOffset) - d.directorySize - d.directoryOffset

	if o := int64(d.directoryOffset); o < 0 || o >= size {
		return 0, zip.ErrFormat
	}
	return startOfArchive, nil
}

func findDirectory64End(r io.ReaderAt, directoryEndOffset int64) (int64, error) {
	locOffset := directoryEndOffset - directory64LocLen
	if locOffset < 0 {
		return -1, nil
	}
	buf := make([]byte, directory64LocLen)
	if _, err := r.ReadAt(buf, locOffset); err != nil {
		return -1, err
	}
	b := readBuf(buf)
	if sig := b.uint32(); sig != directory64LocSignature {
		return -1, nil
	}
	if b.uint32() != 0 {
		return -1, nil
	}
	p := b.uint64()
	if b.uint32() != 1 {
		return -1, nil
	}
	return int64(p), nil
}

func readDirectory64End(r io.ReaderAt, offset int64, d *directoryEnd) (err error) {
	buf := make([]byte, directory64EndLen)
	if _, err := r.ReadAt(buf, offset); err != nil {
		return err
	}

	b := readBuf(buf)
	if sig := b.uint32(); sig != directory64EndSignature {
		return errors.New("could not read directory64End")
	}

	b = b[12:]
	d.diskNbr = b.uint32()
	d.dirDiskNbr = b.uint32()
	d.dirRecordsThisDisk = b.uint64()
	d.directoryRecords = b.uint64()
	d.directorySize = b.uint64()
	d.directoryOffset = b.uint64()

	return nil
}

func findSignatureInBlock(b []byte) int {
	for i := len(b) - directoryEndLen; i >= 0; i-- {

		if b[i] == 'P' && b[i+1] == 'K' && b[i+2] == 0x05 && b[i+3] == 0x06 {

			n := int(b[i+directoryEndLen-2]) | int(b[i+directoryEndLen-1])<<8
			if n+directoryEndLen+i <= len(b) {
				return i
			}
		}
	}
	return -1
}
