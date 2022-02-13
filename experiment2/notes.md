process starts
-> opens listener

receive SIGUSR1

new process
-> inherits open listener
-> opens tcp listener from file

receive SIGUSR1

new process
-> inherits open listener
-> opens tcp listener from file
