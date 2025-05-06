FROM golang:1.24 AS build

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o url-shortener

FROM gcr.io/distroless/base-debian12

WORKDIR /app
COPY --from=build /app/url-shortener .
COPY static/ ./static/
COPY data/ ./data/

EXPOSE 8081
CMD ["./url-shortener"]
