package main

import (
	"flag"
	"fmt"
	"os"
	// "github.com/klauspost/reedsolomon"
    "hash/crc32"
    "encoding/binary"
	"alexhalogen/rsfileprotect/internal/filehelper"
	"alexhalogen/rsfileprotect/internal/types"
)


var eccName = flag.String("ecc", "", "ecc file containing code needed to restore file")
var crcName = flag.String("crc", "", "crc file for quick integrity check and restoration")

func main() {
	flag.Parse()
	args := flag.Args()

	inName := args[0]

	inFile, err := os.Open(inName)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer inFile.Close()

	eccFile, err := os.Open(*eccName)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer eccFile.Close()
	
	var crcFile *os.File
	if crcName != nil {
		crcFile, err = os.Open(*crcName)
		if err != nil {
			fmt.Println(err)
			return
		}	
	}

	fmt.Printf("Data: %s, ECC: %s, CRC: %s\n", inName, *eccName, *crcName)

	var meta types.Metadata;
	err = filehelper.ReadMeta(eccFile, &meta)
	if err != nil {
		fmt.Println("Failed to read metadata from ecc file!")
		return
	}

	fmt.Println(meta)

	// fileSize := meta.FileSize
	numData := (int)(meta.NumData)
	numRecovery := (int)(meta.NumRecovery)
	bufferSize := (int)(meta.BlockSize)
	fileBufferPages := make([][]byte, numData);
	eccBuffer := make([][]byte, numRecovery)
	crcBuffer := make([]uint32, numData+numRecovery)
	
	fileBuffer := make([][]byte, numData)
	// eccBuffer := make([][]byte, numRecovery)

	zero_page := make([]byte, bufferSize)
	filehelper.Memset(zero_page, 0, bufferSize, 0)
	
	for i, _ := range fileBuffer {
		fileBufferPages[i] = make([]byte, bufferSize)
	}
	for i, _ := range eccBuffer {
		eccBuffer[i] = make([]byte, bufferSize)
	}
	// enc, err := reedsolomon.New(numData, numRecovery)

	eccReader := filehelper.NewChunkedReader(eccFile, bufferSize, 0)
	fileReader := filehelper.NewChunkedReader(inFile, bufferSize, 0)

	batchCount := 0

	for {
		fileBuffer = fileBufferPages
		// eccBuffer = eccBufferPages
		fRead, feof := fileReader.ReadNext(fileBuffer)
		eRead, eeof := eccReader.ReadNext(eccBuffer) // eRead == len(eccBuffer), else there should be some problem..

		if feof && eeof {
			break
		}

		if feof || eeof {
			fmt.Println("File read error: ecc/data ended earlier than the other")
		}

		if fRead < numData {
			// link to zero page?
			for i:= fRead; i<numData; i++ {
				fileBuffer[i] = zero_page
			}
		}

		if eRead < numRecovery {
			// error
			fmt.Printf("ECC Read Error at chunk %d\n", batchCount*numRecovery+eRead)
			return
		}

		crcs, err := readCRC(crcBuffer, crcFile)
		if err != nil {
			fmt.Println(err)
			return
		}
		if crcs != len(crcBuffer) {
			fmt.Println("Less crc read")
		}
		for i:=0; i<fRead; i++ {
			buf := fileBuffer[i]
			crc := crc32.ChecksumIEEE(buf)
			if crcBuffer[i] != crc {
				fmt.Printf("Data Block %d damaged, has crc %x, expected %x\n", batchCount*numData+i, crc, crcBuffer[i], )
			}
		}

		for i, buf := range eccBuffer {
			crc := crc32.ChecksumIEEE(buf)
			if crcBuffer[i+numData] != crc {
				fmt.Printf("ECC  Block %d damaged, has crc %x, expected %x\n", batchCount*numRecovery+i, crc, crcBuffer[i+numData])
			}
		}

		batchCount += 1
	}

}

func readCRC(buffer []uint32, f *os.File) (int, error) {
	byteBuffer := make([]byte, 4)
	for i, _ := range buffer {
		err := binary.Read(f, binary.LittleEndian, byteBuffer)
		buffer[i] = binary.LittleEndian.Uint32(byteBuffer)
		if err != nil {
			if err != nil {
				return i, err
			}
		}
	}
	return len(buffer), nil
}