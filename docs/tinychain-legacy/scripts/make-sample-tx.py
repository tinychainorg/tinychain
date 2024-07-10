
msg = 'so long and thanks for all the fish'
def generate_bf_program(input_string):
    bf_program = ''
    bf_program += '[-]'  # Clear the current cell

    for char in input_string:
        bf_program += '+' * ord(char) + '.>'  # Increment the value, print character, move to the next cell

    bf_program += '[-]'  # Clear the last cell

    return bf_program
print(generate_bf_program(msg))
