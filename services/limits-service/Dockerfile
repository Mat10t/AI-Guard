FROM golang:1.23-alpine AS build
WORKDIR /src
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/limits-service ./services/limits-service

FROM alpine:3.20
WORKDIR /app
COPY --from=build /out/limits-service /app/limits-service
EXPOSE 8083
ENTRYPOINT ["/app/limits-service"]
