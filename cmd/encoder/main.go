package main

import (
	"flag"
	"log"
	"os"
	"alexhalogen/rsfileprotect/internal/types"
	"alexhalogen/rsfileprotect/internal/encoding"
)


var eccName = flag.String("ecc", "", "Filename of generated ecc file")
var blockSize = flag.Int("bs", 4096, "Size of chunks that files are splitted into during reed-solomon encoding")
var level = flag.Int("level", 1, "Number of ecc symbols per 10 data symbols, default 1")
var data = flag.String("data", "", "Required, file to be encoded")
var showHelp = flag.Bool("h", false, "Prints this message")

func mainWithExitCode() (int){

	flag.Parse()

	if !sanitizeArgs(os.Args) {
		printUsage()
		return 1
	}

	dataName := *data
	dataFile, err := os.Open(dataName)

	if err != nil {
		log.Println(err)
		return 1
	}
	defer dataFile.Close()

	eccFile, err := os.OpenFile(*eccName, os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0644)

	if err != nil {
		log.Println(err)
		return 1
	}
	defer eccFile.Close()

	crcFile, err := os.OpenFile((*eccName)+".crc", os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0644)

	if err != nil {
		log.Println(err)
		return 1
	}
	defer crcFile.Close()


	fs, err := dataFile.Stat()
	if err != nil {
		log.Printf("Cannot read stats for %s\n", dataName)
		return 1
	}

	meta := types.Metadata{FileSize: fs.Size(), BlockSize:int32(*blockSize), NumData:10, NumRecovery: uint16(*level)}
	success := encoding.Encode(meta, dataFile, eccFile, crcFile)
	if !success {
		return 1
	}
	return 0
}

func printUsage() {
	log.Println("Command usage:\n  encoder <-data filename> [-ecc filename] [-level lvl]\n")
	flag.PrintDefaults()
}

func main() {
	log.SetFlags(log.Flags() &^ (log.Ldate | log.Ltime))
	os.Exit(mainWithExitCode())
}

func sanitizeArgs(args []string) bool {
	if len(args) < 1 {
		return false
	}
	if *showHelp {
		return false
	}
	if *data == "" {
		return false
	}
	if *eccName == "" {
		newName := *data + ".ecc"
		eccName = &newName
	}

	if *level < 1 || *level > 10 {
		log.Println("Only 1 to 10 symbols are allowed")
		return false
	}

	if *blockSize < 0 {
		log.Println("Chunk size must be a positive integer")
		return false
	}

	return true
}
