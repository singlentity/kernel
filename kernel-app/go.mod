module kernel

go 1.24.7

require (
	github.com/jchv/go-webview2 v0.0.0-00010101000000-000000000000
	golang.org/x/sys v0.39.0
)

require github.com/jchv/go-winloader v0.0.0-20250406163304-c1995be93bd1 // indirect

replace (
	github.com/jchv/go-webview2 => ../deps/go-webview2
	github.com/jchv/go-winloader => ../deps/go-winloader
	golang.org/x/sys => ../deps/sys
)
