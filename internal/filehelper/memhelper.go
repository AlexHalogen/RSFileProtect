package filehelper

func Memset(a []byte, c byte, n int, offset int) {
	count := n
	if len(a) - offset < n {
		count = len(a) - offset
	}
	for i:=offset; i<offset+count; i++ {
		a[i] = c
	}
}