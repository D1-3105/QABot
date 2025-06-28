FROM golang:1.24.4-bookworm AS GObuilder
COPY . /server
WORKDIR /server
RUN go mod tidy
RUN go build -o /bin/qabot main.go

FROM node:22-alpine AS JSbuilder
RUN corepack enable && corepack prepare yarn@stable --activate

COPY ./frontend/web-interface /frontend
WORKDIR /frontend
RUN yarn install
RUN yarn build
RUN mv dist /static

FROM gcr.io/distroless/static:nonroot
COPY --from=GObuilder /bin/qabot /bin/qabot
COPY --from=GObuilder /server/assets /assets
COPY --from=JSbuilder /static /static

ENTRYPOINT ["/bin/qabot", "--logtostderr=true"]
