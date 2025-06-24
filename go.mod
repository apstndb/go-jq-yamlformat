module github.com/apstndb/go-jq-yamlformat

go 1.22

toolchain go1.24.0

require (
	github.com/apstndb/go-yamlformat v0.0.0-20241224000000-000000000000
	github.com/goccy/go-yaml v1.18.0
	github.com/google/go-cmp v0.7.0
	github.com/itchyny/gojq v0.12.16
)

require github.com/itchyny/timefmt-go v0.1.6 // indirect

replace github.com/apstndb/go-yamlformat => ../../go-yamlformat
