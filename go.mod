module github.com/zhaojh329/rttys

go 1.12

replace (
	golang.org/x/crypto => github.com/golang/crypto v0.0.0-20190426145343-a29dc8fdc734
	golang.org/x/net => github.com/golang/net v0.0.0-20190424112056-4829fb13d2c6
	golang.org/x/sync => github.com/golang/sync v0.0.0-20190423024810-112230192c58
	golang.org/x/sys => github.com/golang/sys v0.0.0-20190426135247-a129542de9ae
	golang.org/x/text => github.com/golang/text v0.3.2
	golang.org/x/tools => github.com/golang/tools v0.0.0-20190425222832-ad9eeb80039a
)

require (
	github.com/GehirnInc/crypt v0.0.0-20190301055215-6c0105aabd46
	github.com/gorilla/websocket v1.4.0
	github.com/howeyc/gopass v0.0.0-20170109162249-bf9dde6d0d2c
	github.com/json-iterator/go v1.1.6
	github.com/kylelemons/go-gypsy v0.0.0-20160905020020-08cad365cd28
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.1 // indirect
	github.com/rakyll/statik v0.1.6
	github.com/rifflock/lfshook v0.0.0-20180920164130-b9218ef580f5
	github.com/sirupsen/logrus v1.4.1
	golang.org/x/crypto v0.0.0-20190308221718-c2843e01d9a2 // indirect
)
