# Use as:
#   make KEY="google_api_key"
#
# To parse the latest schede in parent directory.


sqlite.db: converter
	./converter --schede ./schede_monitoraggio_*.csv --key=$(KEY) --output_sqlite sqlite.db 2>&1 | tee converter.log

comuni.json: converter
	./converter --schede ./schede_monitoraggio_*.csv --key=$(KEY)

converter: converter.go
	go build

format:
	go fmt
