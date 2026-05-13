.PHONY: run build clean

run: build
	./opencola

build:
	go build -o opencola .

clean:
	rm -f opencola
