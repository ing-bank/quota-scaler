FROM golang:1.17-buster as builder

RUN mkdir /app
RUN mkdir /app/conf

ADD . /app/
WORKDIR /app

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main github.com/ing-bank/quota-scaler/cmd/quota-scaler

FROM scratch

LABEL com.ing.name="ichp_quota_scaler" \
      com.ing.summary="ICHP Quota Autoscaler" \
      com.ing.description="Automatically scales ResourceQuotas based on QuotaAutoscaler CRD"

WORKDIR /app/
COPY --from=builder /app/main /app/main
COPY build/tls-ca-bundle.pem /etc/ssl/certs/ca-certificates.crt
USER 1001

CMD ["./main"]
