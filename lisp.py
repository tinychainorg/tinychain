
# Lisp S-Expression parser.
# Copied from:
# https://github.com/kanaka/mal/blob/master/impls/python/reader.py
# https://github.com/kanaka/mal/blob/master/impls/python/mal_types.py

import re
import sys, copy, types as pytypes

# python 3.0 differences
if sys.hexversion > 0x3000000:
    _u = lambda x: x
    _s2u = lambda x: x
else:
    import codecs
    _u = lambda x: codecs.unicode_escape_decode(x)[0]
    _s2u = lambda x: unicode(x)

if sys.version_info[0] >= 3:
    str_types = [str]
else:
    str_types = [str, unicode]

# General functions

def _equal_Q(a, b):
    ota, otb = type(a), type(b)
    if _string_Q(a) and _string_Q(b):
        return a == b
    if not (ota == otb or (_sequential_Q(a) and _sequential_Q(b))):
        return False;
    if _symbol_Q(a):
        return a == b
    elif _list_Q(a) or _vector_Q(a):
        if len(a) != len(b): return False
        for i in range(len(a)):
            if not _equal_Q(a[i], b[i]): return False
        return True
    elif _hash_map_Q(a):
        akeys = sorted(a.keys())
        bkeys = sorted(b.keys())
        if len(akeys) != len(bkeys): return False
        for i in range(len(akeys)):
            if akeys[i] != bkeys[i]: return False
            if not _equal_Q(a[akeys[i]], b[bkeys[i]]): return False
        return True
    else:
        return a == b

def _sequential_Q(seq): return _list_Q(seq) or _vector_Q(seq)

def _clone(obj):
    #if type(obj) == type(lambda x:x):
    if type(obj) == pytypes.FunctionType:
        if obj.__code__:
            return pytypes.FunctionType(
                    obj.__code__, obj.__globals__, name = obj.__name__,
                    argdefs = obj.__defaults__, closure = obj.__closure__)
        else:
            return pytypes.FunctionType(
                    obj.func_code, obj.func_globals, name = obj.func_name,
                    argdefs = obj.func_defaults, closure = obj.func_closure)
    else:
        return copy.copy(obj)

#
# Exception type
#

class MalException(Exception):
    def __init__(self, object):
        self.object = object

# Scalars
def _nil_Q(exp):    return exp is None
def _true_Q(exp):   return exp is True
def _false_Q(exp):  return exp is False
def _string_Q(exp):
    if type(exp) in str_types:
        return len(exp) == 0 or exp[0] != _u("\u029e")
    else:
        return False
def _number_Q(exp): return type(exp) == int

# Symbols
class Symbol(str): pass
def _symbol(str): return Symbol(str)
def _symbol_Q(exp): return type(exp) == Symbol

# Keywords
# A specially prefixed string
def _keyword(str):
    if str[0] == _u("\u029e"): return str
    else:                      return _u("\u029e") + str
def _keyword_Q(exp):
    if type(exp) in str_types:
        return len(exp) != 0 and exp[0] == _u("\u029e")
    else:
        return False

# Functions
def _function(Eval, Env, ast, env, params):
    def fn(*args):
        return Eval(ast, Env(env, params, List(args)))
    fn.__meta__ = None
    fn.__ast__ = ast
    fn.__gen_env__ = lambda args: Env(env, params, args)
    return fn
def _function_Q(f):
    return callable(f)

# lists
class List(list):
    def __add__(self, rhs): return List(list.__add__(self, rhs))
    def __getitem__(self, i):
        if type(i) == slice: return List(list.__getitem__(self, i))
        elif i >= len(self): return None
        else:                return list.__getitem__(self, i)
    def __getslice__(self, *a): return List(list.__getslice__(self, *a))
def _list(*vals): return List(vals)
def _list_Q(exp):   return type(exp) == List


# vectors
class Vector(list):
    def __add__(self, rhs): return Vector(list.__add__(self, rhs))
    def __getitem__(self, i):
        if type(i) == slice: return Vector(list.__getitem__(self, i))
        elif i >= len(self): return None
        else:                return list.__getitem__(self, i)
    def __getslice__(self, *a): return Vector(list.__getslice__(self, *a))
def _vector(*vals): return Vector(vals)
def _vector_Q(exp): return type(exp) == Vector

# Hash maps
class Hash_Map(dict): pass
def _hash_map(*key_vals):
    hm = Hash_Map()
    for i in range(0,len(key_vals),2): hm[key_vals[i]] = key_vals[i+1]
    return hm
