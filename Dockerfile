FROM gliderlabs/alpine:3.4
ADD bin/quietyear /go/bin/quietyear
EXPOSE 8080
ENTRYPOINT ["/go/bin/quietyear"]
