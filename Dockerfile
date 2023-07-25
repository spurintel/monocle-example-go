FROM golang:alpine AS builder

FROM scratch
COPY ./build/monocleclientserver_linux_amd64 /bin/monocleclientserver
WORKDIR /monocleclientserver
EXPOSE 8080
CMD [ "/bin/monocleclientserver" ]