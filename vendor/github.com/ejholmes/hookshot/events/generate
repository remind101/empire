#!/bin/bash

events="
commit_comment
create
delete
deployment
deployment_status
fork
gollum
issue_comment
issues
member
membership
page_build
ping
public
pull_request
pull_request_review_comment
push
release
repository
status
team_add
watch
"

for event in $events; do
  name=$(ruby -r 'active_support' -r 'active_support/core_ext' -e "print '${event}'.camelize")
  curl -Ls https://raw.githubusercontent.com/github/developer.github.com/master/lib/webhooks/${event}.payload.json | gojson -pkg=events -name=${name} > ${event}.go
done
