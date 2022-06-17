# build environment
FROM docker.io/golang:1.18.3-alpine3.16 AS build-env
ADD . /src
RUN cd /src && go build -mod=vendor

# run environment
FROM docker.io/alpine:3.16
WORKDIR /
COPY --from=build-env /src/airlock /usr/local/bin/airlock
CMD [ "/usr/local/bin/airlock", "serve" ]
