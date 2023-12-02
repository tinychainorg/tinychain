from collections import defaultdict
import sys

def is_valid_bf(code):
    language = '><+-.,[]'

    # Validate program.
    # Verify code only contains alphabet from the language.
    for char in code:
        if char not in language:
            return False
    return True

def create_jump_table(chars):
    jump_table = {}
    left_positions = []

    position = 0
    for char in chars:
        if char == '[':
            left_positions.append(position)

        elif char == ']':
            left = left_positions.pop()
            right = position
            jump_table[left] = right
            jump_table[right] = left
        position += 1

    return jump_table



def dump_mem(mem):
    buf = ""
    MEM_STEP = 8
    for i in range((len(mem) // MEM_STEP) + 1):
        buf += "{:04d} ".format(i)
        for j in range(MEM_STEP):
            buf += chr(mem[i+j])
    return buf

def brainfuck_fmt(code):
    output_ = ""
    depth = 0
    indent = "  "
    for char in code:
        if char == '[':
            output_ += "\n"
            output_ += (depth * indent) + "["
            depth += 1
            output_ += "\n"
            output_ += depth * indent
        elif char == ']':
            output_ += "\n"
            depth -= 1
            output_ += (depth * indent) + "]"
            output_ += "\n"
            output_ += depth * indent
        else:
            output_ += char
    return output_

def brainfuck_run(code):
    # data pointer
    dp = 0
    # memory cells
    mem = defaultdict(int) # default cell value of 0

    # output buffer
    output = ""
    # input buffer
    input_ = ""

    # debug log
    class DebugStream:
        def __init__(self):
            self.log = ""
        
        def write(self, x):
            self.log += x
    
    debug_stream = DebugStream()
    def debug(x):
        debug_stream.write(x)
        if False:
            sys.stdout.write(x)

    # instruction pointer
    ip = 0
    # jump register
    jmp_register = -1
    # jump table
    jmp_table = create_jump_table(code)

    # is_debugging = True
    is_debugging = False

    while True:
        if len(code) - 1 < ip:
            # end of program.
            break
        opcode = code[ip]
        
        # 00100 + 
        debug("{:05d} {}\n".format(ip, opcode))

        if is_debugging:
            input("op={} dp={:05d} d={:02d} >".format(opcode, dp, mem[dp]))

        if opcode == '>':
            # Increment the data pointer by one
            dp += 1
        elif opcode == '<':
            # Decrement the data pointer by one
            dp -= 1
        elif opcode == '+':
            # Increment the byte at the data pointer by one. 
            mem[dp] += 1
        elif opcode == '-':
            # Decrement the byte at the data pointer by one. 
            mem[dp] -= 1
        elif opcode == '.':
            # Output the byte at the data pointer. 
            output += chr(mem[dp])
        elif opcode == ',':
            # Accept one byte of input, storing its value in the byte at the data pointer. 
            # TODO.
            pass
        elif opcode == '[':
            # Jump If Zero
            # Jumps to the matching ] instruction if the value of the current cell is zero
            if mem[dp] == 0:
                jmp_register = jmp_table[ip]
                
        elif opcode == ']':
            # Jump If Not Zero
            # Jumps to the matching [ instruction if the value of the current cell is nonzero
            if mem[dp] != 0:
                jmp_register = jmp_table[ip]

        # Advance instruction pointer if not a jump instruction.
        if jmp_register == -1:
            ip += 1
        else:
            debug("{:05d} JUMP {}\n".format(ip, jmp_register))
            ip = jmp_register
            jmp_register = -1
        
        continue
    
    return (output, debug_stream.log, mem)
    

if __name__ == "__main__":
    # print "Hello world" to screen.
    # (output, debug, mem) = brainfuck_run("++++++++[>++++[>++>+++>+++>+<<<<-]>+>+>->>+[<]<-]>>.>---.+++++++..+++.>>.<-.<.+++.------.--------.>>+.>++.")
    # (output, debug, mem) = brainfuck_run("[->+<]")
    print(brainfuck_fmt("+[-->-[>>+>-----<<]<--<---]>-.>>>+.>>..+++[.>]<<<<.+++.------.<<-.>>>>+."))
    (output, debug, mem) = brainfuck_run("+[-->-[>>+>-----<<]<--<---]>-.>>>+.>>..+++[.>]<<<<.+++.------.<<-.>>>>+.")

    print(output)
    # print(dump_mem(mem))
    # print(mem)


    