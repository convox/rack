FROM httpd

RUN apt update > /dev/null 2>&1 && \
    apt install -y curl > /dev/null 2>&1

COPY . /usr/local/apache2
