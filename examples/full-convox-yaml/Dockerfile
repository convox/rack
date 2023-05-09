FROM httpd

RUN apt update > /dev/null 2>&1 && \
    apt install -y curl > /dev/null 2>&1

COPY . /usr/local/apache2

RUN chmod +x /usr/local/apache2/timer-cmd.sh
