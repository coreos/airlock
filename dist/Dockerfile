# build environment
FROM golang:1.12-alpine3.9 AS build-env
ADD . /src
RUN cd /src && go build -mod=vendor

# run environment
FROM alpine:3.9
WORKDIR /
COPY --from=build-env /src/airlock /usr/local/bin/airlock
CMD [ "/usr/local/bin/airlock", "serve" ]
