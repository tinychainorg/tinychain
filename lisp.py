
def parse_expr(expression):
    tokens = expression.replace('(', ' ( ').replace(')', ' ) ').split()
    stack = []
    for token in tokens:
        if token == '(':
            stack.append([])
        elif token == ')':
            if len(stack) > 1:
                sublist = stack.pop()
                stack[-1].append(sublist)
            else:
                return "Invalid Expression: Mismatched parentheses"
        else:
            try:
                stack[-1].append(int(token))
            except ValueError:
                stack[-1].append(token)
    
    if len(stack) != 1:
        return "Invalid Expression: Mismatched parentheses"
    
    return stack[0]



class LispVM:
    def __init__(self):
        pass

    def eval(self, code):
        print(parse_expr(code))



vm = LispVM()
vm.eval("(+ 2 (* 3 4))")