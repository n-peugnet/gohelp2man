#!/bin/sh
while IFS= read -r line
do
	printf "%s\n" "$line"
done < $GOHELP2MAN_TESTCASE
