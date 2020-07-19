#!/bin/sh

trap 'echo "INT SIGNAL"; exit 131' INT

i=1; while [ $i -le 5 ]; do sleep 0.01; i=$(( i + 1 )); done

>&2 echo "write a line to stderr"
exit 10
