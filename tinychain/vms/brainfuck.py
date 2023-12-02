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
    for key, value in mem.items():
        print("{:04d}: {}".format(key, value))
    
    # buf = ""
    # MEM_STEP = 8
    # for i in range((len(mem) // MEM_STEP) + 1):
    #     buf += "{:04d} ".format(i)
    #     for j in range(MEM_STEP):
    #         buf += chr(mem[i+j])
    # return buf

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

# The out of gas exception.
class OutOfGasException(Exception):
    pass

# Runs a Brainfuck interpreter on code.
# `memory` is all zeros by default.
def brainfuck_run(code, input_="", mem=defaultdict(int), gas_limit=250000000000, debugger=False):
    # data pointer
    dp = 0

    # output buffer
    output = ""
    # input buffer
    input_i = 0

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

    # gas metering.
    GAS_USE_TABLE = {
        'memory': 3,
        'compute': 1
    }
    gas_used = 0

    while True:
        if len(code) - 1 < ip:
            # end of program.
            break
        
        if gas_limit < gas_used:
            raise OutOfGasException("gas_limit = {}, gas_used = {}".format(gas_limit, gas_used))

        opcode = code[ip]
        
        gas_used += GAS_USE_TABLE["compute"]
        
        # "00100 +"
        debug("{:05d} {}\n".format(ip, opcode))

        if debugger:
            input("op={} dp={:05d} d={:02d} >".format(opcode, dp, mem[dp]))

        if opcode == '>':
            # Increment the data pointer by one
            dp += 1
        elif opcode == '<':
            # Decrement the data pointer by one
            dp -= 1
        elif opcode == '+':
            # Increment the byte at the data pointer by one. 
            gas_used += GAS_USE_TABLE["memory"]
            mem[dp] += 1
        elif opcode == '-':
            # Decrement the byte at the data pointer by one. 
            gas_used += GAS_USE_TABLE["memory"]
            mem[dp] -= 1
        elif opcode == '.':
            # Output the byte at the data pointer. 
            # print(mem[dp])
            output += chr(mem[dp] % 256)
        elif opcode == ',':
            # Accept one byte of input, storing its value in the byte at the data pointer. 
            if input_i < len(input_) - 1:
                mem[dp] = ord(input_[input_i])
                input_i += 1
            else:
                # TODO. exception?
                mem[dp] = 0
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
        elif opcode == ';':
            # Comment.
            # Skip all characters until the next semicolon.
            while code[ip] != ';':
                ip += 1

        # Advance instruction pointer if not a jump instruction.
        if jmp_register == -1:
            ip += 1
        else:
            debug("{:05d} JUMP {}\n".format(ip, jmp_register))
            ip = jmp_register
            jmp_register = -1
        
        continue
    
    return (output, debug_stream.log, mem, gas_used)

class BrainfuckVM:
    def __init__(self):
        self.memory = defaultdict(int)
    
    def eval(self, code="", gas_limit=0):
        memory1 = self.memory.copy()
        (output, debug, mem, gas_cost) = brainfuck_run(code, memory1, gas_limit)

    def apply(self, code="", gas_limit=0):
        memory1 = self.memory.copy()
        (output, debug, mem, gas_cost) = brainfuck_run(code, "", mem=memory1, gas_limit=gas_limit)
        self.memory = memory1
        return (output, gas_cost)
    
    def dump_memory(self):
        buf = ""
        for key, value in self.memory.items():
            buf += "{:04d}: {}\n".format(key, chr(value))
        return buf


if __name__ == "__main__":
    # print "Hello world" to screen.
    # (output, debug, mem, gas_cost) = brainfuck_run("++++++++[>++++[>++>+++>+++>+<<<<-]>+>+>->>+[<]<-]>>.>---.+++++++..+++.>>.<-.<.+++.------.--------.>>+.>++.")
    # (output, debug, mem, gas_cost) = brainfuck_run(">>>>>++")
    
    
    
    # (output, debug, mem, gas_cost) = brainfuck_run(
    #     open('brainfuck/bf-interpreter.bf').read(),
    #     # input=open('brainfuck/helloworld.bf').read()
    #     input_="++"
    # )


    (output, debug, mem, gas_cost) = brainfuck_run(
        open('brainfuck/solong.bf').read(),
        # "+++++>++++++temp0[-]x[temp0+x-]y[x+y-]temp0[y+temp0-]",
        # input=open('brainfuck/helloworld.bf').read()
        # input_="++"
    )


    # (output, debug, mem) = brainfuck_run("[->+<]")
    # print(brainfuck_fmt("+[-->-[>>+>-----<<]<--<---]>-.>>>+.>>..+++[.>]<<<<.+++.------.<<-.>>>>+."))
    # (output, debug, mem, gas_cost) = brainfuck_run("+[-->-[>>+>-----<<]<--<---]>-.>>>+.>>..+++[.>]<<<<.+++.------.<<-.>>>>+.")

    print("output: {}".format(output))
    print("gas_cost: {}".format(gas_cost))
    print(dump_mem(mem))
    # print(mem)


    