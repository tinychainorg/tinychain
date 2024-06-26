package vm

import (
	"fmt"
	"io/ioutil"
	"testing"
)

func TestBrainfuckRun(t *testing.T) {
	content, err := ioutil.ReadFile("brainfuck/solong.bf")
	
	if err != nil {
		fmt.Println("Error reading file:", err)
		return
	}

	code := string(content)
	vm := NewBrainfuckVM()
	output, gasCost, err := vm.Apply(code, 250000000000)
	if err != nil {
		fmt.Println("Error:", err)
	} else {
		fmt.Println("output:", output)
		fmt.Println("gas_cost:", gasCost)
		fmt.Println(vm.DumpMemory())
	}
}
