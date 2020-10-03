PROJECT_ROOT:=/home/isucon/webapp
BUILD_DIR:=/home/isucon/webapp/golang

BIN_NAME:=isuumo # TODO
BIN_PATH:=/home/isucon/isuumo/webapp/go/isuumo # TODO
SERVICE_NAME:=isuumo.go # TODO
APP_LOCAL_URL:=http://localhost:1323 # TODO

NGX_SERVICE=envoy # TODO
NGX_LOG:=/var/log/envoy/access.log # TODO

MYSQL_SERVICE=mysql
MYSQL_LOG:=/var/log/mysql/mysql.log

HOSTNAME:=$(shell hostname)

BRANCH:=master

all: build

.PHONY: clean
clean:
	cd $(BUILD_DIR); \
	make clean

.PHONY: deploy
deploy: before build config-files start

.PHONY: deploy-nolog
deploy-nolog: before build-nolog config-files start

.PHONY: checkout
checkout:
	git fetch && \
	git reset --hard origin/$(BRANCH)

.PHONY: build
build: checkout
	cd $(BUILD_DIR); \
	make all

.PHONY: build-nolog
build-nolog: checkout
	cd $(BUILD_DIR); \
	make all
	# TODO

.PHONY: config-files
config-files:
	sudo rsync -v -r $(HOSTNAME)/ /

.PHONY: start
start:
	sh $(HOSTNAME)/deploy.sh

.PHONY: pprof
pprof:
	pprof -png -output /tmp/pprof.png $(BIN_PATH) $(APP_LOCAL_URL)/debug/pprof/profile
	# slackcat /tmp/pprof.png
	pprof -http=0.0.0.0:9090 $(BIN_PATH) `ls -lt $(HOME)/pprof/* | head -n 1 | gawk '{print $$9}'`

.PHONY: kataru
kataru:
	sudo cat $(NGX_LOG) | kataribe -f /etc/kataribe.toml | slackcat

.PHONY: before
before:
	$(eval when := $(shell date "+%s"))
	mkdir -p ~/logs/$(when)
	sudo touch $(NGX_LOG)
	sudo touch $(MYSQL_LOG)
	sudo mv -f $(NGX_LOG) ~/logs/$(when)/
	sudo mv -f $(MYSQL_LOG) ~/logs/$(when)/
