package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/klauspost/reedsolomon"
)


var metaName = flag.String("meta", "", "Metadata file containing block info")
var recName = flag.String("ecc", "", "ecc file containing reed-solomon code")


func main() {
	flag.Parse()
	args := flag.Args()

	inFile := args[0]

	numChunks := 10
	numRecovery := 1
	bufferSize := 4096
	buffer := make([][]byte, numChunks+numRecovery);
	for arr := range buffer {
		buffer[arr] = make([]byte, bufferSize)
	}

	enc, err := reedsolomon.New(numChunks, numRecovery)

}