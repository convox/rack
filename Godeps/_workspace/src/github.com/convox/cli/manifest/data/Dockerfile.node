FROM node:0.10

EXPOSE 3000
ENV PORT 3000

WORKDIR /app

COPY package.json /app/package.json
RUN npm install

COPY . /app

CMD ["npm", "start"]
