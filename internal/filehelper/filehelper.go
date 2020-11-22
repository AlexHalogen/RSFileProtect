package filehelper

import (
	"os"
)

type ChunkedFile struct {
	file *os.File
	chunkSize int
	offset int
}

func NewChunkedFile(f *os.File, cs int, offset int) (ChunkedFile) {
	cf := ChunkedFile{file: f, chunkSize: cs, offset: offset}
	return cf
}

func (cf ChunkedFile) ReadNext(buffer [][]byte) (chunksRead int, eof bool){
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
			eof = true
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


