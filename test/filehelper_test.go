package main

import (
	"io/ioutil"
	"path/filepath"
	"os"
	"testing"
	"alexhalogen/rsfileprotect/internal/filehelper"
)


// return 0 if ok; 1 if error in non-zero region; -1 if in zero-padded region
func comp(c int, cs int, buffer [][]byte, orig []byte, offset *int) int {
	for i:=0; i<c; i++ {
		for j:=0; j<cs; j++ {
			idx := *offset+j
			if idx < len(orig) {
				if buffer[i][j] != orig[idx] {
					return 1
				}	
			} else {
				if buffer[i][j] != 0 {
					return -1
				}

			}
		}
		(*offset) += cs
	}
	return 0
}

func readAndCompare(t *testing.T, f *os.File, chunkSize int, count int, orig []byte) {
	reader := filehelper.NewChunkedReader(f, chunkSize, 0)
	buffer := make([][]byte, count)
	for i := range buffer {
		buffer[i] = make([]byte, chunkSize)
	}

	total := 0

	for  {
		c, eof := reader.ReadNext(buffer)
		if eof {
			break
		}
		r := comp(c, chunkSize, buffer, orig, &total)
		if r == 1 {
			t.Error("Content mismatch")
			t.FailNow()
		} else if r == -1 {
			t.Error("Content mismatch in zero-padded region")
			t.FailNow()
		}

	}
	f.Seek(0,0)
	return
}

func TestRead(t *testing.T) {
	
	// dir := t.TempDir()
	dir, err := ioutil.TempDir("","")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	contents := make([]byte, 1024)
	for i:=0; i<64; i++ {
		for j:=0; j<16; j++ {
			contents[i*16+j] = byte((i+j) & 0xFF)
		}
	}

	fn := filepath.Join(dir, "readbase.file")
	basef, err := os.Create(fn)
	
	if err != nil {
		t.Fatal("Failed to create file for testing, ", err)
	}
	
	basef.Write(contents)

	t.Run("bs=32,c=16", func (t *testing.T) {readAndCompare(t, basef, 32,16, contents)})
	t.Run("bs=33,c=16", func (t *testing.T) {readAndCompare(t, basef, 33,16, contents)})
	t.Run("bs=33,c=28", func (t *testing.T) {readAndCompare(t, basef, 33,28, contents)})
	t.Run("bs=1024,c=1", func (t *testing.T) {readAndCompare(t, basef, 1024,1, contents)})
	t.Run("bs=1024,c=2", func (t *testing.T) {readAndCompare(t, basef, 1024,2, contents)})
	t.Run("bs=1026,c=1", func (t *testing.T) {readAndCompare(t, basef, 1026,1, contents)})
	t.Run("bs=1026,c=2", func (t *testing.T) {readAndCompare(t, basef, 1026,2, contents)})
	t.Run("bs=32,c=10", func (t *testing.T) {readAndCompare(t, basef, 32,10, contents)})
	t.Run("bs=1,c=1", func (t *testing.T) {readAndCompare(t, basef, 1,1, contents)})
	
}
