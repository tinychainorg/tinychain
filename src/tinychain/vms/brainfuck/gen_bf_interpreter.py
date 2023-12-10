import sys

symbols = '+-[]><.,'
ords = sorted([ord(c) for c in symbols])


def test():
    for i in ords:
        sys.stdout.write("{} {}\n".format(chr(i), i))

i = 0

def write_cell(val):
    global i

    loc = i
    # write the value to the cell.
    print("+" * val)
    # increment the data pointer.
    print(">")
    i += 1
    # return the address of our written cell.
    return loc

# Write the interpreter.
def gen_branch():
    x = ords[-1]

    # set cell 0 to ord's value.
    print("+" * x)

    # now write the equality check using a branch.
    # cell 0 represents this ord.
    template = """
temp0[-]
temp1[-]
x[temp0+temp1+x-]temp0[x+temp0-]+
temp1[
 code1
 temp0-
temp1[-]]
temp0[
 code2
temp0-]
"""

    print("[")


def test_swap():
    y = write_cell(50)
    x = write_cell(3)
#     print("""temp0[-]
# x[temp0+x-]
# y[x+y-]
# temp0[y+temp0-]""")

test_swap()
# gen_branch()