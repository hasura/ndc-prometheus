# build context at repo root: docker build -f Dockerfile .
FROM golang:1.23 AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -v -o ndc-prometheus ./server

# stage 2: production image
FROM gcr.io/distroless/static-debian12:nonroot

# Copy the binary to the production image from the builder stage.
COPY --from=builder /app/ndc-prometheus /ndc-prometheus

ENV HASURA_CONFIGURATION_DIRECTORY=/etc/connector
ENV OTEL_SERVICE_NAME=ndc_prometheus

ENTRYPOINT ["/ndc-prometheus"]

# Run the web service on container startup.
CMD ["serve"]