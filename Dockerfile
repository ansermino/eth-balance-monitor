FROM  golang:1.15-stretch AS builder
ADD . /src
WORKDIR /src
RUN go mod download
RUN go build -o /eth-balance-monitor .

FROM debian:stretch-slim
RUN apt-get -y update && apt-get -y upgrade && apt-get install ca-certificates wget -y
COPY --from=builder /eth-balance-monitor ./
RUN chmod +x ./eth-balance-monitor

ENTRYPOINT ["./eth-balance-monitor"]