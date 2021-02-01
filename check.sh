#!/bin/bash
err=0
function lint_dir() {
    for file in $(ls $1); do
        if [ -d $1"/"$file ]; then
            lint_dir "$1/$file"
        elif [ "${file##*.}" == "go" ]; then
            golint -set_exit_status=true "$1/$file" || err=$(($err + 1))
        fi
    done
}
lint_dir ./
if [ $err -gt 0 ]; then
    exit 1
fi
echo "code golint check success"
