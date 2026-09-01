FROM golang:1.27-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/pagamento ./cmd/pagamento

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/pagamento /pagamento
USER nonroot:nonroot
EXPOSE 8080
ENTRYPOINT ["/pagamento"]
