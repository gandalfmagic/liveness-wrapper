#!/bin/sh

trap '>&2 echo "INT SIGNAL"; exit 131' INT
trap 'i=1; while [ $i -le 5 ]; do sleep 0.01; i=$(( i + 1 )); done' EXIT

i=1; while [ $i -le 5 ]; do sleep 0.01; i=$(( i + 1 )); done

sleep 0.01
exit 0
