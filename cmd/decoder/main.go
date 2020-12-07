package main

import (
	"flag"
	"log"
	"os"
	"alexhalogen/rsfileprotect/internal/decoding"
	"alexhalogen/rsfileprotect/internal/filehelper"
	"alexhalogen/rsfileprotect/internal/types"
	"alexhalogen/rsfileprotect/internal/cmdparser"
	"fmt"
)


var autoSet = flag.NewFlagSet("a", flag.ContinueOnError)
var manualSet = flag.NewFlagSet("m", flag.ContinueOnError)
var scanSet = flag.NewFlagSet("s", flag.ContinueOnError)

var showHelp bool
var eccName string
var crcName string
var dataName string
var eccDmgIdxs, dataDmgIdxs string

var eccDmgIdx []int
var dataDmgIdx []int
var output string


func mainWithExitCode() (int){
	initCmds()

	if len(os.Args) < 5  { // exec, action, ecc, data, crc
		printUsage()
		return 1
	}
	action := os.Args[1]
	if !sanitizeInput(action) {
		printUsage()
		return 1
	}

	if showHelp {
		printUsage()
		return 1
	}

	dataFile, err := os.Open(dataName)
	if err != nil {
		log.Println(err)
		return 1
	}
	defer dataFile.Close()

	eccFile, err := os.Open(eccName)
	if err != nil {
		log.Println(err)
		return 1
	}
	defer eccFile.Close()

	crcFile, err := os.Open(crcName)
	if err != nil {
		log.Println(err)
		return 1
	}

	log.Printf("Data: %s, ECC: %s, CRC: %s\n", dataName, eccName, crcName)
	meta := readMeta(eccFile)
	if meta == nil {
		return 1
	}
	log.Printf("Metadata: File Size: %d, Chunk size: %d, #Data: %d, #Recovery: %d", meta.FileSize, meta.BlockSize, meta.NumData, meta.NumRecovery)

	var damages []decoding.DamageDesc
	if action == "m" {
		damages = cmdparser.CSVToDamage(meta, dataDmgIdx, eccDmgIdx)
	} else {
		var failed bool
		damages, failed = decoding.ScanFile(nil, dataFile, eccFile, crcFile)
		if failed {
			log.Printf("Severe error prevented repair of file %s\n", dataName)
			return 1
		}
	}

	if action == "s" {
		sd, se := cmdparser.DamageToCSV(damages, meta)
		if len(*sd) != 0 || len(*se) != 0 {
			fmt.Printf("%s: Data=[%s] ECC=[%s]\n", dataName, *sd, *se)	
		}
	}


	if len(damages) > 0 {
		dataFile.Seek(0,0)
		eccFile.Seek(0,0)

		if action == "r" || action == "m" {
			outFile, err := os.OpenFile(output, os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				log.Printf("Failed to open %s for repair\n", output)
			}

			repaired, success := decoding.FastRepair(nil, outFile, dataFile, eccFile, damages)
			if success {
				log.Printf("Successfully repaired %s\n", dataName)
			} else {
				log.Printf("File reconstruction failed, partial result saved")
				log.Printf("Repaired sections: %v", repaired)
			}
		}

	}
	return 0
}

func readMeta(eccFile *os.File) *types.Metadata {
	var fmeta types.Metadata;
	metaErr := filehelper.ReadMeta(eccFile, &fmeta)
	if metaErr != nil {
		log.Println(metaErr)
		log.Println("Failed to read metadata from ecc file!")
		return nil
	}
	eccFile.Seek(0,0)
	return &fmeta // ok to do this in go...
}



func initCmds() {
	for _, s := range []*flag.FlagSet{autoSet, manualSet, scanSet} {
		s.StringVar(&eccName, "ecc", "", "required, ecc file containing code needed to restore file")
		s.StringVar(&crcName, "crc", "", "required, crc file for quick integrity check and restoration")
		s.StringVar(&dataName,"data", "", "required,  file needed to be verified or repaired")
		s.BoolVar(&showHelp, "h", false, "Prints this help message")
		cs := s // capture value in closure
		cs.Usage = func() {
			fmt.Fprintf(cs.Output(), "\nArguments for action %s:\n", cs.Name())
			cs.PrintDefaults()
		}
	}

	autoSet.StringVar(&output, "out", "", "required, file name of repaired file")

	manualSet.StringVar(&output, "out", "", "required, file name of repaired file")
	manualSet.StringVar(&eccDmgIdxs, "edmg", "", "required, chunk indices of ecc damages, comma-separated list quoted in square brackets, e.g [1,15,69]")
	manualSet.StringVar(&dataDmgIdxs, "ddmg", "", "required, chunk indices of data damages, comma-separated list quoted in square brackets, e.g [1,15,69]")

}


func sanitizeInput(action string) bool {

	var err error
	switch action {
		case "a": // auto repair
			err = autoSet.Parse(os.Args[2:])
			if err != nil {
				return false
			}
			if output == "" { // no output file
				return false
			}

		case "m": // manual repair, damage positions needed

			err = manualSet.Parse(os.Args[2:])

			if err != nil {
				return false
			}

			if eccDmgIdxs == "" && dataDmgIdxs == "" || output == "" { // both missing
				return false
			}

			eccDmgIdx = cmdparser.CSVToIntArr(eccDmgIdxs)
			dataDmgIdx = cmdparser.CSVToIntArr(dataDmgIdxs)

			if eccDmgIdx == nil || dataDmgIdx == nil {
				log.Println("Error parsing damage positions")
				return false
			}

		case "s": // scan only
			err = scanSet.Parse(os.Args[2:])
			if err != nil {
				return false
			}

		default:
			log.Printf("Unsupported action %s\n", action)
			return false
	}
	return true
}



func printUsage() {
	output := autoSet.Output()

	fmt.Fprintf(output, "\nCommand usage:\n  %s <action> <Args...> [-h]\n", "decoder")
	fmt.Fprintf(output, "\nActions: \n")
	fmt.Fprintf(output, "  a  Automatically scan and repairs the file if damaged\n")
	fmt.Fprintf(output, "  s  Scan the file and report damaged chunks in formats tha can be used for manual repairs; Reports nothing to stdout if no errors were found\n")
	fmt.Fprintf(output, "  m  Repair damaged file with user-provided damage positions\n")

	autoSet.Usage()
	scanSet.Usage()
	manualSet.Usage()
}

func main() {
	log.SetFlags(log.Flags() &^ (log.Ldate | log.Ltime))
	os.Exit(mainWithExitCode())
}