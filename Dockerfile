# Start from a Debian image with the latest version of Go installed
# and a workspace (GOPATH) configured at /go.
FROM golang

# Copy the local package files to the container's workspace.
ADD . /go/src/PNWPowder

COPY . /go
RUN go install PNWPowder
RUN find . -not -path '*/\.*'

# Run the outyet command by default when the container starts.
ENTRYPOINT /go/bin/PNWPowder

# Document that the service listens on port 80.
EXPOSE 80