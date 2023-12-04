s = """
++
[>+<-]
>[-[>+x++<-]x+>[<+>-]<]
"""

print(
    s
    .replace("temp0", ">>>")
    .replace("temp1", ">>")
    .replace("x", ">")
)
