FROM golang:1.26-alpine AS builder
ARG VERSION

WORKDIR /src

# Copy the whole module (including the vendored ./sdk replace target) so the
# local replace directive resolves, then build a static binary.
COPY . .
RUN go mod download

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w -X main.version=${VERSION}" -o main /src/cmd/main.go


FROM alpine:3.19.7
USER nobody
# Binary
COPY --from=builder /src/main /app/main
# Certs
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs

EXPOSE 8080

HEALTHCHECK --interval=10s --timeout=3s --start-period=10s --retries=5 \
  CMD wget -q -O /dev/null http://localhost:8080/health || exit 1

ENTRYPOINT [ "/app/main" ]
