module github.com/arvinsg/cess-fuse

go 1.15

require (
	github.com/aws/aws-sdk-go v1.44.146
	github.com/fagongzi/log v0.0.0-20201106014031-b41ebf3bd287 // indirect
	github.com/google/uuid v1.1.2-0.20190416172445-c2e93f3ae59f // indirect
	github.com/gopherjs/gopherjs v0.0.0-20210202160940-bed99a852dfe // indirect
	github.com/jacobsa/fuse v0.0.0-20201216155545-e0296dec955f
	github.com/kardianos/osext v0.0.0-20190222173326-2bc1f35cddc0 // indirect
	github.com/kr/pretty v0.1.1-0.20190720101428-71e7e4993750 // indirect
	github.com/kylelemons/godebug v1.1.0 // indirect
	github.com/mattn/go-ieproxy v0.0.0-20190805055040-f9202b1cfdeb // indirect
	github.com/mitchellh/go-homedir v1.1.0 // indirect
	github.com/satori/go.uuid v1.2.1-0.20181028125025-b2ce2384e17b // indirect
	github.com/sevlyar/go-daemon v0.1.5 // indirect
	github.com/shirou/gopsutil v0.0.0-20190731134726-d80c43f9c984
	github.com/sirupsen/logrus v1.4.3-0.20190807103436-de736cf91b92
	github.com/smartystreets/goconvey v1.6.1-0.20160119221636-995f5b2e021c // indirect
	github.com/urfave/cli v1.21.1-0.20190807111034-521735b7608a
	golang.org/x/sys v0.1.0
	gopkg.in/check.v1 v1.0.0-20180628173108-788fd7840127 // indirect
	gopkg.in/ini.v1 v1.46.0 // indirect
)

replace github.com/jacobsa/fuse => github.com/kahing/fusego v0.0.0-20200327063725-ca77844c7bcc
