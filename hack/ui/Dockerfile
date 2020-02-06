FROM node:lts-alpine as build-stage
WORKDIR /app
COPY ./ui/ .
RUN yarn && npm run build

# production stage
FROM alpine:3.11.2
COPY /dist /app/ui