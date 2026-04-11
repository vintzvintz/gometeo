FROM golang:1.26.2-bookworm AS builder
ARG COMMIT_ID=unknown
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags "-X gometeo/appconf.CommitID=${COMMIT_ID}" -o gometeo

FROM alpine:3.21
RUN apk add --no-cache ca-certificates tzdata
COPY --from=builder /build/gometeo /gometeo
EXPOSE 1051
HEALTHCHECK --interval=30s --timeout=5s --start-period=60s --retries=3 \
  CMD wget -qO- http://localhost:1051/healthz || exit 1
CMD ["/gometeo"]