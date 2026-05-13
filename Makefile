.PHONY: run build clean

run: clean build
	./opencola

build:
	go build -o opencola .

clean:
	rm -f opencola

push:
	git add .
	git commit -m "Update" || true
	git push
