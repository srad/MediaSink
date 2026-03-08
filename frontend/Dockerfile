FROM node:22-bookworm as build-stage

RUN apt update && apt upgrade -y
RUN apt install python3 make g++

WORKDIR /app

COPY package*.json ./

# install dependencies. Timeout for buildx emulation, it is so slow that NPM install might time-out
RUN npm install -g pnpm
RUN pnpm install

COPY . .

RUN pnpm run test:unit
RUN pnpm run build

# Build time vars
ARG APP_VERSION=1.0.0-alpha
ARG APP_BUILD
ARG APP_API_VERSION

# Run time vars
ARG APP_NAME=MediaSink
ARG APP_BASE
ARG APP_API_URL
ARG APP_FILE_URL
ARG APP_SOCKET_URL

ENV APP_API_URL $APP_API_URL
ENV APP_BASE $APP_BASE
ENV APP_NAME $APP_NAME
ENV APP_FILE_URL $APP_FILE_URL
ENV APP_SOCKET_URL $APP_SOCKET_URL

RUN cat >./dist/build.js <<EOL
window.APP_BUILD = "${APP_BUILD}";
window.APP_VERSION = "${APP_VERSION}";
window.APP_API_VERSION = "${APP_API_VERSION}";
EOL

# production stage
FROM nginx:stable as production-stage

COPY --from=build-stage /app/dist /usr/share/nginx/html
COPY ./nginx.conf.default /etc/nginx/nginx.conf
# Add only if you want simple basic authentication protection to the app
#COPY .htpasswd /etc/nginx

EXPOSE 80

COPY docker-entrypoint.sh /usr/local/bin/

RUN chmod +x /usr/local/bin/docker-entrypoint.sh

ENTRYPOINT ["/usr/local/bin/docker-entrypoint.sh"]
