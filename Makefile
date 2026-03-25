.PHONY: build run test clean download

build:
	go build -o mail2dingtalk

run: build
	./mail2dingtalk

test:
	python3 test_email.py

download:
	go mod download
	go mod tidy

clean:
	rm -f mail2dingtalk
	rm -rf data/emails/*
	rm -rf tmp/attachments/*
	rm -f logs/*.log

install: build
	cp mail2dingtalk /usr/local/bin/
	cp config.yaml /etc/mail2dingtalk/
