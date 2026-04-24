FROM golang:1.23-alpine AS build
WORKDIR /src
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/auth-org-service ./services/auth-org-service

FROM alpine:3.20
WORKDIR /app
COPY --from=build /out/auth-org-service /app/auth-org-service
EXPOSE 8081
ENTRYPOINT ["/app/auth-org-service"]
