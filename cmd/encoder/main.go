package main

import (
	"flag"
	"fmt"
	"os"
    "hash/crc32"
	"github.com/klauspost/reedsolomon"
	"alexhalogen/rsfileprotect/internal/filehelper"
	"alexhalogen/rsfileprotect/internal/types"
)


var outName = flag.String("out", "", "Output name")

func main() {
	
	var meta types.Metadata

	flag.Parse()
	args := flag.Args()
	inName := args[0]
	inFile, err := os.Open(inName)

	if err != nil {
		fmt.Println(err)
		return
	}
	defer inFile.Close()

	eccFile, err := os.OpenFile(*outName, os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0644)

	if err != nil {
		fmt.Println(err)
		return
	}
	defer eccFile.Close()

	crcFile, err := os.OpenFile((*outName)+".crc", os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0644)

	if err != nil {
		fmt.Println(err)
		return
	}
	defer crcFile.Close()


	numData := 10
	numRecovery := 1
	bufferSize := 4096
	
	meta.FileSize = 114514
	meta.BlockSize = (uint32)(bufferSize)
	meta.NumData = 10
	meta.NumRecovery = 1

	writer := filehelper.NewFileWriter(meta, eccFile, crcFile)
	writer.WriteMeta()
	buffer_pages := make([][]byte, numData+numRecovery) // keeps buffer references
	buffer := make([][]byte, numData+numRecovery) // buffer array used during calculation
	for arr := range buffer {
		buffer_pages[arr] = make([]byte, bufferSize)
	}

	zero_page := make([]byte, bufferSize)
	filehelper.Memset(zero_page, 0, bufferSize, 0)

	enc, err := reedsolomon.New(numData, numRecovery)

	cf := filehelper.NewChunkedFile(inFile, bufferSize, numData)
	eof := false
	for !eof {
		buffer = buffer_pages
		var chunksRead int
		chunksRead, eof = cf.ReadNext(buffer[0:numData])
		if chunksRead != numData {
			for i:=chunksRead+1; i<numData; i++ {
				buffer[i] = zero_page
			}
		}

		err = enc.Encode(buffer)
		
		if err != nil {
			fmt.Println("Encoding failed!")
			return
		}

		ok, err := enc.Verify(buffer)

		if err != nil || !ok {
			fmt.Println("Encoding verification failed!")
			return
		}
		err =  writer.WriteECCChunk(buffer[numData:])
		if err != nil {
			fmt.Println(err)
		}
		eccs := make([]uint32, len(buffer))
		for i:=0; i<len(buffer); i++ {
			eccs[i] = crc32.ChecksumIEEE(buffer[i])
		}
		err = writer.WriteCRCChunk(eccs)
		if err != nil {
			fmt.Println(err)
		}
	}

}
