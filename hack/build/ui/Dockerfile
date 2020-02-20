FROM node:lts-alpine as builder
WORKDIR /app
COPY ./ui/ .
RUN yarn && yarn run build

# production stage
FROM alpine:3.11.2
COPY --from=builder /app/dist /app/ui
