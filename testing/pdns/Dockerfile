FROM alpine

RUN apk add pdns
RUN apk add pdns-backend-pgsql
RUN apk add supervisor 
ADD run.sh .
RUN chmod a+x ./run.sh
ADD pdns.conf /etc/pdns/
ADD supervisord.conf /etc/supervisord.conf

EXPOSE 53/tcp 53/udp 80/tcp
CMD ./run.sh