,.
; Read in each char ; 
; Memory layout
  0 - current character
  1 - instruction pointer
  2 - >
  3 - <
  4 - +
  5 - -
;

; 
Conditions are built using branches
current character == '+'
is translated to:

store '+' at dp=4
    increment ip to 4
    then increment dp to '+'
    ord('+') - 43

comparison is implemented by subtraction

x = '+'
y = '+'
while ord(y) != 0:
    y--

basically we have a massive 8 branches to just do these comparisons
IF
and if a comparison matches, then we fall through

[]