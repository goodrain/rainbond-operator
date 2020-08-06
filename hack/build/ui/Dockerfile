FROM rainbond/node:lts-alpine as builder
WORKDIR /app
COPY ./ui/ .
RUN cnpm install && yarn run build

# production stage
FROM alpine:3.11.2
COPY --from=builder /app/dist /app/ui
