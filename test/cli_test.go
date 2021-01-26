package test

import (
	"os"
	"os/exec"
	"testing"
	"io/ioutil"
	"path/filepath"
)

type switches struct {
	encode bool
	action string
	in string
	ecc string // encode & decode
	crc string
	out string
	ddmg string
	edmg string

	bs string // encode only
	level string // encode only
}


func TestArgs(t *testing.T) {

	type test struct {
		swt switches
		rc int
		want bool
	}

	dir, fn, en, cn := makeFileAndNames(t, 35*1024*1024)
	defer os.RemoveAll(dir)
	
	tests := []test {
		{switches{encode:true, in:fn, ecc:en, crc:cn}, 0, true},
		{switches{encode:true, in:fn}, 0, true},
		{switches{encode:true, ecc:en, crc:cn}, 0, false},
		{switches{encode:true, in:fn, ecc:en, crc:cn, bs:"4096", level:"2"}, 0, true},
		{switches{encode:true, in:fn, ecc:en, crc:cn, bs:"4096", level:"23"}, 0, false},
		{switches{encode:true, in:fn, ecc:en, crc:cn, bs:"409-6", level:"2"}, 0, false},
		{switches{encode:false, in:fn, ecc:en, crc:cn, action:"s"}, 0, true},
	}
	
	for _, c := range tests {
		assert(t, runOne(t, c.swt, c.rc), c.want)
	}
}


func (s *switches)makeArgs() []string {

	var args []string
	if s.encode {
		if s.in != "" {
			args = append(args, "-data", s.in)
		}
		if s.level != "" {
			args = append(args, "-level", s.level)
		}
		if s.bs != "" {
			args = append(args, "-bs", s.bs)
		}
		
		if s.ecc != "" {
			args = append(args, "-ecc", s.ecc)
		}

	} else {
		if s.action != "" {
			args = append(args, s.action)
		}
		if s.in != "" {
			args = append(args, "-data", s.in)
		}
		if s.ecc != "" {
			args = append(args, "-ecc", s.ecc)
		}
		if s.crc != "" {
			args = append(args, "-crc", s.crc)
		}
		if s.out != "" {
			args = append(args, "-out", s.out)
		}
		if s.ddmg != "" {
			args = append(args, "-ddmg", s.ddmg)
		}
		if s.edmg != "" {
			args = append(args, "-edmg", s.edmg)
		}
	}
	return args

}


func makeFileAndNames(t *testing.T, size int) (string, string, string, string){
	
	dir, err := ioutil.TempDir("","")
	if err != nil {
		t.Fatal(err)
	}
	contents := make([]byte, size)
	for i:=0; i<size; i++ {
		contents[i] = byte( i ^ 0x19)
	}
	fn := filepath.Join(dir, "test.file")
	en := filepath.Join(dir, "test.ecc")
	cn := filepath.Join(dir, "test.ecc.crc")

	f,err := os.Create(fn)
	if err != nil {
		t.Fatal("Cannot create file for testing")
		t.FailNow()
	}
	f.Write(contents)
	f.Close()
	return dir, fn, en, cn
}

func runOne(t *testing.T, swt switches, rc int) bool {
	
	args := swt.makeArgs()
	var exe string
	if swt.encode {
		exe = "../encoder"
	} else {
		exe = "../decoder"
	}
	cmd := exec.Command(exe, args...)
	t.Log(cmd.String())
	output, err := cmd.CombinedOutput()

	if err != nil {
		if rc == 0 {
			/*t.Error(err)
			t.Errorf("%s\n", output)*/
			t.Logf("%s\n", output)
			/*t.FailNow()*/
			return false
		} else {
			if exitError, ok := err.(*exec.ExitError); ok {// https://stackoverflow.com/a/55055100
				code := exitError.ExitCode() 
				if code != rc {
					/*t.Errorf("Expected exit code %d, has %d\n", rc, code)
					t.FailNow()*/
					t.Logf("%s\n", output)
					return false
				}
			}
		}	
	}
	
	if err == nil  && rc != 0 {
		t.Errorf("Expected exit code %d, has 0\n", rc)
		t.FailNow()
	}

	return true
}


func assert(t *testing.T, rc, exp bool) {
	if rc != exp {
		t.FailNow()
	}
}
