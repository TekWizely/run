module github.com/tekwizely/run

go 1.25

// To update:
//
// $ go get github.com/tekwizely/go-parsing/lexer@master
// $ go get github.com/tekwizely/go-parsing/lexer/token@master
// $ go get github.com/tekwizely/go-parsing/parser@master
//
require (
	github.com/goreleaser/fileglob v1.4.0
	github.com/subosito/gotenv v1.6.0
	github.com/tekwizely/go-parsing/lexer v0.0.0-20210910181107-ed69a13f4d15
	github.com/tekwizely/go-parsing/lexer/token v0.0.0-20210910181107-ed69a13f4d15
	github.com/tekwizely/go-parsing/parser v0.0.0-20210910181107-ed69a13f4d15
)

require (
	github.com/gobwas/glob v0.2.3 // indirect
	golang.org/x/text v0.12.0 // indirect
)
