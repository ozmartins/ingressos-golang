FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /bin/estoque ./cmd/estoque

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /bin/estoque /estoque
USER nonroot:nonroot
EXPOSE 50051 8090
ENTRYPOINT ["/estoque"]
