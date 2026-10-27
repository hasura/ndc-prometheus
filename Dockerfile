# build context at repo root: docker build -f Dockerfile .
FROM golang:1.26 AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -v -o ndc-prometheus ./server

# CA bundle without the eMudhra (emSign) roots, which are being distrusted.
# Debian's ca-certificates still ships some of them, so deselect them here.
FROM debian:trixie-slim AS certs
RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates \
    && sed -i -E 's|^mozilla/emSign|!&|' /etc/ca-certificates.conf \
    && update-ca-certificates --fresh \
    && ! ls /etc/ssl/certs | grep -i emsign

# stage 2: production image
FROM gcr.io/distroless/static-debian13:nonroot

# Copy the binary to the production image from the builder stage.
COPY --from=builder /app/ndc-prometheus /ndc-prometheus
COPY --from=certs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

ENV HASURA_CONFIGURATION_DIRECTORY=/etc/connector

ENTRYPOINT ["/ndc-prometheus"]

# Run the web service on container startup.
CMD ["serve"]