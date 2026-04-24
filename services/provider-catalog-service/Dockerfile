FROM golang:1.23-alpine AS build
WORKDIR /src
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/provider-catalog-service ./services/provider-catalog-service

FROM alpine:3.20
WORKDIR /app
COPY --from=build /out/provider-catalog-service /app/provider-catalog-service
EXPOSE 8084
ENTRYPOINT ["/app/provider-catalog-service"]
