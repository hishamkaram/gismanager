#!/bin/bash
export PGPASSWORD="golang"
$PWD/scripts/wait.sh -h localhost -p 5436 -t 600 -- echo "Postgis Ready!!!!"
