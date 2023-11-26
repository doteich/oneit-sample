
#Build Stage
FROM golang:1.21.4-alpine3.18 as build
WORKDIR /app
COPY . .
RUN go build -o /oneit

#Copy files from build, to slim down the overall image size
FROM alpine:3.18
WORKDIR /root/
COPY --from=build ./app ./
COPY --from=build /oneit ./
CMD ["./oneit"] 