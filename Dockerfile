FROM golang:1.13

WORKDIR /github.com/iCell/potail

COPY . .

RUN go mod download

RUN go build -o potail *.go

CMD ["./potail"]
