#!/bin/sh
export SOURCE_DATE_EPOCH=0
for f in testdata/test_full_*.txt
do
	cat "${f%.txt}.args" 2> /dev/null \
	| xargs -d'\n' sh -x -c "GOHELP2MAN_TESTCASE=$f go run . -opt-include ${f%.txt}.h2m \"\$@\" testdata/test.sh" "go" > "${f%.txt}.1"
done
