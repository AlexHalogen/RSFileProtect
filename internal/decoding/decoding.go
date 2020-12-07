/*The MIT License (MIT)

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
package decoding

import (
	"os"
	"log"
	"math"
	"github.com/klauspost/reedsolomon"
    "hash/crc32"
	"alexhalogen/rsfileprotect/internal/filehelper"
	"alexhalogen/rsfileprotect/internal/types"
)

type DamageDesc struct {
	Section int
	DataDamage []int
	EccDamage []int
}

func ScanFile(meta *types.Metadata, dataFile *os.File, eccFile *os.File, crcFile *os.File) ([]DamageDesc, bool){	
	err := false
	damages := make([]DamageDesc, 0, 8)
	
	// seek file past meta area without using unsafe methods
	var fmeta types.Metadata;
	metaErr := filehelper.ReadMeta(eccFile, &fmeta)
	if metaErr != nil {
		log.Println(metaErr)
		log.Println("Failed to read metadata from ecc file!")
		return damages, true
	}
	// trust metadata read from file if not specified in parameters
	if meta == nil {
		meta = &fmeta
	}

	numData := (int)(meta.NumData)
	numRecovery := (int)(meta.NumRecovery)
	bufferSize := (int)(meta.BlockSize)

	fileBufferPages := make([][]byte, numData);
	eccBuffer := make([][]byte, numRecovery)
	crcBuffer := make([]uint32, numData+numRecovery)
	fileBuffer := make([][]byte, numData)
	zero_page := make([]byte, bufferSize)
	

	for i, _ := range fileBuffer {
		fileBufferPages[i] = make([]byte, bufferSize)
	}
	for i, _ := range eccBuffer {
		eccBuffer[i] = make([]byte, bufferSize)
	}
	eccReader := filehelper.NewChunkedReader(eccFile, bufferSize, 0)
	fileReader := filehelper.NewChunkedReader(dataFile, bufferSize, 0)
	crcReader := filehelper.NewCRCReader(crcFile, 0)
	batchCount := 0

	for {
		fileBuffer = fileBufferPages
		fRead, feof := fileReader.ReadNext(fileBuffer)
		eRead, eeof := eccReader.ReadNext(eccBuffer) // eRead == len(eccBuffer), else there should be some problem..

		if feof && eeof {
			break
		}

		if feof || eeof {
			log.Println("File read error: ecc/data ended earlier than the other")
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
			log.Printf("ECC Read Error at chunk %d\n", batchCount*numRecovery+eRead)
			err = true
			return damages, err
		}

		crcs, err := crcReader.ReadNext(crcBuffer)
		if err != nil {
			log.Println(err)
			return damages, true
		}
		if crcs != len(crcBuffer) {
			log.Println("Less crc read")
		}

		dDamages := make([]int, 0,2)
		eDamages := make([]int, 0,2)

		for i:=0; i<fRead; i++ {
			buf := fileBuffer[i]
			crc := crc32.ChecksumIEEE(buf)
			if crcBuffer[i] != crc {
				idx := batchCount*numData+i
				log.Printf("Data Block %d damaged, has crc %x, expected %x\n", idx, crc, crcBuffer[i])
				dDamages = append(dDamages, i)
			}
		}


		for i, buf := range eccBuffer {
			crc := crc32.ChecksumIEEE(buf)
			if crcBuffer[i+numData] != crc {
				idx := batchCount*numRecovery+i
				log.Printf("ECC  Block %d damaged, has crc %x, expected %x\n", idx, crc, crcBuffer[i+numData])
				eDamages = append(eDamages, i)
			}
		}

		if len(dDamages) > 0 || len(eDamages) > 0 {
			damages = append(damages, DamageDesc{batchCount, dDamages, eDamages})
		}
		batchCount += 1
	}

	return damages, err
}


/**
 * Fast repair by setting damaged chunks to nil;
 * return location of repaired sections and whether all damages have been repaired
 */
func FastRepair(meta *types.Metadata, outFile *os.File, dataFile *os.File, eccFile *os.File, damages []DamageDesc) ([]int, bool) {
	success := true
	repaired := make([]int, 0, len(damages))

	// seek file past meta area without using unsafe methods
	var fmeta types.Metadata;
	metaErr := filehelper.ReadMeta(eccFile, &fmeta)
	if metaErr != nil {
		log.Println(metaErr)
		log.Println("Failed to read metadata from ecc file!")
		return repaired, success
	}
	// trust metadata read from file if not specified in parameters
	if meta == nil {
		meta = &fmeta
	}

	numData := int(meta.NumData)
	fileSize := int(meta.FileSize)
	numRecovery := int(meta.NumRecovery)
	eccReader := filehelper.NewChunkedReader(eccFile, int(meta.BlockSize), 0)
	fileReader := filehelper.NewChunkedReader(dataFile, int(meta.BlockSize), 0)
	blockSize := int(meta.BlockSize)
	zero_page := make([]byte, blockSize)

	fileBufferPages := make([][]byte, numData)
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
	sections := int(math.Ceil(float64(fileSize)/float64(numData*blockSize)))
	fileBuffer := make([][]byte, numData)
	eccBuffer := make([][]byte, numRecovery)
	for i:=0; i<sections; i++ {
		copy(fileBuffer, fileBufferPages)
		copy(eccBuffer, eccBufferPages)
		
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
				log.Println("EOF during read to ecc file\n")
				success = false
				break
			}

			totalDmg := len(dmg.DataDamage) + len(dmg.EccDamage)
			if len(dmg.DataDamage) == 0 {
				// only ecc damage, no need to repair
			} else if totalDmg > numRecovery {
				log.Printf("Failed to repair block %d-%d due to too many damages\n", i*numData, (i+1)*numData)
				success = false
				for i := range fileBuffer {
					fileBuffer[i] = zero_page
				}
			} else {
				// necessary and able to repair
				for _,d := range dmg.DataDamage {
					fileBuffer[d] = nil
				}
				for _,d := range dmg.EccDamage {
					eccBuffer[d] = nil
				}

				repairBuffer := make([][]byte, numData+numRecovery)
				copy(repairBuffer, fileBuffer)
				copy(repairBuffer[numData:], eccBuffer)
				enc.Reconstruct(repairBuffer)
				ok, err := enc.Verify(repairBuffer)
				if !ok || err != nil {
					log.Printf("Reconstruction failed unexpectedly at block %d-%d\n", i, i+numData)
					success = false
				}
				copy(fileBuffer, repairBuffer[:numData]) // copy back repaired chunks for writing
				repaired = append(repaired, dmg.Section)
			}

		} else { // no damage occured within the range, skip a section of ecc file		
			eccReader.SkipNext(numRecovery, blockSize)
		}

		for j:=0; j<chunksRead; j++ {
			_, err := outFile.Write(fileBuffer[j])
			if err != nil {
				log.Println(err)
			}
		}
	}

	size := meta.FileSize
	if size % int64(blockSize) != 0 { // truncate the end of file	
		outFile.Truncate(size)
	}
	return repaired, success
}