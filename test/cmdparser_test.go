package main

import (
	"testing"
	"strings"
	"strconv"
	"alexhalogen/rsfileprotect/internal/types"
	"alexhalogen/rsfileprotect/internal/decoding"
	"alexhalogen/rsfileprotect/internal/cmdparser"
	"fmt"
	"math/rand"
	"time"
)

func TestMalformedArgs(t *testing.T) {
	type test struct {
		input string
		want []int
	}
	
	tests := []test{ 
		{input: "[", want: nil},
		{input: "]", want: nil},
		{input: "123]", want: nil},
		{input: "[a", want: nil},
		{input: "", want: nil},
		{input: "123", want: nil},
		{input: "deadfeeb", want: nil},
		{input: "deadfXXb", want: nil},
		{input: "[1,]", want: nil},
		{input: "[1,X]", want: nil},
		{input: "[1;2]", want: nil},
		{input: "[1,2,3,]", want: nil},
		{input: "[,2,3]", want: nil},
		{input: "[,2,3,6,]", want: nil},
		{input: "[1,2,3,4.598]", want: nil},
		{input: "[-1, 2]", want: nil},
		{input: "[1，2，3]", want: nil}, // unicode comma
		{input: "[1,2,3,4,5/]", want: nil}, // unicode comma
		{input: "[1,　2,　3]", want: []int{1,2,3}}, // unicode full-width spaces
		{input: "[1, 2]", want: []int{1,2}},
	}

	for _, c := range tests {
		res := cmdparser.CSVToIntArr(c.input)
		if !equals(res, c.want) {
			t.Error("Result differs")
			fmt.Printf("Want: %v", c.want)
			fmt.Printf("Has:  %v", res)
			t.FailNow()
		}
	}
}

func TestDamageCSV(t *testing.T) {
	t.Run("rs=10,1", func(t *testing.T) { // data ends early
		testDmgCSVHelper(t, 10, 1, "0|1,2,3|; 1||0; 3|8,9|0; 19||0")
	})
	t.Run("rs=10,1", func(t *testing.T) { // data ends early
		testDmgCSVHelper(t, 10, 1, "0||0; 1||0; 3|8,9|0; 19||0")
	})
	t.Run("rs=10,1", func(t *testing.T) { // ecc ends early
		testDmgCSVHelper(t, 10, 1, "0|1,2,3|; 1||0; 3|8,9|0; 21|1,3|")
	})
	t.Run("rs=10,1", func(t *testing.T) { // ecc ends early
		testDmgCSVHelper(t, 10, 1, "0||0; 1|1,2,3|0; 3|8,9|0; 21|1,3|")
	})
	t.Run("rs=10,1", func(t *testing.T) { // only data
		testDmgCSVHelper(t, 10, 1, "0|1,2,3|; 1|8|; 3|8,9|; 13||0")
	})
	t.Run("rs=10,1", func(t *testing.T) { // only ecc
		testDmgCSVHelper(t, 10, 1, "0||0; 1||0; 3||0; 13||0")
	})

	t.Run("rs=11,2", func(t *testing.T) { // data ends early
		testDmgCSVHelper(t, 11, 2, "0|1,2,3|; 1||0; 3|8,9|0; 19||1")
	})
	t.Run("rs=10,1", func(t *testing.T) { // data ends early
		testDmgCSVHelper(t, 11, 2, "0||1; 1||0; 3|8,9|0; 19||0")
	})
	t.Run("rs=11,2", func(t *testing.T) { // ecc ends early
		testDmgCSVHelper(t, 11, 2, "0||0; 1|1,2,3|0; 3|8,9|0; 21|1,3|")
	})
	t.Run("rs=11,2", func(t *testing.T) { // ecc ends early
		testDmgCSVHelper(t, 11, 2, "0|1,2,3|; 1||0; 3|8,9|0; 21|1,3|")
	})
	t.Run("rs=11,2", func(t *testing.T) { // only data
		testDmgCSVHelper(t, 11, 1, "0|1,2,3|; 1|8|; 3|8,9|; 13||0")
	})
	t.Run("rs=11,2", func(t *testing.T) { // only ecc
		testDmgCSVHelper(t, 11, 2, "0||1; 1||0; 3||1; 13||1")
	})
}

