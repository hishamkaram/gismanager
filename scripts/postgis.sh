#!/bin/bash
export PGPASSWORD="golang"
$PWD/scripts/wait.sh -h localhost -p 5432 -t 600 -- echo "Postgis Ready!!!!"
