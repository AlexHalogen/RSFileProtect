package main

import (
	"flag"
	"fmt"
	"os"
	"alexhalogen/rsfileprotect/internal/types"
	"alexhalogen/rsfileprotect/internal/encoding"
)


var outName = flag.String("out", "", "Output name")

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


	fs, err := inFile.Stat()
	if err != nil {
		fmt.Printf("Cannot read stats for %s\n", inName)
	}

	meta := types.Metadata{FileSize: fs.Size(), BlockSize:4096, NumData:10, NumRecovery: 1}
	encoding.Encode(meta, inFile, eccFile, crcFile)

}


