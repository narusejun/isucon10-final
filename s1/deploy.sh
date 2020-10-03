#!/bin/bash

sudo systemctl daemon-reload
sudo systemctl restart mysql
sudo systemctl restart xsuportal-web-golang.service
sudo systemctl restart xsuportal-api-golang.service
sudo systemctl restart envoy
