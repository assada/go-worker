FROM golang

WORKDIR /$GOPATH/src/github.com/assada/go-worker

RUN curl -fsSL -o /usr/local/bin/dep https://github.com/golang/dep/releases/download/v0.5.4/dep-linux-amd64 && chmod +x /usr/local/bin/dep

COPY . .

RUN dep ensure -vendor-only && CGO_ENABLED=0 go build -o /myapp

FROM scratch

COPY --from=0 /myapp /myapp

ENTRYPOINT ["/myapp"]