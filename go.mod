module github.com/arvinsg/cess-fuse

go 1.17

require (
	github.com/aws/aws-sdk-go v1.38.7
	github.com/jacobsa/fuse v0.0.0-20201216155545-e0296dec955f
	github.com/kr/pretty v0.1.1-0.20190720101428-71e7e4993750 // indirect
	github.com/kylelemons/godebug v1.1.0 // indirect
	github.com/shirou/gopsutil v0.0.0-20190731134726-d80c43f9c984
	github.com/sirupsen/logrus v1.4.3-0.20190807103436-de736cf91b92
	github.com/urfave/cli v1.21.1-0.20190807111034-521735b7608a
	golang.org/x/sys v0.0.0-20200930185726-fdedc70b468f
	gopkg.in/check.v1 v1.0.0-20180628173108-788fd7840127 // indirect
)

replace github.com/jacobsa/fuse => github.com/kahing/fusego v0.0.0-20200327063725-ca77844c7bcc
