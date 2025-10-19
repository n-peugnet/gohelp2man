#!/bin/sh
export SOURCE_DATE_EPOCH=0
set -x
for f in testdata/test_full_*.txt
do
	GOHELP2MAN_TESTCASE=$f go run . -include "${f%.txt}.h2m" "testdata/test.sh" > "${f%.txt}.1"
done
