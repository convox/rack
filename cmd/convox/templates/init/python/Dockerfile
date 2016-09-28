FROM python:3.6

# copy only the files needed for pip install
COPY requirements.txt /tmp/requirements.txt
RUN pip3 install --upgrade pip
RUN pip3 install -r /tmp/requirements.txt

# copy the rest of the app
COPY . /app
WORKDIR /app

CMD gunicorn --bind 0.0.0.0:5000 your_project_name.wsgi:application
