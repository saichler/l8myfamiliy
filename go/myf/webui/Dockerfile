FROM saichler/builder:latest AS build

COPY main.go /home/src/github.com/saichler/build/main.go
COPY .netrc /root/.netrc

RUN go mod init
RUN GOPROXY=direct GOPRIVATE=github.com go mod tidy
RUN go build -o fweb

FROM saichler/my-family-security:latest AS final
COPY --from=build /home/src/github.com/saichler/build/fweb /home/run/fweb
COPY ./web /home/run/web
ENTRYPOINT ["/home/run/fweb"]