#!/bin/sh

trap '>&2 echo "INT SIGNAL"; exit 131' INT
trap 'i=1; while [ $i -le 10 ]; do sleep 0.01; i=$(( i + 1 )); done' EXIT

i=1; while [ $i -le 10 ]; do sleep 0.01; i=$(( i + 1 )); done

exit 0
