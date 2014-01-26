# Go MUX

pty multiplexer, because tmux is too overkill for what we need.
We just want a pty multiplexer, not a full blown terminal multiplexer.

## How to test this

- Get all dependencies
- Start the server with: `go install ./... && rm -rf /tmp/gomux.sock && $GOBIN/gomux`
- Start the client with: `go -c=true -i=1`
- You can start as many client as you want

This works thanks alot to docker. This is essentially docker interactive/attach
code, applied to pty only but we want to add more of a session management touch to it.
