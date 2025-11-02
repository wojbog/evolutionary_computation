#!/bin/bash

set -xe

pandoc --self-contained  -V lang=en  report.md -o report.html
