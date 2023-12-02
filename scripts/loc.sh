#!/bin/bash
LOC=`find . -type f -name "*.py" -exec cat {} + | grep -v '^ *#' | grep -v '^\s*$' | wc -l`
echo $LOC lines of code
