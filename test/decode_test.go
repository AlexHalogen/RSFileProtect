package test

import(
	"testing"
	"os"
	"io"
	"path/filepath"
	"io/ioutil"
	"alexhalogen/rsfileprotect/internal/decoding"
	"alexhalogen/rsfileprotect/internal/encoding"
	"alexhalogen/rsfileprotect/internal/types"
)


/*func filterDataDmg(dmgs []decoding.DamageDesc) []int {
	ret = make([]int, )

}*/
func corrupt(contents []byte, pos []int) {
	for _, i := range pos {
		contents[i] ^= 0xFF
	}
}

func corruptFile(f *os.File, pos[]int) {
	for _, i := range pos {
		b:= make([]byte, 1)
		f.ReadAt(b, int64(i))
		b[0] ^= 0xFF
		f.WriteAt(b, int64(i))
	}
}

func corruptChunk(contents []byte, chunks []int, size int) {
	pos := make([]int, len(chunks))
	for i, c := range chunks {
		pos[i] = c * size
	}
}


func makeTestFiles(t *testing.T, meta types.Metadata, dir string, prefix string) ([]byte, *os.File, *os.File, *os.File) {
	
	size := int(meta.FileSize) // not using super big files in testing...
	file, err := os.Create(filepath.Join(dir, prefix+".file"))
	if err != nil {
		t.Fatal("Cannot create file for testing")
	}
	contents := make([]byte, size)

	ir := size / 1024
	jr := 1024

	for i:=0; i<ir; i++ {
		for j:=0; j<jr; j++ {
			if i*jr + j >= size {
				break
			}
			contents[i*1024+j] = byte((i+j)& 0xFF)
		}
	}

	file.Write(contents)
	file.Seek(0,io.SeekStart)

	// encode 
	ecc_n := filepath.Join(dir, prefix+".ecc")
	crc_n := filepath.Join(dir, prefix+".crc")
	ef, err := os.Create(ecc_n)
	if err != nil {
		t.Fatal("Cannot create file for encoding")
	}

	cf, err := os.Create(crc_n)
	if err != nil {
		t.Fatal("Cannot create file for encoding")
	}

	success := encoding.Encode(meta, file, ef, cf)
	if !success {
		t.Error("Encoding failed")
	}

	ef.Seek(0,io.SeekStart)
	cf.Seek(0,io.SeekStart)
	file.Seek(0,io.SeekStart)
	return contents, file, ef, cf
}


func encodeThenDecode(
		t *testing.T, meta types.Metadata, prefix string, 
		dmgPos []int, 
		eccDmgPos []int, 
		expectedDmgs []int,  // dmg sections
		expectedRepairs []int) {

	dir, err := ioutil.TempDir("","")
	size := int(meta.FileSize)
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	contents, file, ef, cf := makeTestFiles(t, meta, dir, prefix)
	defer file.Close()
	defer ef.Close()
	defer cf.Close()

	if len(dmgPos) != 0 || len(eccDmgPos) != 0 {
		// to corrupt the file
		corruptFile(file, dmgPos) // corrupt ecc file directly
		corruptFile(ef, eccDmgPos) // corrupt ecc file directly
		file.Seek(0,io.SeekStart)
		ef.Seek(0,io.SeekStart)
	}

	damages, e := decoding.ScanFile(nil, file, ef, cf) // feed in damaged file
	if e {
		t.Error("Generic error when decoding")
		t.FailNow()
	}

	// filter damages by type
	var dd, ed, ad []int
	for _, d := range damages {
		if len(d.DataDamage) != 0 {
			dd = append(dd, d.Section)
		}

		if len(d.EccDamage) != 0 {
			ed = append(ed, d.Section)
		}
		ad = append(ad, d.Section)
	}

	if !equals(ad, expectedDmgs) {
		t.Errorf("Decoder detected error %v, expecting %v", ad, expectedDmgs)
		t.FailNow()
	}

	if len(dmgPos) != 0 || len(eccDmgPos) != 0 {
		// attempt to repair the file

		file.Seek(0,io.SeekStart)
		ef.Seek(0,io.SeekStart)
		cf.Seek(0,io.SeekStart)
		
		rf, err := os.Create(filepath.Join(dir, prefix+".fixed"))
		if err != nil {
			t.Fatal("Cannot create fixed file")
		}
		repaired, success := decoding.FastRepair(nil, rf, file, ef, damages) // feed in damaged file
		if !equals(repaired, expectedRepairs) {
			t.Log(repaired)
			t.Log(expectedRepairs)
			t.Error("Failed to repair all recoverable damages")
			t.FailNow()
		}

		
		if equals(dd, expectedRepairs) != success { // equals && !success, or !equals && success
			t.Error("Incorrect fastRepair return value")
			t.Log(dd)
			t.Log(expectedRepairs)
			t.Log(success)
			t.FailNow()
		}

		rf.Seek(0, io.SeekStart)
		fs, _ := rf.Stat()
		newSize := fs.Size()
		if newSize != int64(size) {
			t.Errorf("Repaired file size differs: has %d, expected %d", newSize, size)
			t.FailNow()
		}
		rContents := make([]byte, newSize)
		rf.Read(rContents)
		for i := range contents {
			if contents[i] != rContents[i] {
				section := i / (int(meta.BlockSize) * int(meta.NumRecovery))
				for _, v := range(expectedRepairs) {
					if section == v {
						// this damage should be repaired
						t.Errorf("Repaired file content differs at offset %X", i)
						t.FailNow()		
					}
				}
			}
		}
	}
}


