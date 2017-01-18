FROM python:3.6

# install dependencies first for faster rebuilds
COPY requirements.txt /tmp/requirements.txt
RUN pip3 install --upgrade pip
RUN pip3 install -r /tmp/requirements.txt

# copy the rest of the app
COPY . /app
WORKDIR /app
RUN pip3 install .

# Replace with your own command
CMD ["echo", "Hello world! Now update this Dockerfile to run your own application."]
