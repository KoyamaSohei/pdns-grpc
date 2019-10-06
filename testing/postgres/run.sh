#!/bin/bash

if [[ -z "${POSTGRES_USER}" ]];
then 
  sed -i "s/{{POSTGRES_USER}}/pdns/" /docker-entrypoint-initdb.d/01_schema.sql
else 
  sed -i "s/{{POSTGRES_USER}}/${POSTGRES_USER}/" /docker-entrypoint-initdb.d/01_schema.sql
fi

if [[ -z "${POSTGRES_PASSWORD}" ]];
then 
  sed -i "s/{{POSTGRES_PASSWORD}}/changeme/" /docker-entrypoint-initdb.d/01_schema.sql
else 
  sed -i "s/{{POSTGRES_PASSWORD}}/${POSTGRES_PASSWORD}/" /docker-entrypoint-initdb.d/01_schema.sql
fi