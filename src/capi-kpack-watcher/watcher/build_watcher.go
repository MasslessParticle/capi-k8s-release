package watcher

import (
	"log"
	"regexp"

	"capi_kpack_watcher/capi"
	"capi_kpack_watcher/kubernetes"
	"capi_kpack_watcher/model"

	"k8s.io/client-go/tools/cache"

	kpack "github.com/pivotal/kpack/pkg/apis/build/v1alpha1"
	kpackclient "github.com/pivotal/kpack/pkg/client/clientset/versioned"
	kpackinformer "github.com/pivotal/kpack/pkg/client/informers/externalversions"

	"github.com/davecgh/go-spew/spew"
)

const buildGUIDLabel = "cloudfoundry.org/build_guid"
const buildStagedState = "STAGED"
const buildFailedState = "FAILED"

// AddFunc handles when new Builds are detected.
func (bw *buildWatcher) AddFunc(obj interface{}) {
	build := obj.(*kpack.Build)

	log.Printf("[AddFunc] New Build: %s\n", build.GetName())
}

// UpdateFunc handles when Builds are updated.
func (bw *buildWatcher) UpdateFunc(oldobj, newobj interface{}) {
	build := newobj.(*kpack.Build)

	log.Printf(
		`[UpdateFunc] Update to Build: %s
status: %s
steps:  %+v

`, build.GetName(), spew.Sdump(build.Status.Status), build.Status.StepsCompleted)

	if bw.isBuildGUIDMissing(build) {
		return
	}

	c := build.Status.GetCondition("Succeeded")
	if c.IsTrue() {
		bw.handleSuccessfulBuild(build)
	} else if c.IsFalse() {
		bw.handleFailedBuild(build)
	} // c.isUnknown() is also available for pending builds
}

// NewBuildWatcher initializes a Watcher that watches for Builds in Kpack.
func NewBuildWatcher(c kpackclient.Interface) Watcher {
	factory := kpackinformer.NewSharedInformerFactory(c, 0)

	bw := &buildWatcher{
		client:     capi.NewCAPIClient(),
		kubeClient: kubernetes.NewInClusterClient(),
		informer:   factory.Build().V1alpha1().Builds().Informer(),
	}

	// TODO: ignore added builds at watcher startup
	bw.informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    bw.AddFunc,
		UpdateFunc: bw.UpdateFunc,
	})

	return bw
}

// Run runs the informer and begins watching for Builds. This can be stopped by
// sending to the stopped channel.
func (bw *buildWatcher) Run() {
	stopper := make(chan struct{})
	defer close(stopper)

	bw.informer.Run(stopper)
}

type kubeClient interface {
	GetContainerLogs(podName, containerName string) ([]byte, error)
}

type buildWatcher struct {
	client capi.CAPI // The watcher uses this client to talk to CAPI.

	// The watcher uses this kubernetes client to talk to the Kubernetes master.
	kubeClient kubeClient

	// Below are Kubernetes-internal objects for creating Kubernetes Informers.
	// They are in this struct to abstract away the Informer boilerplate.
	informer cache.SharedIndexInformer
}

func (bw *buildWatcher) isBuildGUIDMissing(build *kpack.Build) bool {
	labels := build.GetLabels()
	if labels == nil {
		return true
	} else if _, ok := labels[buildGUIDLabel]; !ok {
		return true
	}

	return false
}

func (bw *buildWatcher) handleSuccessfulBuild(build *kpack.Build) {
	labels := build.GetLabels()
	guid := labels[buildGUIDLabel]

	model := model.BuildStatus{
		State: buildStagedState,
	}

	if err := bw.client.UpdateBuild(guid, model); err != nil {
		log.Fatalf("[UpdateFunc] Failed to send request: %v\n", err)
	}
}

func (bw *buildWatcher) handleFailedBuild(build *kpack.Build) {
	labels := build.GetLabels()
	guid := labels[buildGUIDLabel]
	model := model.BuildStatus{
		State: buildFailedState,
	}

	status := build.Status

	// Retrieve the last container's logs. In kpack, the steps correspond
	// to container names, so we want the last container's logs.
	container := status.StepsCompleted[len(status.StepsCompleted)-1]

	logs, err := bw.kubeClient.GetContainerLogs(status.PodName, container)
	if err != nil {
		log.Printf("[UpdateFunc] Failed to get pod logs: %v\n", err)

		model.Error = "Kpack build failed"
	} else {
		// Take the first word character to the end of the line to avoid ANSI color codes
		regex := regexp.MustCompile(`ERROR:[^\w\[]*(\[[0-9]+m)?(\w[^\n]*)`)
		model.Error = string(regex.FindSubmatch(logs)[2])
	}

	if err := bw.client.UpdateBuild(guid, model); err != nil {
		log.Fatalf("[UpdateFunc] Failed to send request: %v\n", err)
	}
}
