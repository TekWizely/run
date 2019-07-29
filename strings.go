package main

func defaultIfEmpty(src string, def string) string {
	if len(src) > 0 {
		return src
	}
	return def
}
