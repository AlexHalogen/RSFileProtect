package cmdparser

import (
	"strconv"
	"strings"
	"alexhalogen/rsfileprotect/internal/decoding"
	"alexhalogen/rsfileprotect/internal/types"
	"log"
	"fmt"
)

func CSVToDamage(meta *types.Metadata, dataDmg, eccDmg []int) []decoding.DamageDesc {
	nd := int(meta.NumData)
	nr := int(meta.NumRecovery)
	dmgs := make([]decoding.DamageDesc, 0, 16)

	id := 0
	ie := 0 // current
	cd := decoding.DamageDesc{Section:-1} // sentinel node
	for id < len(dataDmg) && ie < len(eccDmg) {
		c1 := dataDmg[id] / nd
		r1 := dataDmg[id] % nd
		c2 := eccDmg[ie] / nr
		r2 := eccDmg[ie] % nr

		if c1 <= c2 { // consume dataDmg
			if c1 == cd.Section { // still working in the same section
				cd.DataDamage = append(cd.DataDamage, r1)
			} else {
				dmgs = append(dmgs, cd)
				cd = decoding.DamageDesc{Section: c1}
				cd.DataDamage = append(cd.DataDamage, r1)
			}
			id++

		} else { // consume eccDmg
			if c2 == cd.Section {
				cd.EccDamage = append(cd.EccDamage, r2)
			} else {
				dmgs = append(dmgs, cd)
				cd = decoding.DamageDesc{Section: c2}
				cd.EccDamage = append(cd.EccDamage, r2)
			}
			ie++
		}
	}

	for id < len(dataDmg) {
		c := dataDmg[id] / nd
		r := dataDmg[id] % nd
		if c == cd.Section { // still working in the same section
			cd.DataDamage = append(cd.DataDamage, r)
		} else {
			dmgs = append(dmgs, cd)
			cd = decoding.DamageDesc{Section: c}
			cd.DataDamage = append(cd.DataDamage, r)
		}
		id++
	}

	for ie < len(eccDmg) {
		c := eccDmg[ie] / nr
		r := eccDmg[ie] % nr
		if c == cd.Section { // still working in the same section
			cd.EccDamage = append(cd.EccDamage, r)
		} else {
			dmgs = append(dmgs, cd)
			cd = decoding.DamageDesc{Section: c}
			cd.EccDamage = append(cd.EccDamage, r)
		}
		ie++
	}

	dmgs = append(dmgs, cd)

	return dmgs[1:] // exclude sentinel node
}


func DamageToCSV(dmgs []decoding.DamageDesc, meta *types.Metadata) (*string, *string){
	var bd, be strings.Builder

	nd := int(meta.NumData)
	nr := int(meta.NumRecovery)

	for _, d := range dmgs {
		base := d.Section
		if len(d.DataDamage) != 0 {
			for _,v := range d.DataDamage {
				fmt.Fprintf(&bd, "%d,", base * nd + v)
			}
		}

		if len(d.EccDamage) != 0 {
			for _,v := range d.EccDamage {
				fmt.Fprintf(&be, "%d,", base * nr + v)
			}
		}
	}

	var s1, s2 *string
	if bd.Len() != 0 {
		s := bd.String()[:bd.Len()-1]
		s1 = &s // safe to do this in go
	} else {
		s := ""
		s1 = &s
	}
	if be.Len() != 0 {
		s := be.String()[:be.Len()-1]
		s2 = &s // safe to do this in go
	} else {
		s := ""
		s2 = &s
	}

	return s1, s2
}

func CSVToIntArr(line string) []int {

	if len(line) < 2 {
		return nil
	}

	if line[0] != '[' || line[len(line)-1] != ']' {
		return nil
	}

	if len(line) == 2 {
		return []int{}
	}

	line = line[1:len(line)-1]
	pos := strings.Split(line, ",")
	ret := make([]int, len(pos))

	for i, s := range pos {
		val, err := strconv.Atoi(strings.TrimSpace(s))
		if val < 0 {
			log.Printf("Negative chunk number:%d\n", val)
			return nil
		}
		if err != nil {
			log.Println(err)
			return nil
		}
		ret[i] = val
	}
	return ret
}
