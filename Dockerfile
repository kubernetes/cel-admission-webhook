FROM golang:bullseye AS builder

ADD . /build
WORKDIR /build
RUN go build -trimpath -buildmode=pie -o /usr/local/bin/cel-webhook ./cmd/cel-admission-webhook

FROM gcr.io/distroless/base-debian11
COPY --from=builder /usr/local/bin/cel-webhook /usr/bin/cel-webhook
ENTRYPOINT ["/usr/bin/cel-webhook"]
