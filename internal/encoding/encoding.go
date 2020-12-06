/*
This package depends on the reedsolomon library written by Klaus Post. The 
original license is reproduced below:

The MIT License (MIT)

Copyright (c) 2015 Klaus Post
Copyright (c) 2015 Backblaze

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.*/

package encoding
import (
	"os"
	"log"
	"hash/crc32"
	"github.com/klauspost/reedsolomon"
	"alexhalogen/rsfileprotect/internal/filehelper"
	"alexhalogen/rsfileprotect/internal/types"
)

func Encode(meta types.Metadata, inFile *os.File, eccFile *os.File, crcFile *os.File) bool {

	bufferSize := int(meta.BlockSize)
	numData := int(meta.NumData)
	numRecovery := int(meta.NumRecovery)

	writer := filehelper.NewFileWriter(meta, eccFile, crcFile)
	writer.WriteMeta()
	bufferPages := make([][]byte, numData+numRecovery) // keeps buffer references
	buffer := make([][]byte, numData+numRecovery) // buffer array used during calculation
	for arr := range buffer {
		bufferPages[arr] = make([]byte, bufferSize)
	}

	zero_page := make([]byte, bufferSize)
	filehelper.Memset(zero_page, 0, bufferSize, 0)

	enc, err := reedsolomon.New(numData, numRecovery)
	if err != nil {
		log.Printf("Coder initialization failed at (%d, %d)\n", numData, numRecovery)
		return false
	}


	cf := filehelper.NewChunkedReader(inFile, bufferSize, numData)
	
	for {
		copy(buffer, bufferPages)
		var chunksRead int
		chunksRead, eof := cf.ReadNext(buffer[0:numData])
		if eof {
			break
		}
		if chunksRead != numData {
			for i:=chunksRead+1; i<numData; i++ {
				buffer[i] = zero_page
			}
		}

		err = enc.Encode(buffer)
		
		if err != nil {
			log.Println("Encoding failed!")
			return false
		}

		ok, err := enc.Verify(buffer)

		if err != nil || !ok {
			log.Println("Encoding verification failed!")
			return false
		}
		err =  writer.WriteECCChunk(buffer[numData:])
		if err != nil {
			log.Println(err)
			return false
		}
		eccs := make([]uint32, len(buffer))
		for i:=0; i<len(buffer); i++ {
			eccs[i] = crc32.ChecksumIEEE(buffer[i])
		}
		err = writer.WriteCRCChunk(eccs)
		if err != nil {
			log.Println(err)
			return false
		}
	}
	writer.Sync()
	return true
}