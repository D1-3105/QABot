FROM golang:1.24.4-bookworm AS GObuilder
COPY . /server
WORKDIR /server
RUN go mod tidy
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /bin/qabot main.go
RUN ls -lh /bin/qabot && file /bin/qabot && ldd /bin/qabot || true

FROM node:22-alpine AS JSbuilder
RUN corepack enable && corepack prepare yarn@stable --activate

COPY ./frontend/web-interface /frontend
WORKDIR /frontend
RUN yarn install
RUN yarn build
RUN mv dist /static

FROM debian:bookworm-slim
WORKDIR /
COPY --from=GObuilder /bin/qabot /bin/qabot
COPY --from=GObuilder /server/assets /assets
COPY --from=JSbuilder /static /static

ENTRYPOINT ["/bin/qabot", "--logtostderr=true"]
