package filehelper

import (
	"os"
	"bufio"
	"encoding/binary"
	"alexhalogen/rsfileprotect/internal/types"
)

type ChunkedReader struct {
	file *os.File
	chunkSize int
	offset int
}

type CRCReader struct {
	file *bufio.Reader
	buffer []byte
}

func NewChunkedReader(f *os.File, cs int, offset int) (ChunkedReader) {
	cf := ChunkedReader{file: f, chunkSize: cs, offset: offset}
	return cf
}

func NewCRCReader(f *os.File, size int) (CRCReader) {
	var reader CRCReader
	reader.buffer = make([]byte, 4)
	if size == 0 {
		reader.file = bufio.NewReader(f)
	} else {
		reader.file = bufio.NewReaderSize(f, size)
	}
	return reader
}

func (cr CRCReader) ReadNext(out []uint32) (int, error) {
	for i, _ := range out {
		err := binary.Read(cr.file, binary.LittleEndian, cr.buffer)
		if err != nil {
			return i, err
		}
		out[i] = binary.LittleEndian.Uint32(cr.buffer)
	}
	return len(out), nil
}

func (cf ChunkedReader) ReadNext(buffer [][]byte) (chunksRead int, eof bool){
	numChunks := len(buffer)
	if numChunks == 0 {
		return 0, false
	}
	bufferSize := cap(buffer[0])
	eof = false
	lastChunk := -1
	lastSize := 0

	for i:=0; i<numChunks; i++ {
		bytesRead, err := cf.file.Read(buffer[i])
		if err != nil {
			if bytesRead != 0 {
				print(err.Error())	// non-eof
			}
			if i == 0 {
				eof = true
				chunksRead = 0
				return
			}
			break
		} else {
			lastSize = bytesRead
			lastChunk = i
		}
	}
	// fill unread portion in lastChunk with zeros
	if lastSize != bufferSize {
		Memset(buffer[lastChunk], 0, bufferSize-lastSize, lastSize)
	}

	chunksRead = lastChunk+1
	return
}

func (cf ChunkedReader) SkipNext(chunks int, chunkSize int) (error){
	_, err := cf.file.Seek( int64(chunks*chunkSize), 1)
	return err
}


func ReadMeta(f *os.File, meta *types.Metadata) (error) {
	return binary.Read(f, binary.LittleEndian, meta)
}