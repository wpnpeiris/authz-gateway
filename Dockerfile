FROM golang:1.26.5 AS builder

WORKDIR /src

COPY . .

RUN make build

FROM gcr.io/distroless/static-debian12

COPY --from=builder /src/bin/authz-gateway /usr/local/bin/authz-gateway

ENTRYPOINT ["/usr/local/bin/authz-gateway"]
