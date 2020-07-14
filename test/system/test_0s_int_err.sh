#!/bin/sh

trap '>&2 echo "INT SIGNAL"; exit 131' INT

i=1; while [ $i -le 10 ]; do sleep 0.01; echo ${i}0ms; i=$(( i + 1 )); done

exit 0