func TestDecodeSimple(t *testing.T) {
	t.Run("bs=4096,rs=10-1,size=35M", func(t *testing.T) {
		encodeThenDecode(
			t, 
			types.Metadata{FileSize: 1024*1024*35, BlockSize:4096, NumData:10, NumRecovery:1},
			"simple35m",
			[]int{},
			[]int{},
			[]int{},
			[]int{})
	})

	t.Run("bs=4096,rs=10-1,size=35M+3", func(t *testing.T) {
		encodeThenDecode(
			t, 
			types.Metadata{FileSize: 1024*1024*35+3, BlockSize:4096, NumData:10, NumRecovery:1},
			"simple35m3",
			[]int{},
			[]int{},
			[]int{},
			[]int{})
	})
	t.Run("bs=4096,rs=10-1,size=103B", func(t *testing.T) {
		encodeThenDecode(
			t, 
			types.Metadata{FileSize: 103, BlockSize:4096, NumData:10, NumRecovery:1},
			"simple103",
			[]int{},
			[]int{},
			[]int{},
			[]int{})
	})
}

func TestDecode1Error(t *testing.T) {
	t.Run("bs=4096,rs=10-1,size=35M, e=1", func(t *testing.T) {
		encodeThenDecode(
			t, 
			types.Metadata{FileSize: 1024*1024*35, BlockSize:4096, NumData:10, NumRecovery:1},
			"oneerror35m",
			[]int{36978},
			[]int{},
			[]int{0},
			[]int{0})
	})

	t.Run("bs=4096,rs=10-1,size=35M+3, e=1", func(t *testing.T) {
		encodeThenDecode(
			t, 
			types.Metadata{FileSize: 1024*1024*35+3, BlockSize:4096, NumData:10, NumRecovery:1},
			"oneerror35m3",
			[]int{36978},
			[]int{},
			[]int{0},
			[]int{0})
	})

	t.Run("bs=4096,rs=10-1,size=35M+4k, e=1", func(t *testing.T) {
		encodeThenDecode(
			t, 
			types.Metadata{FileSize: 1024*1024*35+4096, BlockSize:4096, NumData:10, NumRecovery:1},
			"oneerror35m4k",
			[]int{36978},
			[]int{},
			[]int{0},
			[]int{0})
	})
	t.Run("bs=4096,rs=10-1,size=103B, e=1", func(t *testing.T) {
		encodeThenDecode(
			t, 
			types.Metadata{FileSize: 103, BlockSize:4096, NumData:10, NumRecovery:1},
			"oneerror103",
			[]int{100},
			[]int{},
			[]int{0},
			[]int{0})
	})

}

