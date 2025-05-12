# Stage 1: Build dependencies
FROM node:16-alpine AS builder
WORKDIR /app
COPY package*.json ./
RUN npm install

# Stage 2: Compile assets
FROM node:16-alpine AS assets
WORKDIR /assets
COPY --from=builder /app/node_modules ./node_modules
COPY public ./public
COPY webpack.config.js ./webpack.config.js
RUN npx webpack --config webpack.config.js

# Stage 3: Final stage
FROM node:16-alpine
ARG BUILD_VERSION=dev
ENV BUILD_VERSION=${BUILD_VERSION}

WORKDIR /app
COPY --from=builder /app/node_modules ./node_modules
COPY --from=assets /assets/build ./public/build
COPY . .

EXPOSE 3000
CMD ["node", "server.js"]