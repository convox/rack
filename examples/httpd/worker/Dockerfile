FROM ubuntu

WORKDIR /usr/src/app
COPY . /usr/src/app

RUN chmod +x /usr/src/app/entrypoint.sh
RUN chmod +x /usr/src/app/cmd.sh

CMD ["./entrypoint.sh"]
