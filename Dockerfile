FROM golang:1.15.3-alpine
ADD . /gone
WORKDIR /gone
RUN go build -o /bin/gone

FROM alpine
COPY --from=0 /bin/gone /bin/
WORKDIR /gone
EXPOSE 8080
ENTRYPOINT /bin/gone

# usage:
# docker build -t localhost:5000/gone . && docker push localhost:5000/gone


# to run a local registry:
# docker run -d -p 5000:5000 --restart=always --name registry registry:2