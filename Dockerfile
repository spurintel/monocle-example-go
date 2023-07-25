# Build stage
FROM golang:alpine AS builder
WORKDIR /app
COPY . .
RUN go build -a -o server main.go

# Final stage
FROM scratch
COPY --from=builder /app/server /server
EXPOSE 8080
CMD ["/server"]