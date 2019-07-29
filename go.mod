module github.com/tekwizely/run-cmd

go 1.12

// To update:
//
// $ go get github.com/tekwizely/go-parsing/lexer@master
// $ go get github.com/tekwizely/go-parsing/lexer/token@master
// $ go get github.com/tekwizely/go-parsing/parser@master
//
require (
	github.com/tekwizely/go-parsing/lexer v0.0.0-20190714215300-5be83bb42370
	github.com/tekwizely/go-parsing/lexer/token v0.0.0-20190714215300-5be83bb42370
	github.com/tekwizely/go-parsing/parser v0.0.0-20190714215300-5be83bb42370
)

//replace github.com/tekwizely/go-parsing/parser => /Users/david/Documents/Dev/TekWizely/Go/go-parsing/src/github.com/tekwizely/go-parsing/parser