def _hash_map_Q(exp): return type(exp) == Hash_Map

# atoms
class Atom(object):
    def __init__(self, val):
        self.val = val
def _atom(val): return Atom(val)
def _atom_Q(exp):   return type(exp) == Atom

def py_to_mal(obj):
        if type(obj) == list:   return List(obj)
        if type(obj) == tuple:  return List(obj)
        elif type(obj) == dict: return Hash_Map(obj)
        else:                   return obj

class Blank(Exception): pass

class Reader():
    def __init__(self, tokens, position=0):
        self.tokens = tokens
        self.position = position

    def next(self):
        self.position += 1
        return self.tokens[self.position-1]

    def peek(self):
        if len(self.tokens) > self.position:
            return self.tokens[self.position]
        else:
            return None

def tokenize(str):
    tre = re.compile(r"""[\s,]*(~@|[\[\]{}()'`~^@]|"(?:[\\].|[^\\"])*"?|;.*|[^\s\[\]{}()'"`@,;]+)""");
    return [t for t in re.findall(tre, str) if t[0] != ';']

def _unescape(s):
    return s.replace('\\\\', _u('\u029e')).replace('\\"', '"').replace('\\n', '\n').replace(_u('\u029e'), '\\')

def read_atom(reader):
    int_re = re.compile(r"-?[0-9]+$")
    float_re = re.compile(r"-?[0-9][0-9.]*$")
    string_re = re.compile(r'"(?:[\\].|[^\\"])*"')
    token = reader.next()
    if re.match(int_re, token):     return int(token)
    elif re.match(float_re, token): return int(token)
    elif re.match(string_re, token):return _s2u(_unescape(token[1:-1]))
    elif token[0] == '"':           raise Exception("expected '\"', got EOF")
    elif token[0] == ':':           return _keyword(token[1:])
    elif token == "nil":            return None
    elif token == "true":           return True
    elif token == "false":          return False
    else:                           return _symbol(token)

def read_sequence(reader, typ=list, start='(', end=')'):
    ast = typ()
    token = reader.next()
    if token != start: raise Exception("expected '" + start + "'")

    token = reader.peek()
    while token != end:
        if not token: raise Exception("expected '" + end + "', got EOF")
        ast.append(read_form(reader))
        token = reader.peek()
    reader.next()
    return ast

def read_hash_map(reader):
    lst = read_sequence(reader, list, '{', '}')
    return _hash_map(*lst)

def read_list(reader):
    return read_sequence(reader, _list, '(', ')')

def read_vector(reader):
    return read_sequence(reader, _vector, '[', ']')

def read_form(reader):
    token = reader.peek()
    # reader macros/transforms
    if token[0] == ';':
        reader.next()
        return None
    elif token == '\'':
        reader.next()
        return _list(_symbol('quote'), read_form(reader))
    elif token == '`':
        reader.next()
        return _list(_symbol('quasiquote'), read_form(reader))
    elif token == '~':
        reader.next()
        return _list(_symbol('unquote'), read_form(reader))
    elif token == '~@':
        reader.next()
        return _list(_symbol('splice-unquote'), read_form(reader))
    elif token == '^':
        reader.next()
        meta = read_form(reader)
        return _list(_symbol('with-meta'), read_form(reader), meta)
    elif token == '@':
        reader.next()
        return _list(_symbol('deref'), read_form(reader))

    # list
    elif token == ')': raise Exception("unexpected ')'")
    elif token == '(': return read_list(reader)

    # vector
    elif token == ']': raise Exception("unexpected ']'");
    elif token == '[': return read_vector(reader);

    # hash-map
    elif token == '}': raise Exception("unexpected '}'");
    elif token == '{': return read_hash_map(reader);

    # atom
    else:              return read_atom(reader);

def read_str(str):
    tokens = tokenize(str)
    if len(tokens) == 0: raise Blank("Blank Line")
    return read_form(Reader(tokens))

# class LispVM:
#     def __init__(self):
#         pass

#     def eval(self, code):
#         parsed = SExpr.parseString(code, parseAll=True)
#         print(parsed)
#         # print(parsed[0])
#         # print(parse_expr(code))



# vm = LispVM()
# # vm.eval("(+ 2 (* 3 4))")
# vm.eval("(+ 2 2)")

expr = "(add 2 (* 3 4))"
print(read_str(expr))