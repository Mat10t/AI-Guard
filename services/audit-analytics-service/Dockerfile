FROM golang:1.23-alpine AS build
WORKDIR /src
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/audit-analytics-service ./services/audit-analytics-service

FROM alpine:3.20
WORKDIR /app
COPY --from=build /out/audit-analytics-service /app/audit-analytics-service
EXPOSE 8085
ENTRYPOINT ["/app/audit-analytics-service"]
