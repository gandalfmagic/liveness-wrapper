#!/bin/sh

trap 'echo "INT SIGNAL"; exit 0' INT
trap 'sleep 0.05; echo "EXIT: wait 50ms"' EXIT

i=1; while [ $i -le 5 ]; do sleep 0.01; echo ${i}0ms; i=$(( i + 1 )); done

sleep 0.01
>&2 echo "write a line to stderr"
exit 10
