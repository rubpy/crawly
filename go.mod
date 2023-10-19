module github.com/rubpy/crawly

go 1.21

replace (
	github.com/rubpy/crawly/cclient => ./cclient
	github.com/rubpy/crawly/clog => ./clog
	github.com/rubpy/crawly/csync => ./csync
)

require (
	github.com/rubpy/crawly/clog v0.0.0-00010101000000-000000000000
	github.com/rubpy/crawly/csync v0.0.0-00010101000000-000000000000
)
