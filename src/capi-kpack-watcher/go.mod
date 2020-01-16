module capi_kpack_watcher

go 1.13

require (
	code.cloudfoundry.org/clock v0.0.0-20180518195852-02e53af36e6c
	code.cloudfoundry.org/lager v2.0.0+incompatible
	code.cloudfoundry.org/trace-logger v0.0.0-20170119230301-107ef08a939d // indirect
	code.cloudfoundry.org/uaa-go-client v0.0.0-20190819190728-86bc743fdd89
	github.com/davecgh/go-spew v1.1.1
	github.com/pivotal/kpack v0.0.5
	github.com/sclevine/spec v1.3.0
	github.com/stretchr/testify v1.4.0
	github.com/tedsuo/ifrit v0.0.0-20191009134036-9a97d0632f00 // indirect
	github.com/vektra/mockery v0.0.0-20181123154057-e78b021dcbb5 // indirect
	k8s.io/api v0.0.0-20190819141258-3544db3b9e44
	k8s.io/client-go v0.0.0-20190819141724-e14f31a72a77
	knative.dev/pkg v0.0.0-20190927181044-f6eb4a55ec68
)
