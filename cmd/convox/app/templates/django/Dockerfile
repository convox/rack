FROM convox/django

# replace exampleapp and exampleproject to tailor the image to your project
# PROJECT is also used by gunicorn in the convox/django Dockerfile
ENV APP exampleapp
ENV PROJECT exampleproject

# an empty SECRET_KEY during manage.py commands will result in an error
ENV SECRET_KEY foo

# copy only the files needed for pip install
COPY requirements.txt /app/requirements.txt
RUN pip3 install --upgrade pip
RUN pip3 install -r requirements.txt

# copy only the files needed for collectstatic
COPY ${APP}/static /app/${APP}/static
COPY ${PROJECT} /app/${PROJECT}
COPY manage.py /app/manage.py
RUN python3 manage.py collectstatic --noinput

# copy the rest of the app
COPY . /app
