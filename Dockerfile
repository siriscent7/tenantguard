# ---- build stage ----
FROM golang:1.26 AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o tenantguard .

# ---- run stage ----
FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=build /app/tenantguard .
EXPOSE 8080
CMD ["./tenantguard"]