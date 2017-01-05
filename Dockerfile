FROM golang

# Copy the local package files to the container's workspace.
ADD app /go/src/route35
ADD test/config.json /
ADD test/records /
ADD test/blocks /

RUN go get route35

# Build the outyet command inside the container.
# (You may fetch or manage dependencies here,
# either manually or with a tool like "godep".)
RUN go install route35

# Run the outyet command by default when the container starts.
ENTRYPOINT /go/bin/route35 /config.json

# Document that the service listens on port 8080.
EXPOSE 8081
EXPOSE 5300:53/udp
EXPOSE 5300:53/tcp
