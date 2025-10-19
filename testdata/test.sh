#!/bin/sh
while IFS= read -r line
do
	echo "$line"
done < $GOHELP2MAN_TESTCASE
