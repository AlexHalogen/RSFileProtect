package main

import (
	"flag"
	"fmt"
	"os"
	"math"
	"github.com/klauspost/reedsolomon"
    "hash/crc32"
    "encoding/binary"
	"alexhalogen/rsfileprotect/internal/filehelper"
	"alexhalogen/rsfileprotect/internal/types"
)

type damageDesc struct {
	Section int
	DataDamage []int
	EccDamage []int
}
var eccName = flag.String("ecc", "", "ecc file containing code needed to restore file")
var crcName = flag.String("crc", "", "crc file for quick integrity check and restoration")

func main() {
	flag.Parse()
	args := flag.Args()

	dataFileName := args[0]

	dataFile, err := os.Open(dataFileName)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer dataFile.Close()

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

	fmt.Printf("Data: %s, ECC: %s, CRC: %s\n", dataFileName, *eccName, *crcName)

	var meta types.Metadata;
	err = filehelper.ReadMeta(eccFile, &meta)
	if err != nil {
		fmt.Println("Failed to read metadata from ecc file!")
		return
	}

	fmt.Println(meta)
	damages, failed := scanFile(meta, dataFile, eccFile, crcFile)
	if failed {
		fmt.Printf("Severe error prevented repair of file %s\n", dataFileName)
		return
	}

	if len(damages) > 0 {
		dataFile.Seek(0,0)
		eccFile.Seek(0,0)
		outFile, err := os.OpenFile(dataFileName+".fix", os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			fmt.Printf("Failed to open %s for repair\n", dataFileName+".fix")

		}
		filehelper.ReadMeta(eccFile, &meta) // skip the meta part without using unsafe methods
		success := fastRepair(meta, outFile, dataFile, eccFile, damages)
		if success {
			fmt.Printf("Successfully repaired %s\n", dataFileName)
		} else {
			fmt.Printf("File reconstruction failed, partial result saved")
		}
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

func scanFile(meta types.Metadata, dataFile *os.File, eccFile *os.File, crcFile *os.File) ([]damageDesc, bool){	
	err := false
	damages := make([]damageDesc, 0, 8)
	// fileSize := meta.FileSize
	numData := (int)(meta.NumData)
	numRecovery := (int)(meta.NumRecovery)
	bufferSize := (int)(meta.BlockSize)

	fileBufferPages := make([][]byte, numData);
	eccBuffer := make([][]byte, numRecovery)
	crcBuffer := make([]uint32, numData+numRecovery)
	fileBuffer := make([][]byte, numData)
	zero_page := make([]byte, bufferSize)
	// filehelper.Memset(zero_page, 0, bufferSize, 0)

	for i, _ := range fileBuffer {
		fileBufferPages[i] = make([]byte, bufferSize)
	}
	for i, _ := range eccBuffer {
		eccBuffer[i] = make([]byte, bufferSize)
	}
	eccReader := filehelper.NewChunkedReader(eccFile, bufferSize, 0)
	fileReader := filehelper.NewChunkedReader(dataFile, bufferSize, 0)
	batchCount := 0

	for {
		fileBuffer = fileBufferPages
		fRead, feof := fileReader.ReadNext(fileBuffer)
		eRead, eeof := eccReader.ReadNext(eccBuffer) // eRead == len(eccBuffer), else there should be some problem..

		if feof && eeof {
			break
		}

		if feof || eeof {
			fmt.Println("File read error: ecc/data ended earlier than the other")
			err = true
			// return // ?
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
			err = true
			return damages, err
		}

		crcs, err := readCRC(crcBuffer, crcFile)
		if err != nil {
			fmt.Println(err)
			return damages, true
		}
		if crcs != len(crcBuffer) {
			fmt.Println("Less crc read")
		}

		dDamages := make([]int, 0,2)
		eDamages := make([]int, 0,2)

		for i:=0; i<fRead; i++ {
			buf := fileBuffer[i]
			crc := crc32.ChecksumIEEE(buf)
			if crcBuffer[i] != crc {
				idx := batchCount*numData+i
				fmt.Printf("Data Block %d damaged, has crc %x, expected %x\n", idx, crc, crcBuffer[i], )
				dDamages = append(dDamages, i)
			}
		}


		for i, buf := range eccBuffer {
			crc := crc32.ChecksumIEEE(buf)
			if crcBuffer[i+numData] != crc {
				idx := batchCount*numRecovery+i
				fmt.Printf("ECC  Block %d damaged, has crc %x, expected %x\n", idx, crc, crcBuffer[i+numData])
				eDamages = append(eDamages, i)
			}
		}

		if len(dDamages) > 0 {
			damages = append(damages, damageDesc{batchCount, dDamages, eDamages})
		}
		batchCount += 1
	}

	return damages, err
}

func fastRepair(meta types.Metadata, outFile *os.File, dataFile *os.File, eccFile *os.File, damages []damageDesc) (bool) {
	numData := int(meta.NumData)
	fileSize := int(meta.FileSize)
	numRecovery := int(meta.NumRecovery)
	eccReader := filehelper.NewChunkedReader(eccFile, int(meta.BlockSize), 0)
	fileReader := filehelper.NewChunkedReader(dataFile, int(meta.BlockSize), 0)
	blockSize := int(meta.BlockSize)
	zero_page := make([]byte, blockSize)
	success := false

	fileBufferPages := make([][]byte, meta.NumData)
	for i := range fileBufferPages {
		fileBufferPages[i] = make([]byte, blockSize)
	}
	eccBufferPages := make([][]byte, numRecovery)
	for i := range eccBufferPages {
		eccBufferPages[i] = make([]byte, blockSize)
	}

	enc, _ := reedsolomon.New(numData, numRecovery)

	cur := 0
	eof := false
	sections := int(math.Ceil(float64(fileSize)/float64(numData)))
	for i:=0; i<sections; i++ {

		fileBuffer := fileBufferPages
		eccBuffer := eccBufferPages
		
		var chunksRead int
		chunksRead, eof = fileReader.ReadNext(fileBuffer)
		if eof {
			break
		}

		if cur < len(damages) && i == damages[cur].Section { // damage with in this range
			dmg := damages[cur]
			cur++
			_, eof := eccReader.ReadNext(eccBuffer)
			if eof {
				fmt.Println("EOF during read to ecc file\n")
				success = false
				break
			}
			totalDmg := len(dmg.DataDamage) + len(dmg.EccDamage)
			if totalDmg > numRecovery {
				fmt.Println("Failed to repair block %d-%d due to too many damages\n", i*numData, (i+1)*numData)
				success = false
				for i := range fileBuffer {
					fileBuffer[i] = zero_page
				}
			} else {
				// repair
				for _,d := range dmg.DataDamage {
					fileBuffer[d] = nil
				}
				for _,d := range dmg.EccDamage {
					eccBuffer[d] = nil
				}

				fileBuffer = append(fileBuffer, eccBuffer...)
				enc.Reconstruct(fileBuffer)
				ok, err := enc.Verify(fileBuffer)
				if !ok || err != nil {
					fmt.Println("Reconstruction failed unexpectedly at block %d-%d\n", i, i+numData)
					success = false
				}	
			}

		} else { // no damage occured within the range, skip a section of ecc file		
			eccReader.SkipNext(numRecovery, blockSize)
		}
		for j:=0; j<chunksRead; j++ {
			_, err := outFile.Write(fileBuffer[j])
			if err != nil {
				fmt.Println(err)
			}
		}
	}

	size := meta.FileSize
	if size % int64(blockSize) != 0 { // truncate the end of file	
		outFile.Truncate(size)
	}
	success = true
	return success
}