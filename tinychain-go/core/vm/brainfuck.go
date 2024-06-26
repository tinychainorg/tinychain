package vm

import (
	"fmt"
)

var language = "><+-.,[]"

func isValidBF(code string) bool {
	for _, char := range code {
		if !contains(language, char) {
			return false
		}
	}
	return true
}

func contains(s string, c rune) bool {
	for _, char := range s {
		if char == c {
			return true
		}
	}
	return false
}

func createJumpTable(chars string) map[int]int {
	jumpTable := make(map[int]int)
	leftPositions := []int{}

	for pos, char := range chars {
		switch char {
		case '[':
			leftPositions = append(leftPositions, pos)
		case ']':
			left := leftPositions[len(leftPositions)-1]
			leftPositions = leftPositions[:len(leftPositions)-1]
			jumpTable[left] = pos
			jumpTable[pos] = left
		}
	}
	return jumpTable
}

func dumpMem(mem map[int]int) {
	for key, value := range mem {
		fmt.Printf("%04d: %c\n", key, value)
	}
}

func brainfuckFmt(code string) string {
	output := ""
	depth := 0
	indent := "  "
	for _, char := range code {
		switch char {
		case '[':
			output += "\n" + string(repeat(' ', depth*len(indent))) + "[\n"
			depth++
			output += string(repeat(' ', depth*len(indent)))
		case ']':
			output += "\n"
			depth--
			output += string(repeat(' ', depth*len(indent))) + "]\n"
			output += string(repeat(' ', depth*len(indent)))
		default:
			output += string(char)
		}
	}
	return output
}

func repeat(char rune, count int) []rune {
	repeated := make([]rune, count)
	for i := 0; i < count; i++ {
		repeated[i] = char
	}
	return repeated
}

type OutOfGasException struct {
	Message string
}

func (e *OutOfGasException) Error() string {
	return e.Message
}

func brainfuckRun(code string, input string, mem map[int]int, gasLimit int64, debugger bool) (string, string, map[int]int, int64, error) {
	dp := 0
	output := ""
	inputIdx := 0
	ip := 0
	jmpRegister := -1
	jmpTable := createJumpTable(code)

	GAS_USE_TABLE := map[string]int64{
		"memory":  3,
		"compute": 1,
	}

	gasUsed := int64(0)

	for ip < len(code) {
		if gasLimit < gasUsed {
			return "", "", mem, gasUsed, &OutOfGasException{Message: fmt.Sprintf("gas_limit = %d, gas_used = %d", gasLimit, gasUsed)}
		}

		opcode := rune(code[ip])
		gasUsed += GAS_USE_TABLE["compute"]

		if debugger {
			fmt.Printf("op=%c dp=%05d d=%02d >\n", opcode, dp, mem[dp])
			fmt.Scanln()
		}

		switch opcode {
		case '>':
			dp++
		case '<':
			dp--
		case '+':
			gasUsed += GAS_USE_TABLE["memory"]
			mem[dp]++
		case '-':
			gasUsed += GAS_USE_TABLE["memory"]
			mem[dp]--
		case '.':
			output += string(rune(mem[dp] % 256))
		case ',':
			if inputIdx < len(input) {
				mem[dp] = int(input[inputIdx])
				inputIdx++
			} else {
				mem[dp] = 0
			}
		case '[':
			if mem[dp] == 0 {
				jmpRegister = jmpTable[ip]
			}
		case ']':
			if mem[dp] != 0 {
				jmpRegister = jmpTable[ip]
			}
		case ';':
			for code[ip] != ';' {
				ip++
			}
		}

		if jmpRegister == -1 {
			ip++
		} else {
			ip = jmpRegister
			jmpRegister = -1
		}
	}

	return output, "", mem, gasUsed, nil
}

type BrainfuckVM struct {
	memory map[int]int
}

func NewBrainfuckVM() *BrainfuckVM {
	return &BrainfuckVM{memory: make(map[int]int)}
}

func (vm *BrainfuckVM) Eval(code string, gasLimit int64) (string, int64, error) {
	memCopy := copyMemory(vm.memory)
	output, _, _, gasCost, err := brainfuckRun(code, "", memCopy, gasLimit, false)
	return output, gasCost, err
}

func (vm *BrainfuckVM) Apply(code string, gasLimit int64) (string, int64, error) {
	memCopy := copyMemory(vm.memory)
	output, _, mem, gasCost, err := brainfuckRun(code, "", memCopy, gasLimit, false)
	if err == nil {
		vm.memory = mem
	}
	return output, gasCost, err
}

func (vm *BrainfuckVM) DumpMemory() string {
	buf := ""
	for key, value := range vm.memory {
		buf += fmt.Sprintf("%04d: %c\n", key, value)
	}
	return buf
}

func copyMemory(mem map[int]int) map[int]int {
	memCopy := make(map[int]int)
	for k, v := range mem {
		memCopy[k] = v
	}
	return memCopy
}
