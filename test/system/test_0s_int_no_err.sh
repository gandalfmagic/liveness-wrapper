#!/bin/sh

trap 'echo "INT SIGNAL"; exit 0' INT

i=1; while [ $i -le 5 ]; do sleep 0.01; echo ${i}0ms; i=$(( i + 1 )); done

sleep 0.01
exit 0
