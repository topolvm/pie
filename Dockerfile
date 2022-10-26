FROM golang:1.19 as builder

WORKDIR /work

COPY . .

RUN go build -o pie

FROM ubuntu:20.04

RUN apt-get update \
    && apt-get install -y --no-install-recommends fio jq \
    && rm -rf /var/lib/apt/lists/*

COPY --from=builder /work/pie /

EXPOSE 8080/tcp
EXPOSE 8081/tcp
EXPOSE 8082/tcp

USER 10000:10000

ENTRYPOINT [ "/pie" ]