func TestDecodeMultError(t *testing.T) {
	t.Run("bs=4096,rs=10-2,size=35M,e=1d+1e", func(t *testing.T) {
		encodeThenDecode(
			t, 
			types.Metadata{FileSize: 1024*1024*35, BlockSize:4096, NumData:10, NumRecovery:2},
			"multerr35m",
			[]int{36978,36999,40000}, // data chunk #9*3
			[]int{32+1,32+128, 32+3096}, // ecc chunk #0 *3
			[]int{0},
			[]int{0})
	})

	t.Run("bs=4096,rs=10-2,size=35M+4k,e=1d+1e", func(t *testing.T) {
		encodeThenDecode(
			t, 
			types.Metadata{FileSize: 1024*1024*35+4096, BlockSize:4096, NumData:10, NumRecovery:2},
			"multerr35m4k",
			[]int{36978,36999,40000}, // data chunk #9*3
			[]int{32+1,32+128, 32+3096}, // ecc chunk #0 *3
			[]int{0},
			[]int{0})
	})

	t.Run("bs=4096,rs=10-2,size=35M+3,e=1d+1e", func(t *testing.T) {
		encodeThenDecode(
			t, 
			types.Metadata{FileSize: 1024*1024*35+3, BlockSize:4096, NumData:10, NumRecovery:2},
			"multerr35m3",
			[]int{36978,36999,40000}, // data chunk #9*3
			[]int{32+1,32+128, 32+3096}, // ecc chunk #0 *3
			[]int{0},
			[]int{0})
	})

	t.Run("bs=4096,rs=10-2,size=103B,e=1d+1e", func(t *testing.T) {
		encodeThenDecode(
			t, 
			types.Metadata{FileSize: 103, BlockSize:4096, NumData:10, NumRecovery:2},
			"multerr103",
			[]int{100,88,79,90,77,88},
			[]int{32+1,32+2,32+3},
			[]int{0},
			[]int{0})
	})

	t.Run("bs=4096,rs=10-1,size=35M,e=2d+1e", func(t *testing.T) {
		encodeThenDecode(
			t, 
			types.Metadata{FileSize: 1024*1024*35, BlockSize:4096, NumData:10, NumRecovery:1},
			"multerr35m",
			[]int{36978,36999,40000,
				  368640, 368840}, // data chunk #9*3
			[]int{32+8192,32+8200, 32+11111}, // ecc chunk #0 *3
			[]int{0,2,9},
			[]int{0,9})
	})

	t.Run("bs=4096,rs=10-1,size=35M+4K,e=2d+1e", func(t *testing.T) {
		encodeThenDecode(
			t, 
			types.Metadata{FileSize: 1024*1024*35+4096, BlockSize:4096, NumData:10, NumRecovery:1},
			"multerr35m",
			[]int{36978,36999,40000,
				  368640, 368840}, // data chunk #9*3
			[]int{32+8192,32+8200, 32+11111}, // ecc chunk #0 *3
			[]int{0,2,9},
			[]int{0,9})
	})

}

func TestDecodeTooManyError(t *testing.T) {
	t.Run("bs=4096,rs=10-1,size=35M,e=2d+2e", func(t *testing.T) {
		encodeThenDecode(
			t, 
			types.Metadata{FileSize: 1024*1024*35, BlockSize:4096, NumData:10, NumRecovery:1},
			"multerr35m",
			[]int{36978,36999,40000,
				  368640, 368840}, // data chunk #9*3
			[]int{32+8192,32+8200, 32+40000}, // ecc chunk #0 *3
			[]int{0,2,9},
			[]int{0})
	})

	t.Run("bs=4096,rs=10-1,size=35M+4K,e=2d+2e", func(t *testing.T) {
		encodeThenDecode(
			t, 
			types.Metadata{FileSize: 1024*1024*35+4096, BlockSize:4096, NumData:10, NumRecovery:1},
			"multerr35m+4",
			[]int{36978,36999,40000,
				  368640, 368840}, // data chunk #9*3
			[]int{32+8192,32+8200, 32+11111, 32+40000}, // ecc chunk #0 *3
			[]int{0,2,9},
			[]int{0})
	})
}

