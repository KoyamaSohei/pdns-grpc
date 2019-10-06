FROM postgres:10

WORKDIR /docker-entrypoint-initdb.d
RUN chmod a+rwx /docker-entrypoint-initdb.d
COPY 01_schema.sql /docker-entrypoint-initdb.d
ADD run.sh .
RUN chmod a+x ./run.sh
EXPOSE 5432
RUN ./run.sh
