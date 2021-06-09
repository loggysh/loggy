FROM ubuntu:latest

COPY test.db /db

RUN apt-get -y update
RUN apt-get -y upgrade
RUN apt-get install -y sqlite3 libsqlite3-dev


RUN /usr/bin/sqlite3 /db/test.db
CMD /bin/bash





