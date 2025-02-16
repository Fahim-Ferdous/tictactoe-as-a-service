# TaaS, TicTacToe as a Service

### Memory Representation

There are 9 slots,

    _ _ _
    _ _ _
    _ _ _

A slot can have any of `0`, `1`, `2`. So a board's state fits in a 32 bit int.

    2 1 2
    _ 1 1
    2 1 _

Taking the top left slot as the most significant digit, the board above can be
serialized as follows (replaced `_` with `0`),

    2 1 2 _ 1 1 2 1 _
    2 1 2 0 1 1 2 1 0

With this, we have a 10^9 int. If we use base 3, we need an int no larger than
3^9 (15 bits). If we just use base 2, we will need 18 bits. In base 3, we can
have a flag that will indicate whether this is a terminal state or not. But
that is for another day.

According to google, there are 5,478 valid states. By "valid", I mean states
that you can encounter when playing. I.e. no extra pieces. So, we need only 13
bits to index a board state.