func TestRandomDamage(t *testing.T) {
	source := rand.NewSource(time.Now().UnixNano())
	r1 := rand.New(source)
	var builder strings.Builder
	for i:=0; i<10000; i++ {
		builder.Reset()
		nd := r1.Intn(19)+1
		nr := r1.Intn(19)+1
		if (nr > nd) {
			t := nd
			nd = nr
			nr = t
		}
		ns := r1.Intn(29)+1
		section := 0
		for i:=0; i<ns; i++ {
			section = section + r1.Intn(3)+1
			nddmg := r1.Intn(nd)
			nedmg := r1.Intn(nr)
			if (nddmg == 0) && (nedmg == 0) {
				continue
			}
			fmt.Fprintf(&builder,"%d|", section)
			for j:=0; j<nddmg-1; j++ {
				fmt.Fprintf(&builder, "%d,", j)
			}
			if nddmg != 0 {
				fmt.Fprintf(&builder, "%d|", nddmg)	
			} else {
				fmt.Fprintf(&builder, "|")
			}
			
			for j:=0; j<nedmg-1; j++ {
				fmt.Fprintf(&builder, "%d,", j)
			}
			if nedmg != 0 {
				fmt.Fprintf(&builder, "%d;", nedmg)	
			} else {
				fmt.Fprintf(&builder, ";")
			}
			
		}

		s := builder.String()
		if len(s)>0 {
			// fmt.Println(s)
			testDmgCSVHelper(t, uint16(nd), uint16(nr), s[:len(s)-1])	
		}
	}
}

func testDmgCSVHelper (t *testing.T, numData, numRecovery uint16, setup string) {
	meta := types.Metadata{FileSize:0, BlockSize:4096, NumData:numData, NumRecovery:numRecovery}
	damages := makeDamageArray(setup)
	s1, s2 := cmdparser.DamageToCSV(damages, &meta)
	d1 := cmdparser.CSVToIntArr("["+*s1+"]")
	d2 := cmdparser.CSVToIntArr("["+*s2+"]")
	calcDmgs := cmdparser.CSVToDamage(&meta, d1, d2)

	if len(calcDmgs) != len(damages) {
		t.Error("Calculated damages differ")
		testDmgCSVInfo(&setup, s1, s2, damages, calcDmgs, numData, numRecovery)
		t.FailNow()
	}

	for i, d := range damages {
		if d.Section != calcDmgs[i].Section {
			t.Error("Calculated damages differ")
			testDmgCSVInfo(&setup, s1, s2, damages, calcDmgs, numData, numRecovery)
			t.FailNow()
		}

		if (!equals(d.DataDamage, calcDmgs[i].DataDamage) || !equals(d.EccDamage, calcDmgs[i].EccDamage)) {
			t.Error("Calculated damages differ")
			testDmgCSVInfo(&setup, s1, s2, damages, calcDmgs, numData, numRecovery)
			t.FailNow()
		}
	}


}

func testDmgCSVInfo(setup, s1, s2 *string, d1, d2 []decoding.DamageDesc, d, r uint16) {
	fmt.Printf("RS(%d, %d)\n", d,r)
	fmt.Println(*setup)
	fmt.Println(d1)
	fmt.Println(d2)
	fmt.Println(*s1)
	fmt.Println(*s2)
}


func makeDamageArray(setup string) []decoding.DamageDesc{ // format: section | a,.. | a,.. ; section | a,.. | a,.. 
	sections := strings.Split(setup, ";")
	arr := make([]decoding.DamageDesc, len(sections))
	for i, s := range sections {
		vars := strings.Split(strings.TrimSpace(s), "|")
		arr[i].Section, _ = strconv.Atoi(vars[0])
		arr[i].DataDamage = csvToArr(vars[1])
		arr[i].EccDamage = csvToArr(vars[2])
	}
	return arr
}


func csvToArr(line string) []int {
	if len(line) == 0 {
		return []int{}
	}
	pos := strings.Split(line, ",")
	ret := make([]int, len(pos))
	for i, s := range pos {
		val, _ := strconv.Atoi(s)
		ret[i] = val
	}
	return ret
}
