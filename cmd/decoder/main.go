package main

import (
	"flag"
	"log"
	"os"
	"alexhalogen/rsfileprotect/internal/decoding"
	// "alexhalogen/rsfileprotect/internal/filehelper"
	// "alexhalogen/rsfileprotect/internal/types"
)


var eccName = flag.String("ecc", "", "ecc file containing code needed to restore file")
var crcName = flag.String("crc", "", "crc file for quick integrity check and restoration")

func main() {
	flag.Parse()
	args := flag.Args()

	dataFileName := args[0]

	dataFile, err := os.Open(dataFileName)
	if err != nil {
		log.Println(err)
		return
	}
	defer dataFile.Close()

	eccFile, err := os.Open(*eccName)
	if err != nil {
		log.Println(err)
		return
	}
	defer eccFile.Close()

	var crcFile *os.File
	if crcName != nil {
		crcFile, err = os.Open(*crcName)
		if err != nil {
			log.Println(err)
			return
		}
	}

	log.Printf("Data: %s, ECC: %s, CRC: %s\n", dataFileName, *eccName, *crcName)

	
	damages, failed := decoding.ScanFile(nil, dataFile, eccFile, crcFile)
	if failed {
		log.Printf("Severe error prevented repair of file %s\n", dataFileName)
		return
	}

	if len(damages) > 0 {
		dataFile.Seek(0,0)
		eccFile.Seek(0,0)
		outFile, err := os.OpenFile(dataFileName+".fix", os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Printf("Failed to open %s for repair\n", dataFileName+".fix")

		}

		repaired, success := decoding.FastRepair(nil, outFile, dataFile, eccFile, damages)
		if success {
			log.Printf("Successfully repaired %s\n", dataFileName)
		} else {
			log.Printf("File reconstruction failed, partial result saved")
			log.Printf("Repaired sections: %v", repaired)
		}
	}


}
