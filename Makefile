GO111MODULE := on

tar: dist bin/icinga2rt dist/icinga2rt.json.example
	mkdir dist/bytemine-icinga2rt
	cp bin/icinga2rt dist/bytemine-icinga2rt/bytemine-icinga2rt
	cp README.md dist/bytemine-icinga2rt
	mv dist/icinga2rt.json.example dist/bytemine-icinga2rt
	mv dist/bytemine-icinga2rt "dist/bytemine-icinga2rt-`bin/icinga2rt -version`"
	cd dist && tar cvzf bytemine-icinga2rt-`../bin/icinga2rt -version`.tar.gz bytemine-icinga2rt-`../bin/icinga2rt -version`
	cd dist && rm -r bytemine-icinga2rt-`../bin/icinga2rt -version`
	sha256sum dist/bytemine-icinga2rt-`bin/icinga2rt -version`.tar.gz

bin: 
	mkdir -p bin

bin/icinga2rt: bin go.mod main.go cache.go ticket.go config.go rt/rt.go filter/filter.go
	go build -o bin/icinga2rt

test:
	go test -v

dist:
	mkdir -p dist

dist/icinga2rt.json.example: dist bin/icinga2rt
	cd dist && ../bin/icinga2rt -example
