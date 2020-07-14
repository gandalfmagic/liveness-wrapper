#!/bin/sh

trap '>&2 echo "INT SIGNAL"; exit 131' INT
trap 'i=1; while [ $i -le 10 ]; do sleep 0.01; echo EXIT ${i}0ms; i=$(( i + 1 )); done' EXIT

i=1; while [ $i -le 10 ]; do sleep 0.01; echo ${i}0ms; i=$(( i + 1 )); done

>&2 echo "write a line to stderr"
exit 10
