FROM quay.io/cybozu/golang:1.18-focal as builder

WORKDIR /work

COPY . .

RUN go build -o csi-driver-availability-monitor

FROM ubuntu:20.04

RUN apt-get update \
    && apt-get install -y --no-install-recommends fio jq \
    && rm -rf /var/lib/apt/lists/*

COPY --from=builder /work/csi-driver-availability-monitor /

EXPOSE 8080/tcp
EXPOSE 8081/tcp
EXPOSE 8082/tcp

ENTRYPOINT [ "/csi-driver-availability-monitor" ]
