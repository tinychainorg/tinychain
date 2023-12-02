from lisp.vm import make_lisp_vm
from lisp.reader import read_str

def run(code):
    [REP, EP, EVAL, env] = make_lisp_vm()

    # Read some builtins.
    REP(open('engine.lisp').read())
    for ast in read_str("(" + open('engine.lisp').read() + ")"):
        EVAL(ast, env)
        # REP(ast)
    

    # print(env.data)

    # Gas cost calculation.
    # For now, we just count parens.
    gas_cost = code.count('(') + code.count(')')
    
    exit_code = 1
    try:
        retval = REP(code)
        print("retval={} exit_code={} gas_cost={}".format(retval, exit_code, gas_cost))
        exit_code = 0
    except Exception as e:
        print(e)
        exit_code = 1
        


run("(println sstore)")
run("(println sload)")
run("(foreach (lambda (x) (print x)) '(1 2 3 4 5))")