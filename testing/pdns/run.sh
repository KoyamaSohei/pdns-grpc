if [[ -z "${GPGSQL_HOST}" ]];
then 
  sed -i "s/{{GPGSQL_HOST}}/127.0.0.1/" /etc/pdns/pdns.conf
else 
  sed -i "s/{{GPGSQL_HOST}}/${GPGSQL_HOST}/" /etc/pdns/pdns.conf
fi

if [[ -z "${GPGSQL_PORT}" ]];
then 
  sed -i "s/{{GPGSQL_PORT}}/5432/" /etc/pdns/pdns.conf
else 
  sed -i "s/{{GPGSQL_PORT}}/${GPGSQL_PORT}/" /etc/pdns/pdns.conf
fi

if [[ -z "${GPGSQL_DBNAME}" ]];
then 
  sed -i "s/{{GPGSQL_DBNAME}}/pdns/" /etc/pdns/pdns.conf
else 
  sed -i "s/{{GPGSQL_DBNAME}}/${GPGSQL_DBNAME}/" /etc/pdns/pdns.conf
fi

if [[ -z "${GPGSQL_USER}" ]];
then 
  sed -i "s/{{GPGSQL_USER}}/pdns/" /etc/pdns/pdns.conf
else 
  sed -i "s/{{GPGSQL_USER}}/${GPGSQL_USER}/" /etc/pdns/pdns.conf
fi

if [[ -z "${GPGSQL_PASSWORD}" ]];
then 
  sed -i "s/{{GPGSQL_PASSWORD}}/changeme/" /etc/pdns/pdns.conf
else 
  sed -i "s/{{GPGSQL_PASSWORD}}/${GPGSQL_PASSWORD}/" /etc/pdns/pdns.conf
fi

exec /usr/bin/supervisord 