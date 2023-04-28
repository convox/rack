#!/bin/bash

# Check if the apps resources are available within the app
ps=$(convox api get /apps/ci2/processes | jq -r '.[]|select(.status=="running")|.id' | head -n 1)

# postgres resource
convox exec -a ci2 $ps -- env | grep "POSTGRES_URL"
convox exec -a ci2 $ps -- env | grep "POSTGRES_USER"
convox exec -a ci2 $ps -- env | grep "POSTGRES_PASS"
convox exec -a ci2 $ps -- env | grep "POSTGRES_HOST"
convox exec -a ci2 $ps -- env | grep "POSTGRES_PORT"
convox exec -a ci2 $ps -- env | grep "POSTGRES_NAME"
# mysql resource
convox exec -a ci2 $ps -- env | grep "MYSQL_URL"
convox exec -a ci2 $ps -- env | grep "MYSQL_USER"
convox exec -a ci2 $ps -- env | grep "MYSQL_PASS"
convox exec -a ci2 $ps -- env | grep "MYSQL_HOST"
convox exec -a ci2 $ps -- env | grep "MYSQL_PORT"
convox exec -a ci2 $ps -- env | grep "MYSQL_NAME"
# mariadb resource
convox exec -a ci2 $ps -- env | grep "MARIADB_URL"
convox exec -a ci2 $ps -- env | grep "MARIADB_USER"
convox exec -a ci2 $ps -- env | grep "MARIADB_PASS"
convox exec -a ci2 $ps -- env | grep "MARIADB_HOST"
convox exec -a ci2 $ps -- env | grep "MARIADB_PORT"
convox exec -a ci2 $ps -- env | grep "MARIADB_NAME"
