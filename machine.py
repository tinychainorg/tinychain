from lisp.vm import make_lisp_vm

def run(code):
    [REP, EP, env] = make_lisp_vm()

    # Read some builtins.
    REP(open('engine.lisp').read())

    # Gas cost calculation.
    # For now, we just count parens.
    gas_cost = code.count('(') + code.count(')')
    
    exit_code = 1
    try:
        retval = EP(code)
        print("retval={} exit_code={} gas_cost={}".format(retval, exit_code, gas_cost))
        exit_code = 0
    except Exception as e:
        print(e)
        exit_code = 1
        


run("(println sstore)")