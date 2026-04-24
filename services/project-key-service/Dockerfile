FROM golang:1.23-alpine AS build
WORKDIR /src
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/project-key-service ./services/project-key-service

FROM alpine:3.20
WORKDIR /app
COPY --from=build /out/project-key-service /app/project-key-service
EXPOSE 8082
ENTRYPOINT ["/app/project-key-service"]
