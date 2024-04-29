package ftail

import (
	"fmt"
	"testing"
)

func TestBasic(t *testing.T) {
	fpath := "/var/log/messages"
	ft := NewFTailer(fpath, true)

	c := make(chan string)
	go ft.Tail(c)
	for line := range c {
		fmt.Println(line)
	}
}
