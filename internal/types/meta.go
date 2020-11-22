package types

type Metadata struct {
	FileSize 		uint64 // file size 
	BlockSize 		uint32 // size of each block
	NumData 		uint16 // number of data chunks in one iteration
	NumRecovery 	uint16 // number of ecc chunks in one iteration
	Ecc				[16]byte // ecc code for above data
}