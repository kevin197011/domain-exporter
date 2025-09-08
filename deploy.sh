#/usr/bin/env bash


docker compose down --rmi all --volumes --remove-orphans
docker compose up -d --build
